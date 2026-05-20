package distlock

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func setup(t *testing.T) (*Locker, *miniredis.Miniredis, *redis.Client) {
	t.Helper()
	s := miniredis.RunT(t)
	c := redis.NewClient(&redis.Options{Addr: s.Addr()})
	t.Cleanup(func() { _ = c.Close() })
	return New(c, Options{TTL: 200 * time.Millisecond, PollInterval: 10 * time.Millisecond}), s, c
}

func TestTryLock_Acquires(t *testing.T) {
	l, _, _ := setup(t)
	ctx := context.Background()

	lock, err := l.TryLock(ctx, "resource")
	if err != nil {
		t.Fatalf("TryLock: %v", err)
	}
	if lock.Token() == "" {
		t.Error("empty token")
	}
	if err := lock.Unlock(ctx); err != nil {
		t.Errorf("Unlock: %v", err)
	}
}

func TestTryLock_ContendedFails(t *testing.T) {
	l, _, _ := setup(t)
	ctx := context.Background()

	a, err := l.TryLock(ctx, "k")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.TryLock(ctx, "k"); !errors.Is(err, ErrNotAcquired) {
		t.Errorf("expected ErrNotAcquired, got %v", err)
	}
	_ = a.Unlock(ctx)
}

func TestUnlock_DoesNotDropOthersLock(t *testing.T) {
	l, s, _ := setup(t)
	ctx := context.Background()

	a, err := l.TryLock(ctx, "k")
	if err != nil {
		t.Fatal(err)
	}
	// Forcibly expire and let someone else claim it.
	s.FastForward(time.Second)

	b, err := l.TryLock(ctx, "k")
	if err != nil {
		t.Fatalf("second TryLock: %v", err)
	}

	// 'a' should fail unlock — the value is now b's token.
	if err := a.Unlock(ctx); !errors.Is(err, ErrNotHeld) {
		t.Errorf("a.Unlock should return ErrNotHeld, got %v", err)
	}
	// 'b' is still ours.
	if err := b.Unlock(ctx); err != nil {
		t.Errorf("b.Unlock: %v", err)
	}
}

func TestUnlock_IdempotentBeforeReturn(t *testing.T) {
	l, _, _ := setup(t)
	ctx := context.Background()

	a, _ := l.TryLock(ctx, "k")
	if err := a.Unlock(ctx); err != nil {
		t.Errorf("first Unlock: %v", err)
	}
	// Second call must not crash.
	if err := a.Unlock(ctx); err != nil && !errors.Is(err, ErrNotHeld) {
		t.Errorf("second Unlock returned %v", err)
	}
}

func TestLock_WaitsAndAcquires(t *testing.T) {
	l, _, _ := setup(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	a, err := l.TryLock(ctx, "k")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = a.Unlock(context.Background())
	}()

	b, err := l.Lock(ctx, "k")
	if err != nil {
		t.Fatalf("Lock: %v", err)
	}
	_ = b.Unlock(ctx)
}

func TestLock_CtxCancel(t *testing.T) {
	l, _, _ := setup(t)

	a, _ := l.TryLock(context.Background(), "k")
	defer a.Unlock(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if _, err := l.Lock(ctx, "k"); err == nil {
		t.Error("expected ctx-cancel error")
	}
}

func TestRefresh_ExtendsTTL(t *testing.T) {
	l, s, _ := setup(t)
	ctx := context.Background()
	lk, _ := l.TryLock(ctx, "k")
	defer lk.Unlock(ctx)

	// Burn most of the TTL.
	s.FastForward(150 * time.Millisecond)
	if err := lk.Refresh(ctx); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	// Now another 150ms should still leave the lock alive.
	s.FastForward(150 * time.Millisecond)
	if _, err := l.TryLock(ctx, "k"); !errors.Is(err, ErrNotAcquired) {
		t.Errorf("lock should still be held; got %v", err)
	}
}

func TestAutoRefresh(t *testing.T) {
	s := miniredis.RunT(t)
	c := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer c.Close()

	l := New(c, Options{
		TTL:          80 * time.Millisecond,
		AutoRefresh:  30 * time.Millisecond,
		PollInterval: 10 * time.Millisecond,
	})

	lk, err := l.TryLock(context.Background(), "renew-me")
	if err != nil {
		t.Fatal(err)
	}

	// Hold longer than the original TTL — auto-refresh must keep it alive.
	time.Sleep(150 * time.Millisecond)

	if _, err := l.TryLock(context.Background(), "renew-me"); !errors.Is(err, ErrNotAcquired) {
		t.Errorf("lock should still be held by auto-refresh; got %v", err)
	}
	_ = lk.Unlock(context.Background())
}

func TestMutualExclusion(t *testing.T) {
	l, _, _ := setup(t)

	const N = 20
	var counter atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lk, err := l.Lock(context.Background(), "shared")
			if err != nil {
				t.Errorf("Lock: %v", err)
				return
			}
			// Tiny "critical section".
			old := counter.Load()
			time.Sleep(2 * time.Millisecond)
			counter.Store(old + 1)
			_ = lk.Unlock(context.Background())
		}()
	}
	wg.Wait()
	if counter.Load() != N {
		t.Errorf("non-exclusive: counter = %d, want %d", counter.Load(), N)
	}
}
