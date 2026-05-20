// Package limiter provides per-key rate limiting.
//
// Two implementations are shipped:
//
//   - TokenBucket — wraps golang.org/x/time/rate with a per-key map and a
//     background sweeper that evicts idle keys so RAM does not grow without
//     bound under high cardinality.
//   - SlidingWindow — a lightweight in-process sliding-window counter, useful
//     when you want hard "N per minute" semantics without the smoothing that
//     token buckets give you.
//
// Both implement the Limiter interface:
//
//	type Limiter interface {
//	    Allow(key string) bool
//	    Wait(ctx context.Context, key string) error
//	}
package limiter

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Limiter is the small interface satisfied by every implementation in this package.
type Limiter interface {
	// Allow reports whether the operation keyed by key may proceed right now.
	Allow(key string) bool

	// Wait blocks until the operation may proceed or ctx is done.
	Wait(ctx context.Context, key string) error
}

// --- TokenBucket -------------------------------------------------------------

// TokenBucketOptions configures NewTokenBucket.
type TokenBucketOptions struct {
	// Rate is the steady-state allowed rate, in operations per second.
	Rate float64

	// Burst is the bucket size, i.e. the max number of operations allowed
	// to arrive at once after a quiet period. Zero falls back to ceil(Rate).
	Burst int

	// IdleEvict, when > 0, controls how long an unused per-key bucket is
	// kept before being garbage-collected. Default 10 minutes.
	IdleEvict time.Duration

	// MaxKeys caps the number of distinct keys held in memory. When the
	// cap is hit, the least-recently-used bucket is evicted to make room.
	// 0 means "no limit" (the historical behaviour). Set this whenever the
	// key derives from untrusted input (IP, header value, ...) to bound
	// worst-case memory.
	MaxKeys int
}

// TokenBucket is a per-key token bucket limiter.
type TokenBucket struct {
	opts TokenBucketOptions
	mu   sync.Mutex
	buckets map[string]*bucket

	stop chan struct{}
	done chan struct{}
}

type bucket struct {
	rl   *rate.Limiter
	last time.Time
}

// NewTokenBucket builds a TokenBucket and starts its idle-eviction goroutine.
// Call Close to stop the goroutine when you're done with the limiter.
func NewTokenBucket(opts TokenBucketOptions) *TokenBucket {
	if opts.Rate <= 0 {
		opts.Rate = 1
	}
	if opts.Burst <= 0 {
		opts.Burst = max(1, int(opts.Rate))
	}
	if opts.IdleEvict <= 0 {
		opts.IdleEvict = 10 * time.Minute
	}

	tb := &TokenBucket{
		opts:    opts,
		buckets: make(map[string]*bucket),
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
	go tb.evictLoop()
	return tb
}

// Allow reports whether one token is available for key right now.
func (tb *TokenBucket) Allow(key string) bool {
	b := tb.getOrCreate(key)
	return b.rl.Allow()
}

// Wait blocks until a token is available or ctx is done.
func (tb *TokenBucket) Wait(ctx context.Context, key string) error {
	b := tb.getOrCreate(key)
	if err := b.rl.Wait(ctx); err != nil {
		return fmt.Errorf("limiter: wait: %w", err)
	}
	return nil
}

// Reset removes the bucket for key, so the next Allow / Wait starts fresh.
func (tb *TokenBucket) Reset(key string) {
	tb.mu.Lock()
	delete(tb.buckets, key)
	tb.mu.Unlock()
}

// Close stops the background eviction goroutine. Safe to call multiple times.
// Returns nil; the error return mirrors io.Closer for stack symmetry.
func (tb *TokenBucket) Close() error {
	select {
	case <-tb.stop:
		return nil
	default:
		close(tb.stop)
		<-tb.done
		return nil
	}
}

func (tb *TokenBucket) getOrCreate(key string) *bucket {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	if b, ok := tb.buckets[key]; ok {
		b.last = time.Now()
		return b
	}
	if tb.opts.MaxKeys > 0 && len(tb.buckets) >= tb.opts.MaxKeys {
		tb.evictLRULocked()
	}
	b := &bucket{
		rl:   rate.NewLimiter(rate.Limit(tb.opts.Rate), tb.opts.Burst),
		last: time.Now(),
	}
	tb.buckets[key] = b
	return b
}

func (tb *TokenBucket) evictLoop() {
	defer close(tb.done)
	t := time.NewTicker(tb.opts.IdleEvict / 2)
	defer t.Stop()
	for {
		select {
		case <-tb.stop:
			return
		case <-t.C:
			cutoff := time.Now().Add(-tb.opts.IdleEvict)
			tb.mu.Lock()
			for k, b := range tb.buckets {
				if b.last.Before(cutoff) {
					delete(tb.buckets, k)
				}
			}
			tb.mu.Unlock()
		}
	}
}

// evictLRULocked drops the single oldest bucket. Caller must hold tb.mu.
func (tb *TokenBucket) evictLRULocked() {
	var oldestKey string
	var oldestTime time.Time
	for k, b := range tb.buckets {
		if oldestKey == "" || b.last.Before(oldestTime) {
			oldestKey = k
			oldestTime = b.last
		}
	}
	if oldestKey != "" {
		delete(tb.buckets, oldestKey)
	}
}

// --- SlidingWindow -----------------------------------------------------------

// SlidingWindow is a per-key in-memory sliding-window counter. Useful for
// fixed budgets ("N requests per window") with no smoothing.
type SlidingWindow struct {
	limit   int
	window  time.Duration
	maxKeys int

	mu       sync.Mutex
	buckets  map[string]*swBucket

	stop chan struct{}
	done chan struct{}
}

type swBucket struct {
	stamps []time.Time
	last   time.Time // for LRU eviction
}

// NewSlidingWindow returns a SlidingWindow allowing limit operations per window.
// maxKeys may be passed via SetMaxKeys.
func NewSlidingWindow(limit int, window time.Duration) *SlidingWindow {
	sw := &SlidingWindow{
		limit:   limit,
		window:  window,
		buckets: make(map[string]*swBucket),
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
	go sw.evictLoop()
	return sw
}

// SetMaxKeys caps the number of distinct keys held in memory. Excess keys
// are evicted in LRU order. Pass 0 to disable the cap. Safe to call at any time.
func (sw *SlidingWindow) SetMaxKeys(n int) {
	sw.mu.Lock()
	sw.maxKeys = n
	sw.mu.Unlock()
}

// Allow returns true if the operation fits in the window.
func (sw *SlidingWindow) Allow(key string) bool {
	now := time.Now()
	cutoff := now.Add(-sw.window)

	sw.mu.Lock()
	defer sw.mu.Unlock()
	b, ok := sw.buckets[key]
	if !ok {
		if sw.maxKeys > 0 && len(sw.buckets) >= sw.maxKeys {
			sw.evictLRULocked()
		}
		// Cap the initial slice capacity. For a "1M req/min" limiter, we don't
		// want to pre-allocate 1M *time.Time per key — that's ~24 MB per key.
		// Most keys see only a handful of requests; let the slice grow as needed.
		b = &swBucket{stamps: make([]time.Time, 0, min(sw.limit+1, 64))}
		sw.buckets[key] = b
	}
	// Drop everything older than cutoff.
	trimmed := b.stamps[:0]
	for _, t := range b.stamps {
		if t.After(cutoff) {
			trimmed = append(trimmed, t)
		}
	}
	b.stamps = trimmed
	b.last = now
	if len(b.stamps) >= sw.limit {
		return false
	}
	b.stamps = append(b.stamps, now)
	return true
}

func (sw *SlidingWindow) evictLRULocked() {
	var oldestKey string
	var oldestTime time.Time
	for k, b := range sw.buckets {
		if oldestKey == "" || b.last.Before(oldestTime) {
			oldestKey = k
			oldestTime = b.last
		}
	}
	if oldestKey != "" {
		delete(sw.buckets, oldestKey)
	}
}

// Wait blocks until a slot is available or ctx is done.
//
// It uses a short polling loop; SlidingWindow is best suited to relatively
// loose limits (limits in the dozens or hundreds per minute). For high-rate
// scenarios prefer TokenBucket.
func (sw *SlidingWindow) Wait(ctx context.Context, key string) error {
	for {
		if sw.Allow(key) {
			return nil
		}
		t := time.NewTimer(50 * time.Millisecond)
		select {
		case <-ctx.Done():
			t.Stop()
			return fmt.Errorf("limiter: wait: %w", ctx.Err())
		case <-t.C:
		}
	}
}

// Close stops the eviction goroutine. Safe to call multiple times.
// Returns nil; the error return mirrors io.Closer.
func (sw *SlidingWindow) Close() error {
	select {
	case <-sw.stop:
		return nil
	default:
		close(sw.stop)
		<-sw.done
		return nil
	}
}

func (sw *SlidingWindow) evictLoop() {
	defer close(sw.done)
	t := time.NewTicker(sw.window)
	defer t.Stop()
	for {
		select {
		case <-sw.stop:
			return
		case <-t.C:
			cutoff := time.Now().Add(-sw.window)
			sw.mu.Lock()
			for k, b := range sw.buckets {
				keep := false
				for _, ts := range b.stamps {
					if ts.After(cutoff) {
						keep = true
						break
					}
				}
				if !keep {
					delete(sw.buckets, k)
				}
			}
			sw.mu.Unlock()
		}
	}
}

// ErrClosed is reserved for future use when a limiter rejects calls because
// it has been shut down. Currently both limiters tolerate post-Close use.
var ErrClosed = errors.New("limiter: closed")

// Compile-time interface assertions.
var (
	_ Limiter = (*TokenBucket)(nil)
	_ Limiter = (*SlidingWindow)(nil)
)
