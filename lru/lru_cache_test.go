package lru

import (
	"context"
	"testing"
	"time"
)

func TestMemCache_SetGet(t *testing.T) {
	ctx := context.Background()
	c := NewMemCache[string, int](MemOptions[string, int]{Capacity: 10})

	if err := c.Set(ctx, "a", 1); err != nil {
		t.Fatal(err)
	}
	v, ok, err := c.Get(ctx, "a")
	if err != nil || !ok || v != 1 {
		t.Fatalf("Get(a) = %v, %v, %v", v, ok, err)
	}
}

func TestMemCache_Eviction(t *testing.T) {
	ctx := context.Background()
	evicted := map[string]int{}
	c := NewMemCache[string, int](MemOptions[string, int]{
		Capacity: 2,
		OnEvict:  func(k string, v int) { evicted[k] = v },
	})

	_ = c.Set(ctx, "a", 1)
	_ = c.Set(ctx, "b", 2)
	_ = c.Set(ctx, "c", 3) // should evict "a"

	if _, ok, _ := c.Get(ctx, "a"); ok {
		t.Error("expected a to be evicted")
	}
	if _, ok := evicted["a"]; !ok {
		t.Errorf("OnEvict missed a; got %v", evicted)
	}
	if n, _ := c.Len(ctx); n != 2 {
		t.Errorf("Len = %d, want 2", n)
	}
}

func TestMemCache_LRUOrder(t *testing.T) {
	ctx := context.Background()
	c := NewMemCache[string, int](MemOptions[string, int]{Capacity: 2})

	_ = c.Set(ctx, "a", 1)
	_ = c.Set(ctx, "b", 2)
	// Touch a, so b becomes LRU.
	_, _, _ = c.Get(ctx, "a")
	_ = c.Set(ctx, "c", 3)

	if _, ok, _ := c.Get(ctx, "b"); ok {
		t.Error("expected b to be evicted (least-recently-used)")
	}
	if _, ok, _ := c.Get(ctx, "a"); !ok {
		t.Error("a should still be present")
	}
}

func TestMemCache_TTL(t *testing.T) {
	ctx := context.Background()
	c := NewMemCache[string, int](MemOptions[string, int]{Capacity: 10, TTL: 20 * time.Millisecond})
	_ = c.Set(ctx, "a", 1)

	if _, ok, _ := c.Get(ctx, "a"); !ok {
		t.Fatal("should be fresh")
	}
	time.Sleep(40 * time.Millisecond)
	if _, ok, _ := c.Get(ctx, "a"); ok {
		t.Error("expected a to be expired")
	}
}

func TestMemCache_Peek(t *testing.T) {
	ctx := context.Background()
	c := NewMemCache[string, int](MemOptions[string, int]{Capacity: 2})
	_ = c.Set(ctx, "a", 1)
	_ = c.Set(ctx, "b", 2)

	// Peek must NOT promote.
	if _, _, _ = c.Peek(ctx, "a"); false {
	}
	_ = c.Set(ctx, "c", 3) // evicts LRU; with Peek not touching, "a" is LRU and should go

	if _, ok, _ := c.Get(ctx, "a"); ok {
		t.Error("Peek should not promote — a should have been evicted")
	}
}

func TestMemCache_Delete(t *testing.T) {
	ctx := context.Background()
	c := NewMemCache[string, int](MemOptions[string, int]{Capacity: 10})
	_ = c.Set(ctx, "a", 1)
	_ = c.Set(ctx, "b", 2)

	_ = c.Delete(ctx, "a", "nonexistent")
	if _, ok, _ := c.Get(ctx, "a"); ok {
		t.Error("a should be deleted")
	}
	if n, _ := c.Len(ctx); n != 1 {
		t.Errorf("Len = %d, want 1", n)
	}
}

func TestMemCache_Range(t *testing.T) {
	ctx := context.Background()
	c := NewMemCache[string, int](MemOptions[string, int]{Capacity: 10})
	for i, k := range []string{"a", "b", "c"} {
		_ = c.Set(ctx, k, i)
	}

	seen := 0
	_ = c.Range(ctx, func(k string, v int) bool { seen++; return true })
	if seen != 3 {
		t.Errorf("Range visited %d entries, want 3", seen)
	}

	// Early-exit semantics.
	seen = 0
	_ = c.Range(ctx, func(k string, v int) bool { seen++; return false })
	if seen != 1 {
		t.Errorf("Range with early exit visited %d, want 1", seen)
	}
}

func TestMemCache_Clear(t *testing.T) {
	ctx := context.Background()
	cleared := 0
	c := NewMemCache[string, int](MemOptions[string, int]{
		Capacity: 10,
		OnEvict:  func(string, int) { cleared++ },
	})
	_ = c.Set(ctx, "a", 1)
	_ = c.Set(ctx, "b", 2)

	_ = c.Clear(ctx)
	if n, _ := c.Len(ctx); n != 0 {
		t.Errorf("Len after Clear = %d, want 0", n)
	}
	if cleared != 2 {
		t.Errorf("OnEvict fired %d times, want 2", cleared)
	}
}

func TestMemCache_PurgeExpired(t *testing.T) {
	ctx := context.Background()
	c := NewMemCache[string, int](MemOptions[string, int]{Capacity: 10, TTL: 20 * time.Millisecond})
	_ = c.Set(ctx, "a", 1)
	_ = c.Set(ctx, "b", 2)

	time.Sleep(40 * time.Millisecond)
	_ = c.PurgeExpired(ctx)
	if n, _ := c.Len(ctx); n != 0 {
		t.Errorf("Len after PurgeExpired = %d, want 0", n)
	}
}

func TestMemCache_Unbounded(t *testing.T) {
	ctx := context.Background()
	c := NewMemCache[string, int](MemOptions[string, int]{}) // Capacity = 0
	for i := 0; i < 1000; i++ {
		_ = c.Set(ctx, key(i), i)
	}
	if n, _ := c.Len(ctx); n != 1000 {
		t.Errorf("Len = %d, want 1000", n)
	}
}

func key(i int) string {
	return string(rune('a'+(i%26))) + string(rune('0'+(i/26%10))) + string(rune('0'+(i/260%10)))
}
