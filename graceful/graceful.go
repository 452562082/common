// Package graceful orchestrates the start and shutdown of long-running
// in-process components.
//
// A program typically does:
//
//	app := graceful.New(graceful.Options{ShutdownTimeout: 30 * time.Second})
//
//	app.Add("kafka-consumer", func(ctx context.Context) error { return consumer.Run(ctx) }, consumer.Close)
//	app.Add("http-server",    server.Run, server.Shutdown)
//
//	if err := app.Run(context.Background()); err != nil {
//	    log.Fatal(err)
//	}
//
// Run blocks until SIGINT/SIGTERM arrives or any component returns. Then it
// shuts every component down in reverse-add order with a hard deadline. The
// idea is "start in dependency order, stop in reverse" — same as defer.
package graceful

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// RunFunc is the long-running body of a component. It should return nil on
// graceful shutdown (typically when ctx is cancelled) or a non-nil error if
// the component crashed.
type RunFunc func(ctx context.Context) error

// CloseFunc is invoked during shutdown. It is given an outer ctx with the
// configured shutdown deadline. CloseFunc should be idempotent.
type CloseFunc func(ctx context.Context) error

type component struct {
	name  string
	run   RunFunc
	close CloseFunc
}

// Options tunes App behaviour.
type Options struct {
	// ShutdownTimeout is the hard deadline applied to the close phase.
	// Zero falls back to 15s.
	ShutdownTimeout time.Duration

	// Signals listened to. Empty falls back to SIGINT + SIGTERM.
	Signals []os.Signal

	// Logger is used for lifecycle events. nil = slog.Default().
	Logger *slog.Logger
}

// App is the orchestrator.
type App struct {
	opts       Options
	log        *slog.Logger
	mu         sync.Mutex
	components []component
}

// New returns an App configured with opts.
func New(opts Options) *App {
	if opts.ShutdownTimeout == 0 {
		opts.ShutdownTimeout = 15 * time.Second
	}
	if len(opts.Signals) == 0 {
		opts.Signals = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	}
	log := opts.Logger
	if log == nil {
		log = slog.Default()
	}
	return &App{opts: opts, log: log}
}

// Add registers a component. close may be nil if the run function handles its
// own cleanup on ctx cancellation. Components are stopped in reverse order.
func (a *App) Add(name string, run RunFunc, close CloseFunc) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.components = append(a.components, component{name: name, run: run, close: close})
}

// Run starts every component, waits for the first of:
//
//   - a signal in Options.Signals,
//   - any component's Run returning,
//   - the supplied ctx being cancelled,
//
// then shuts every component down in reverse order with a bounded deadline.
// Returns the first error encountered, if any.
func (a *App) Run(ctx context.Context) error {
	a.mu.Lock()
	comps := append([]component(nil), a.components...)
	a.mu.Unlock()

	if len(comps) == 0 {
		return errors.New("graceful: no components registered")
	}

	runCtx, cancelRun := context.WithCancel(ctx)
	defer cancelRun()

	type result struct {
		name string
		err  error
	}
	results := make(chan result, len(comps))

	for _, c := range comps {
		// Go 1.22+ each loop iteration has its own scope; no `c := c` needed.
		go func() {
			a.log.Info("graceful: component starting", "name", c.name)
			err := safeRun(runCtx, c.run)
			a.log.Info("graceful: component returned", "name", c.name, "err", err)
			results <- result{name: c.name, err: err}
		}()
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, a.opts.Signals...)
	defer signal.Stop(sigCh)

	var firstErr error
	drained := 0
	select {
	case sig := <-sigCh:
		a.log.Info("graceful: signal received, shutting down", "signal", sig.String())
	case <-ctx.Done():
		a.log.Info("graceful: context cancelled, shutting down")
	case r := <-results:
		drained = 1 // consumed one result via this branch
		if r.err != nil {
			a.log.Error("graceful: component crashed", "name", r.name, "err", r.err)
			firstErr = fmt.Errorf("graceful: component %s: %w", r.name, r.err)
		} else {
			a.log.Info("graceful: component exited, shutting down", "name", r.name)
		}
	}

	cancelRun()

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), a.opts.ShutdownTimeout)
	defer cancelShutdown()
	closeErrs := a.closeAll(shutdownCtx, comps)

	// Drain any remaining run-goroutines (best effort, bounded by ShutdownTimeout).
	for drained < len(comps) {
		select {
		case r := <-results:
			drained++
			if r.err != nil && !errors.Is(r.err, context.Canceled) && firstErr == nil {
				firstErr = fmt.Errorf("graceful: component %s: %w", r.name, r.err)
			}
		case <-shutdownCtx.Done():
			a.log.Warn("graceful: shutdown timed out waiting for components", "remaining", len(comps)-drained)
			drained = len(comps)
		}
	}

	if firstErr != nil {
		return firstErr
	}
	if len(closeErrs) > 0 {
		return errors.Join(closeErrs...)
	}
	return nil
}

func (a *App) closeAll(ctx context.Context, comps []component) []error {
	var errs []error
	for i := len(comps) - 1; i >= 0; i-- {
		c := comps[i]
		if c.close == nil {
			continue
		}
		a.log.Info("graceful: closing component", "name", c.name)
		if err := safeClose(ctx, c.close); err != nil {
			a.log.Error("graceful: close failed", "name", c.name, "err", err)
			errs = append(errs, fmt.Errorf("graceful: close %s: %w", c.name, err))
		}
	}
	return errs
}

func safeRun(ctx context.Context, fn RunFunc) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in component: %v", r)
		}
	}()
	return fn(ctx)
}

func safeClose(ctx context.Context, fn CloseFunc) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in close: %v", r)
		}
	}()
	return fn(ctx)
}
