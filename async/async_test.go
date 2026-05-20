package async

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestPool_SubmitAndDrain(t *testing.T) {
	p := NewPool(3, 10)
	defer p.Close()

	var n atomic.Int32
	for i := 0; i < 20; i++ {
		_ = p.Submit(context.Background(), func(_ context.Context) {
			n.Add(1)
		})
	}
	p.Close()
	if got := n.Load(); got != 20 {
		t.Errorf("ran %d tasks, want 20", got)
	}
}

func TestPool_TrySubmit_FullQueue(t *testing.T) {
	p := NewPool(1, 1)

	// Tie up the only worker, but signal once it actually starts running.
	started := make(chan struct{})
	block := make(chan struct{})
	if err := p.TrySubmit(func(_ context.Context) {
		close(started)
		<-block
	}); err != nil {
		t.Fatal(err)
	}
	<-started // worker is now blocked; the queue is empty.

	// Fill the queue.
	if err := p.TrySubmit(func(_ context.Context) {}); err != nil {
		t.Fatalf("queue should have one slot free: %v", err)
	}
	// Next must fail with busy.
	err := p.TrySubmit(func(_ context.Context) {})
	if !errors.Is(err, ErrPoolBusy) {
		t.Errorf("expected ErrPoolBusy, got %v", err)
	}
	close(block) // release the worker so Close can drain
	p.Close()
}

func TestPool_ClosedReturnsErr(t *testing.T) {
	p := NewPool(1, 1)
	p.Close()
	err := p.Submit(context.Background(), func(_ context.Context) {})
	if !errors.Is(err, ErrPoolClosed) {
		t.Errorf("expected ErrPoolClosed, got %v", err)
	}
}

func TestPool_PanicDoesNotKillWorker(t *testing.T) {
	p := NewPool(1, 1)
	defer p.Close()

	var ran atomic.Int32
	_ = p.Submit(context.Background(), func(_ context.Context) { panic("boom") })
	_ = p.Submit(context.Background(), func(_ context.Context) { ran.Add(1) })
	p.Close()
	if ran.Load() != 1 {
		t.Error("worker did not survive a panic")
	}
}

func TestRetry_EventualSuccess(t *testing.T) {
	var calls atomic.Int32
	err := Retry(context.Background(), RetryOptions{Attempts: 5, Min: time.Millisecond, Max: 5 * time.Millisecond},
		func(_ context.Context) error {
			n := calls.Add(1)
			if n < 3 {
				return errors.New("transient")
			}
			return nil
		})
	if err != nil {
		t.Fatalf("Retry: %v", err)
	}
	if calls.Load() != 3 {
		t.Errorf("expected 3 calls, got %d", calls.Load())
	}
}

func TestRetry_Exhausted(t *testing.T) {
	sentinel := errors.New("nope")
	err := Retry(context.Background(), RetryOptions{Attempts: 3, Min: time.Millisecond, Max: 2 * time.Millisecond},
		func(_ context.Context) error { return sentinel })
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel wrapped, got %v", err)
	}
}

func TestRetry_ShouldRetryStops(t *testing.T) {
	sentinel := errors.New("fatal")
	var calls atomic.Int32
	err := Retry(context.Background(), RetryOptions{
		Attempts:    5,
		Min:         time.Microsecond,
		ShouldRetry: func(err error) bool { return !errors.Is(err, sentinel) },
	}, func(_ context.Context) error {
		calls.Add(1)
		return sentinel
	})
	if !errors.Is(err, sentinel) || calls.Load() != 1 {
		t.Errorf("expected single attempt, got %d (err=%v)", calls.Load(), err)
	}
}

func TestRetry_CtxCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()
	err := Retry(ctx, RetryOptions{Attempts: 10, Min: 20 * time.Millisecond, Max: 50 * time.Millisecond},
		func(_ context.Context) error { return errors.New("nope") })
	if err == nil {
		t.Fatal("expected ctx-cancel error")
	}
}

func TestGroup_Alias(t *testing.T) {
	var g Group
	v, _, _ := g.Do("k", func() (any, error) { return 42, nil })
	if v.(int) != 42 {
		t.Errorf("got %v", v)
	}
}
