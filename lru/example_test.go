package lru_test

import (
	"context"
	"fmt"
	"time"

	"common/lru"
)

// ExampleNewMemCache shows the typical lifecycle of an in-memory LRU.
func ExampleNewMemCache() {
	c := lru.NewMemCache[string, int](lru.MemOptions[string, int]{
		Capacity: 3,
		TTL:      time.Minute,
	})

	ctx := context.Background()
	_ = c.Set(ctx, "a", 1)
	_ = c.Set(ctx, "b", 2)

	if v, ok, _ := c.Get(ctx, "a"); ok {
		fmt.Println("a =", v)
	}
	n, _ := c.Len(ctx)
	fmt.Println("len =", n)
	// Output:
	// a = 1
	// len = 2
}

// ExampleMemCache_eviction shows capacity-driven LRU eviction.
func ExampleMemCache_eviction() {
	c := lru.NewMemCache[string, int](lru.MemOptions[string, int]{
		Capacity: 2,
		OnEvict:  func(k string, v int) { fmt.Printf("evicted %s=%d\n", k, v) },
	})

	ctx := context.Background()
	_ = c.Set(ctx, "a", 1)
	_ = c.Set(ctx, "b", 2)
	_ = c.Set(ctx, "c", 3) // evicts "a"

	_, ok, _ := c.Get(ctx, "a")
	fmt.Println("a still present:", ok)
	// Output:
	// evicted a=1
	// a still present: false
}
