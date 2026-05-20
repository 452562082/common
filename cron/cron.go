// Package cron is a thin wrapper around robfig/cron/v3 with three additions
// over the library's defaults:
//
//   - Jobs are ctx-aware: they receive a context that is cancelled when the
//     Scheduler stops, so long-running jobs can shut down cleanly.
//   - Panics in jobs are recovered and logged, never killing the scheduler.
//   - Each invocation is wrapped in a slog line carrying job name + elapsed,
//     so you get out-of-the-box visibility.
//
// Cron expressions: by default this package accepts 5-field expressions
// (minute hour dom month dow). To use the seconds-prefixed syntax, set
// Options.WithSeconds = true.
package cron

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	rcron "github.com/robfig/cron/v3"
)

// JobFunc is the signature for scheduled work. Return an error for log
// visibility; the scheduler will keep running regardless.
type JobFunc func(ctx context.Context) error

// Options configures NewScheduler.
type Options struct {
	// Location pins the cron expressions to a timezone. Default time.Local.
	Location *time.Location

	// WithSeconds parses 6-field expressions (with a leading seconds field).
	WithSeconds bool

	// Logger is used for job lifecycle events. nil = slog.Default().
	Logger *slog.Logger
}

// Scheduler manages a set of cron jobs.
type Scheduler struct {
	cron *rcron.Cron
	log  *slog.Logger

	mu     sync.Mutex
	cancel context.CancelFunc
	ctx    context.Context
	jobs   map[rcron.EntryID]string
	running atomic.Bool
}

// NewScheduler returns a Scheduler. Use AddFunc to register jobs, then Start.
func NewScheduler(opts Options) *Scheduler {
	loc := opts.Location
	if loc == nil {
		loc = time.Local
	}
	log := opts.Logger
	if log == nil {
		log = slog.Default()
	}

	cronOpts := []rcron.Option{rcron.WithLocation(loc)}
	if opts.WithSeconds {
		cronOpts = append(cronOpts, rcron.WithSeconds())
	}
	return &Scheduler{
		cron: rcron.New(cronOpts...),
		log:  log,
		jobs: make(map[rcron.EntryID]string),
	}
}

// AddFunc registers fn under spec. name is used in log lines.
// Returns the entry ID so callers can Remove the job later.
func (s *Scheduler) AddFunc(spec, name string, fn JobFunc) (rcron.EntryID, error) {
	if name == "" {
		name = spec
	}
	id, err := s.cron.AddFunc(spec, func() { s.runOne(name, fn) })
	if err != nil {
		return 0, fmt.Errorf("cron: add %s: %w", spec, err)
	}
	s.mu.Lock()
	s.jobs[id] = name
	s.mu.Unlock()
	return id, nil
}

// Remove unregisters a previously-added job. Removing an unknown ID is a no-op.
func (s *Scheduler) Remove(id rcron.EntryID) {
	s.cron.Remove(id)
	s.mu.Lock()
	delete(s.jobs, id)
	s.mu.Unlock()
}

// Entries exposes scheduled entries, primarily for diagnostics.
func (s *Scheduler) Entries() []rcron.Entry { return s.cron.Entries() }

// Start the scheduler. Idempotent. The provided context is propagated to
// every JobFunc invocation; cancelling it stops new runs from being scheduled
// and signals in-flight jobs to wind down.
//
// Designed to plug into graceful.App.Add as the run function (and Stop as the
// close function).
func (s *Scheduler) Start(ctx context.Context) error {
	if !s.running.CompareAndSwap(false, true) {
		// Already running; block until ctx done so graceful sees the typical "Run blocks" contract.
		<-ctx.Done()
		return nil
	}
	s.mu.Lock()
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.mu.Unlock()
	s.cron.Start()
	s.log.InfoContext(ctx, "cron: scheduler started", "jobs", len(s.jobs))
	<-s.ctx.Done()
	return nil
}

// Stop halts the scheduler and waits for any in-flight job to finish.
// Safe to call multiple times.
func (s *Scheduler) Stop(ctx context.Context) error {
	if !s.running.CompareAndSwap(true, false) {
		return nil
	}
	s.mu.Lock()
	cancel := s.cancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}

	stopCtx := s.cron.Stop()
	select {
	case <-stopCtx.Done():
		return nil
	case <-ctx.Done():
		return fmt.Errorf("cron: stop ctx: %w", ctx.Err())
	}
}

func (s *Scheduler) runOne(name string, fn JobFunc) {
	s.mu.Lock()
	ctx := s.ctx
	s.mu.Unlock()
	if ctx == nil {
		ctx = context.Background()
	}

	start := time.Now()
	defer func() {
		if r := recover(); r != nil {
			s.log.ErrorContext(ctx, "cron: job panic",
				"name", name,
				"err", r,
				"stack", string(debug.Stack()),
			)
		}
	}()
	err := fn(ctx)
	level := slog.LevelInfo
	if err != nil {
		level = slog.LevelError
	}
	s.log.LogAttrs(ctx, level, "cron: job ran",
		slog.String("name", name),
		slog.Duration("elapsed", time.Since(start)),
		slog.Any("err", err),
	)
}
