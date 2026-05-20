package breaker

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDo_PassesThroughWhenClosed(t *testing.T) {
	b := New(Options{Name: "test"})
	called := false
	err := b.Do(context.Background(), func(ctx context.Context) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if !called {
		t.Error("fn was not called")
	}
	if b.State() != StateClosed {
		t.Errorf("state = %v", b.State())
	}
}

func TestDo_TripsAfterFailures(t *testing.T) {
	b := New(Options{
		Name:             "trip-me",
		FailureThreshold: 3,
		Timeout:          time.Second,
	})
	boom := errors.New("boom")
	for i := 0; i < 3; i++ {
		_ = b.Do(context.Background(), func(_ context.Context) error { return boom })
	}
	if b.State() != StateOpen {
		t.Fatalf("expected open after 3 failures, got %v", b.State())
	}

	// Subsequent call must short-circuit.
	called := false
	err := b.Do(context.Background(), func(_ context.Context) error {
		called = true
		return nil
	})
	if !errors.Is(err, ErrOpen) {
		t.Errorf("expected ErrOpen, got %v", err)
	}
	if called {
		t.Error("fn should NOT be called when breaker is open")
	}
}

func TestDo_HalfOpenRecovery(t *testing.T) {
	b := New(Options{
		Name:             "recover",
		FailureThreshold: 2,
		Timeout:          30 * time.Millisecond,
		MaxRequests:      1,
	})
	boom := errors.New("boom")
	for i := 0; i < 2; i++ {
		_ = b.Do(context.Background(), func(_ context.Context) error { return boom })
	}
	if b.State() != StateOpen {
		t.Fatal("breaker not open")
	}

	// Wait for half-open, then a successful call should close it.
	time.Sleep(40 * time.Millisecond)

	err := b.Do(context.Background(), func(_ context.Context) error { return nil })
	if err != nil {
		t.Fatalf("expected success during half-open, got %v", err)
	}
	if b.State() != StateClosed {
		t.Errorf("expected closed after probe success, got %v", b.State())
	}
}

func TestOnStateChange(t *testing.T) {
	transitions := []State{}
	b := New(Options{
		Name:             "trace",
		FailureThreshold: 1,
		Timeout:          20 * time.Millisecond,
		OnStateChange:    func(from, to State) { transitions = append(transitions, to) },
	})
	_ = b.Do(context.Background(), func(_ context.Context) error { return errors.New("x") })
	// Now Open. Wait for half-open transition.
	time.Sleep(30 * time.Millisecond)
	// Probe and succeed → Closed.
	_ = b.Do(context.Background(), func(_ context.Context) error { return nil })

	if len(transitions) < 2 {
		t.Fatalf("expected at least 2 transitions, got %v", transitions)
	}
	if transitions[0] != StateOpen {
		t.Errorf("first transition should be StateOpen, got %v", transitions[0])
	}
}

func TestDoValue(t *testing.T) {
	b := New(Options{Name: "gen"})
	v, err := DoValue(context.Background(), b, func(_ context.Context) (int, error) {
		return 42, nil
	})
	if err != nil || v != 42 {
		t.Errorf("DoValue → %d, %v", v, err)
	}

	// Trip the breaker, then verify DoValue short-circuits.
	b2 := New(Options{Name: "gen2", FailureThreshold: 1, Timeout: time.Second})
	boom := errors.New("boom")
	_, _ = DoValue(context.Background(), b2, func(_ context.Context) (int, error) { return 0, boom })

	called := false
	_, err = DoValue(context.Background(), b2, func(_ context.Context) (int, error) {
		called = true
		return 1, nil
	})
	if !errors.Is(err, ErrOpen) {
		t.Errorf("expected ErrOpen, got %v", err)
	}
	if called {
		t.Error("fn should not be invoked when breaker open")
	}
}

func TestSnapshot(t *testing.T) {
	b := New(Options{Name: "snap"})
	_ = b.Do(context.Background(), func(_ context.Context) error { return nil })
	c := b.Snapshot()
	if c.Requests == 0 {
		t.Error("Snapshot.Requests should be > 0")
	}
}

func TestStateString(t *testing.T) {
	if StateClosed.String() == "" || StateOpen.String() == "" || StateHalfOpen.String() == "" {
		t.Error("State.String() should be non-empty")
	}
}
