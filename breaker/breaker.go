// Package breaker is a thin, ergonomic wrapper around sony/gobreaker — the
// canonical Go implementation of the circuit-breaker pattern.
//
// A circuit breaker sits in front of an external dependency and trips open
// after consecutive failures, short-circuiting subsequent calls so the
// failing dependency gets time to recover instead of being hammered.
//
// Typical use:
//
//	b := breaker.New(breaker.Options{
//	    Name:               "payments-api",
//	    FailureThreshold:   5,
//	    Timeout:            10 * time.Second,
//	    OnStateChange:      func(from, to breaker.State) { metrics.Inc(...) },
//	})
//
//	err := b.Do(ctx, func(ctx context.Context) error {
//	    return paymentsClient.Charge(ctx, req)
//	})
//	if errors.Is(err, breaker.ErrOpen) {
//	    // tripped — fall back
//	}
package breaker

import (
	"context"
	"errors"
	"fmt"
	"time"

	gobreaker "github.com/sony/gobreaker/v2"
)

// State mirrors gobreaker.State so callers don't have to import it.
type State int

const (
	StateClosed   State = State(gobreaker.StateClosed)
	StateHalfOpen State = State(gobreaker.StateHalfOpen)
	StateOpen     State = State(gobreaker.StateOpen)
)

func (s State) String() string { return gobreaker.State(s).String() }

// Options configures New.
type Options struct {
	// Name labels the breaker in logs and metrics. Required.
	Name string

	// FailureThreshold is the number of consecutive failures that trip the
	// breaker. Default 5. Ignored if ShouldTrip is set.
	FailureThreshold uint32

	// Timeout is how long the breaker stays open before transitioning to
	// half-open. Default 60s.
	Timeout time.Duration

	// MaxRequests is the cap on concurrent requests allowed through during
	// the half-open state. Default 1.
	MaxRequests uint32

	// Interval is how often gobreaker resets its counts while in the closed
	// state. Default 0 (counts never reset until the breaker trips).
	Interval time.Duration

	// ShouldTrip lets callers replace the default consecutive-failure rule
	// with a custom one (e.g. error rate over a window).
	ShouldTrip func(counts gobreaker.Counts) bool

	// OnStateChange, if set, is called whenever the breaker transitions.
	OnStateChange func(from, to State)

	// IsSuccessful, if set, decides whether an error counts as a failure.
	// Default: any non-nil error is a failure.
	IsSuccessful func(err error) bool
}

// Breaker wraps a *gobreaker.CircuitBreaker[any] with a ctx-aware Do helper
// and an exported state-change/sentinel-error surface.
type Breaker struct {
	cb *gobreaker.CircuitBreaker[any]
}

// New builds a Breaker.
func New(opts Options) *Breaker {
	if opts.Name == "" {
		opts.Name = "breaker"
	}
	if opts.FailureThreshold == 0 {
		opts.FailureThreshold = 5
	}
	if opts.Timeout == 0 {
		opts.Timeout = 60 * time.Second
	}
	if opts.MaxRequests == 0 {
		opts.MaxRequests = 1
	}

	settings := gobreaker.Settings{
		Name:        opts.Name,
		MaxRequests: opts.MaxRequests,
		Timeout:     opts.Timeout,
		Interval:    opts.Interval,
	}
	if opts.ShouldTrip != nil {
		settings.ReadyToTrip = opts.ShouldTrip
	} else {
		thresh := opts.FailureThreshold
		settings.ReadyToTrip = func(c gobreaker.Counts) bool {
			return c.ConsecutiveFailures >= thresh
		}
	}
	if opts.OnStateChange != nil {
		settings.OnStateChange = func(name string, from, to gobreaker.State) {
			opts.OnStateChange(State(from), State(to))
		}
	}
	if opts.IsSuccessful != nil {
		settings.IsSuccessful = opts.IsSuccessful
	}

	return &Breaker{cb: gobreaker.NewCircuitBreaker[any](settings)}
}

// ErrOpen is returned when the breaker is open and a call is short-circuited.
var ErrOpen = errors.New("breaker: open")

// ErrTooManyRequests is returned when the breaker is half-open and the
// concurrency cap was reached.
var ErrTooManyRequests = errors.New("breaker: too many requests (half-open)")

// Do invokes fn under the breaker's protection.
//
// When the breaker is open, fn is NOT invoked and ErrOpen is returned.
// When it's half-open and a probe is already in flight, ErrTooManyRequests
// is returned.
func (b *Breaker) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	_, err := b.cb.Execute(func() (any, error) {
		return nil, fn(ctx)
	})
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, gobreaker.ErrOpenState):
		return ErrOpen
	case errors.Is(err, gobreaker.ErrTooManyRequests):
		return ErrTooManyRequests
	default:
		return fmt.Errorf("breaker: %w", err)
	}
}

// DoValue is the generic version of Do. Use when fn returns a value.
//
// Note: this is a free function rather than a method on Breaker because Go
// does not allow methods to introduce new type parameters.
func DoValue[T any](ctx context.Context, b *Breaker, fn func(ctx context.Context) (T, error)) (T, error) {
	var zero T
	v, err := b.cb.Execute(func() (any, error) {
		return fn(ctx)
	})
	if err == nil {
		t, _ := v.(T)
		return t, nil
	}
	switch {
	case errors.Is(err, gobreaker.ErrOpenState):
		return zero, ErrOpen
	case errors.Is(err, gobreaker.ErrTooManyRequests):
		return zero, ErrTooManyRequests
	default:
		return zero, fmt.Errorf("breaker: %w", err)
	}
}

// State reports the current breaker state.
func (b *Breaker) State() State { return State(b.cb.State()) }

// Name returns the breaker's configured name.
func (b *Breaker) Name() string { return b.cb.Name() }

// Counts returns the underlying statistics struct. Useful for metrics
// exporters.
type Counts = gobreaker.Counts

// Snapshot returns a copy of the breaker's request statistics.
func (b *Breaker) Snapshot() Counts { return b.cb.Counts() }
