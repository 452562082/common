// Package async bundles small concurrency primitives that show up in nearly
// every service:
//
//   - Pool — bounded worker pool with a context-aware Submit.
//   - Retry — operation runner with exponential backoff + jitter.
//   - SingleFlight / Group — re-export of golang.org/x/sync/singleflight for
//     de-duplicating in-flight calls.
package async

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// --- Pool ---------------------------------------------------------------------

// Task is a unit of work submitted to a Pool.
type Task func(ctx context.Context)

// Pool is a fixed-size worker pool. Submit blocks until a worker becomes free
// (or the supplied context is cancelled).
type Pool struct {
	workers int
	tasks   chan Task

	mu        sync.Mutex
	closed    bool
	closeOnce sync.Once
	wg        sync.WaitGroup
}

// NewPool returns a Pool with `workers` background goroutines and a queue
// `bufferSize` deep. workers must be >= 1.
func NewPool(workers, bufferSize int) *Pool {
	if workers < 1 {
		workers = 1
	}
	if bufferSize < 0 {
		bufferSize = 0
	}
	p := &Pool{
		workers: workers,
		tasks:   make(chan Task, bufferSize),
	}
	p.wg.Add(workers)
	for i := 0; i < workers; i++ {
		go p.run()
	}
	return p
}

func (p *Pool) run() {
	defer p.wg.Done()
	for t := range p.tasks {
		runTask(context.Background(), t)
	}
}

func runTask(ctx context.Context, t Task) {
	defer func() {
		// Panics in user code must not kill the worker; just swallow them.
		_ = recover()
	}()
	t(ctx)
}

// Submit enqueues t. If the queue is full it blocks until either a worker
// frees a slot or ctx is cancelled. After Close, Submit returns
// ErrPoolClosed.
func (p *Pool) Submit(ctx context.Context, t Task) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return ErrPoolClosed
	}
	p.mu.Unlock()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case p.tasks <- t:
		return nil
	}
}

// TrySubmit is the non-blocking variant. Returns ErrPoolBusy if the queue is full.
func (p *Pool) TrySubmit(t Task) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return ErrPoolClosed
	}
	p.mu.Unlock()
	select {
	case p.tasks <- t:
		return nil
	default:
		return ErrPoolBusy
	}
}

// Close stops accepting new tasks, waits for the queue to drain, then waits
// for all workers to exit.
func (p *Pool) Close() {
	p.closeOnce.Do(func() {
		p.mu.Lock()
		p.closed = true
		close(p.tasks)
		p.mu.Unlock()
		p.wg.Wait()
	})
}

// ErrPoolClosed is returned by Submit / TrySubmit after Close.
var ErrPoolClosed = errors.New("async: pool is closed")

// ErrPoolBusy is returned by TrySubmit when the queue is full.
var ErrPoolBusy = errors.New("async: pool queue is full")

// --- Retry --------------------------------------------------------------------

// RetryOptions configures Retry.
type RetryOptions struct {
	// Attempts is the total number of attempts (initial + retries).
	// Zero or negative falls back to 3.
	Attempts int

	// Min / Max bracket the backoff window. Defaults: 100ms / 5s.
	Min time.Duration
	Max time.Duration

	// Multiplier is the exponential base. Default 2.
	Multiplier float64

	// ShouldRetry, if set, decides whether err is worth retrying.
	// Default: retry every non-nil error except context.Canceled / DeadlineExceeded.
	ShouldRetry func(err error) bool
}

// Retry runs fn until it returns nil, ctx is cancelled, or attempts are
// exhausted. Backoff is exponential with full jitter.
func Retry(ctx context.Context, opts RetryOptions, fn func(ctx context.Context) error) error {
	applyRetryDefaults(&opts)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	var lastErr error
	for attempt := 1; attempt <= opts.Attempts; attempt++ {
		err := fn(ctx)
		if err == nil {
			return nil
		}
		lastErr = err
		if !opts.ShouldRetry(err) {
			return err
		}
		if attempt == opts.Attempts {
			break
		}
		wait := backoff(rng, opts, attempt)
		t := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			t.Stop()
			return fmt.Errorf("async: retry interrupted (%w) after error: %w", ctx.Err(), lastErr)
		case <-t.C:
		}
	}
	return fmt.Errorf("async: gave up after %d attempts: %w", opts.Attempts, lastErr)
}

func applyRetryDefaults(o *RetryOptions) {
	if o.Attempts <= 0 {
		o.Attempts = 3
	}
	if o.Min <= 0 {
		o.Min = 100 * time.Millisecond
	}
	if o.Max <= 0 {
		o.Max = 5 * time.Second
	}
	if o.Multiplier <= 1 {
		o.Multiplier = 2
	}
	if o.ShouldRetry == nil {
		o.ShouldRetry = defaultShouldRetry
	}
}

func defaultShouldRetry(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return true
}

func backoff(rng *rand.Rand, opts RetryOptions, attempt int) time.Duration {
	expo := float64(opts.Min) * pow(opts.Multiplier, attempt-1)
	if expo > float64(opts.Max) {
		expo = float64(opts.Max)
	}
	jitter := rng.Float64() * (expo - float64(opts.Min))
	if jitter < 0 {
		jitter = 0
	}
	return opts.Min + time.Duration(jitter)
}

func pow(base float64, exp int) float64 {
	r := 1.0
	for i := 0; i < exp; i++ {
		r *= base
	}
	return r
}

// --- SingleFlight -------------------------------------------------------------

// Group is an alias for x/sync/singleflight.Group: it lets multiple callers
// share the result of an in-flight call keyed by a string.
type Group = singleflight.Group

// Result mirrors singleflight.Result.
type Result = singleflight.Result
