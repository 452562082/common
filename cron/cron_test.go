package cron

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"
)

func quietOpts() Options {
	return Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

func TestScheduler_FiresJob(t *testing.T) {
	s := NewScheduler(Options{
		WithSeconds: true,
		Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	var ran atomic.Int32
	if _, err := s.AddFunc("* * * * * *", "tick", func(ctx context.Context) error {
		ran.Add(1)
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := s.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if ran.Load() < 1 {
		t.Errorf("job didn't run; count=%d", ran.Load())
	}
}

func TestScheduler_PanicDoesNotKillScheduler(t *testing.T) {
	s := NewScheduler(Options{
		WithSeconds: true,
		Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	var ran atomic.Int32
	_, _ = s.AddFunc("* * * * * *", "panicky", func(ctx context.Context) error {
		ran.Add(1)
		if ran.Load() == 1 {
			panic("boom")
		}
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
	defer cancel()
	_ = s.Start(ctx)
	_ = s.Stop(context.Background())

	if ran.Load() < 2 {
		t.Errorf("scheduler died after panic; count=%d", ran.Load())
	}
}

func TestScheduler_StopIdempotent(t *testing.T) {
	s := NewScheduler(quietOpts())
	_, _ = s.AddFunc("@every 5s", "noop", func(context.Context) error { return nil })

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_ = s.Start(ctx)
	if err := s.Stop(context.Background()); err != nil {
		t.Errorf("first Stop: %v", err)
	}
	if err := s.Stop(context.Background()); err != nil {
		t.Errorf("second Stop: %v", err)
	}
}

func TestScheduler_RemoveJob(t *testing.T) {
	s := NewScheduler(quietOpts())
	id, _ := s.AddFunc("@every 5s", "job", func(context.Context) error { return nil })
	if got := len(s.Entries()); got != 1 {
		t.Errorf("Entries = %d", got)
	}
	s.Remove(id)
	if got := len(s.Entries()); got != 0 {
		t.Errorf("Entries after Remove = %d", got)
	}
}

func TestScheduler_BadSpec(t *testing.T) {
	s := NewScheduler(quietOpts())
	if _, err := s.AddFunc("not-a-cron", "x", func(context.Context) error { return nil }); err == nil {
		t.Error("expected error for invalid spec")
	}
}

func TestScheduler_JobErrorDoesNotKill(t *testing.T) {
	s := NewScheduler(Options{
		WithSeconds: true,
		Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	var ran atomic.Int32
	_, _ = s.AddFunc("* * * * * *", "fails", func(context.Context) error {
		ran.Add(1)
		return errors.New("expected")
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
	defer cancel()
	_ = s.Start(ctx)
	_ = s.Stop(context.Background())
	if ran.Load() < 2 {
		t.Errorf("error-returning job killed scheduler; count=%d", ran.Load())
	}
}
