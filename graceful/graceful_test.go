package graceful

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func quietApp(opts Options) *App {
	if opts.Logger == nil {
		opts.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return New(opts)
}

func TestRun_ShutdownOnContextCancel(t *testing.T) {
	app := quietApp(Options{ShutdownTimeout: time.Second})

	var stopped atomic.Int32
	app.Add("svc",
		func(ctx context.Context) error {
			<-ctx.Done()
			return nil
		},
		func(ctx context.Context) error {
			stopped.Add(1)
			return nil
		},
	)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	if err := app.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if stopped.Load() != 1 {
		t.Errorf("close not invoked")
	}
}

func TestRun_StopOnComponentExit(t *testing.T) {
	app := quietApp(Options{ShutdownTimeout: time.Second})

	app.Add("short-lived",
		func(ctx context.Context) error {
			time.Sleep(20 * time.Millisecond)
			return nil
		},
		nil,
	)
	var otherClosed atomic.Bool
	app.Add("watcher",
		func(ctx context.Context) error {
			<-ctx.Done()
			return nil
		},
		func(ctx context.Context) error {
			otherClosed.Store(true)
			return nil
		},
	)

	if err := app.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !otherClosed.Load() {
		t.Errorf("watcher.Close not invoked")
	}
}

func TestRun_ComponentError(t *testing.T) {
	app := quietApp(Options{ShutdownTimeout: time.Second})
	sentinel := errors.New("boom")

	app.Add("crashy", func(ctx context.Context) error {
		return sentinel
	}, nil)
	app.Add("other", func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	}, nil)

	err := app.Run(context.Background())
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel, got %v", err)
	}
}

func TestRun_ReverseCloseOrder(t *testing.T) {
	app := quietApp(Options{ShutdownTimeout: time.Second})

	var mu sync.Mutex
	var order []string
	mk := func(name string) (RunFunc, CloseFunc) {
		return func(ctx context.Context) error {
				<-ctx.Done()
				return nil
			},
			func(ctx context.Context) error {
				mu.Lock()
				order = append(order, name)
				mu.Unlock()
				return nil
			}
	}
	r, c := mk("a")
	app.Add("a", r, c)
	r, c = mk("b")
	app.Add("b", r, c)
	r, c = mk("c")
	app.Add("c", r, c)

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(10 * time.Millisecond); cancel() }()
	_ = app.Run(ctx)

	mu.Lock()
	defer mu.Unlock()
	if len(order) != 3 || order[0] != "c" || order[1] != "b" || order[2] != "a" {
		t.Errorf("expected reverse order [c b a], got %v", order)
	}
}

func TestRun_PanicInComponent(t *testing.T) {
	app := quietApp(Options{ShutdownTimeout: time.Second})

	app.Add("panicky", func(ctx context.Context) error {
		panic("oops")
	}, nil)
	app.Add("other", func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	}, nil)

	err := app.Run(context.Background())
	if err == nil {
		t.Fatal("expected error from panic")
	}
}

func TestRun_NoComponents(t *testing.T) {
	app := quietApp(Options{})
	if err := app.Run(context.Background()); err == nil {
		t.Error("expected error when no components")
	}
}
