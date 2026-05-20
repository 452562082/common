package limiter

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestTokenBucket_AllowExhausts(t *testing.T) {
	tb := NewTokenBucket(TokenBucketOptions{Rate: 10, Burst: 3})
	defer tb.Close()

	allowed := 0
	for i := 0; i < 10; i++ {
		if tb.Allow("k") {
			allowed++
		}
	}
	// Burst is 3, so the first 3 should pass and the rest should fail
	// because the refill rate (10/s) hasn't produced new tokens yet.
	if allowed != 3 {
		t.Errorf("allowed=%d, expected 3", allowed)
	}
}

func TestTokenBucket_PerKeyIsolated(t *testing.T) {
	tb := NewTokenBucket(TokenBucketOptions{Rate: 1, Burst: 1})
	defer tb.Close()

	if !tb.Allow("a") {
		t.Error("a first call should pass")
	}
	if !tb.Allow("b") {
		t.Error("b first call should pass — keys are isolated")
	}
	if tb.Allow("a") {
		t.Error("a second call should be denied")
	}
}

func TestTokenBucket_Wait(t *testing.T) {
	tb := NewTokenBucket(TokenBucketOptions{Rate: 100, Burst: 1}) // ~10ms per token
	defer tb.Close()

	_ = tb.Allow("k") // consume burst

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	if err := tb.Wait(ctx, "k"); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if time.Since(start) < 5*time.Millisecond {
		t.Errorf("Wait returned too quickly: %v", time.Since(start))
	}
}

func TestTokenBucket_WaitCtxCancel(t *testing.T) {
	tb := NewTokenBucket(TokenBucketOptions{Rate: 0.1, Burst: 1}) // 1 token every 10s
	defer tb.Close()
	_ = tb.Allow("k")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if err := tb.Wait(ctx, "k"); err == nil {
		t.Error("expected ctx-deadline error")
	}
}

func TestTokenBucket_Reset(t *testing.T) {
	tb := NewTokenBucket(TokenBucketOptions{Rate: 1, Burst: 1})
	defer tb.Close()
	_ = tb.Allow("k")
	if tb.Allow("k") {
		t.Error("second call should be denied")
	}
	tb.Reset("k")
	if !tb.Allow("k") {
		t.Error("after Reset the first call should pass again")
	}
}

func TestTokenBucket_IdleEviction(t *testing.T) {
	tb := NewTokenBucket(TokenBucketOptions{Rate: 1, Burst: 1, IdleEvict: 30 * time.Millisecond})
	defer tb.Close()
	_ = tb.Allow("k")

	// Wait past the eviction window. The bucket should be GC'd, so the next
	// Allow gets a fresh bucket with a full burst.
	time.Sleep(60 * time.Millisecond)
	if !tb.Allow("k") {
		t.Error("expected fresh bucket after eviction")
	}
}

func TestSlidingWindow_AllowDeny(t *testing.T) {
	sw := NewSlidingWindow(3, time.Second)
	defer sw.Close()
	for i := 0; i < 3; i++ {
		if !sw.Allow("k") {
			t.Errorf("Allow #%d should pass", i+1)
		}
	}
	if sw.Allow("k") {
		t.Error("4th Allow should be denied")
	}
}

func TestSlidingWindow_WindowAdvance(t *testing.T) {
	sw := NewSlidingWindow(2, 50*time.Millisecond)
	defer sw.Close()
	_ = sw.Allow("k")
	_ = sw.Allow("k")
	if sw.Allow("k") {
		t.Fatal("3rd call should be denied")
	}
	time.Sleep(70 * time.Millisecond)
	if !sw.Allow("k") {
		t.Error("after window advance, should pass again")
	}
}

func TestSlidingWindow_Wait(t *testing.T) {
	sw := NewSlidingWindow(1, 30*time.Millisecond)
	defer sw.Close()
	_ = sw.Allow("k")
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	if err := sw.Wait(ctx, "k"); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if d := time.Since(start); d < 20*time.Millisecond {
		t.Errorf("returned too fast: %v", d)
	}
}

func TestTokenBucket_Concurrent(t *testing.T) {
	tb := NewTokenBucket(TokenBucketOptions{Rate: 1000, Burst: 100})
	defer tb.Close()

	const N = 200
	var wg sync.WaitGroup
	allowed := 0
	var mu sync.Mutex
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if tb.Allow("k") {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	// We should have allowed about Burst (100), possibly slightly more if any
	// refills occurred during the test.
	if allowed < 90 || allowed > 130 {
		t.Errorf("allowed=%d, expected ~100", allowed)
	}
}

func TestTokenBucket_MaxKeysEvictsLRU(t *testing.T) {
	tb := NewTokenBucket(TokenBucketOptions{Rate: 1, Burst: 1, MaxKeys: 3})
	defer tb.Close()

	// Fill the cap.
	for _, k := range []string{"a", "b", "c"} {
		_ = tb.Allow(k)
	}
	// Touch "a" so it becomes the most recent.
	time.Sleep(2 * time.Millisecond)
	_ = tb.Allow("a")

	// Add a fourth — should evict "b" (the LRU).
	_ = tb.Allow("d")

	tb.mu.Lock()
	_, hasA := tb.buckets["a"]
	_, hasB := tb.buckets["b"]
	_, hasC := tb.buckets["c"]
	_, hasD := tb.buckets["d"]
	n := len(tb.buckets)
	tb.mu.Unlock()

	if n != 3 {
		t.Errorf("bucket count = %d, want 3", n)
	}
	if !hasA || hasB || !hasC || !hasD {
		t.Errorf("eviction picked wrong key: a=%v b=%v c=%v d=%v", hasA, hasB, hasC, hasD)
	}
}

func TestSlidingWindow_MaxKeysEvictsLRU(t *testing.T) {
	sw := NewSlidingWindow(1, time.Minute)
	defer sw.Close()
	sw.SetMaxKeys(2)

	_ = sw.Allow("a")
	time.Sleep(time.Millisecond)
	_ = sw.Allow("b")
	time.Sleep(time.Millisecond)
	_ = sw.Allow("c") // should evict "a"

	sw.mu.Lock()
	_, hasA := sw.buckets["a"]
	_, hasB := sw.buckets["b"]
	_, hasC := sw.buckets["c"]
	n := len(sw.buckets)
	sw.mu.Unlock()

	if n != 2 || hasA || !hasB || !hasC {
		t.Errorf("sliding-window eviction wrong: a=%v b=%v c=%v n=%d", hasA, hasB, hasC, n)
	}
}

func TestInterfaceAssertions(t *testing.T) {
	var _ Limiter = NewTokenBucket(TokenBucketOptions{Rate: 1})
	var _ Limiter = NewSlidingWindow(1, time.Second)
}
