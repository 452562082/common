// Example: bounded in-memory LRU with TTL and eviction callback.
//
//	go run ./examples/lru_mem
package main

import (
	"context"
	"fmt"
	"time"

	"common/lru"
)

func main() {
	c := lru.NewMemCache[string, int](lru.MemOptions[string, int]{
		Capacity: 3,
		TTL:      200 * time.Millisecond,
		OnEvict:  func(k string, v int) { fmt.Printf("evicted %s=%d\n", k, v) },
	})

	ctx := context.Background()
	for i, k := range []string{"a", "b", "c", "d"} {
		_ = c.Set(ctx, k, i)
	}

	n, _ := c.Len(ctx)
	fmt.Printf("len after inserts: %d\n", n)

	time.Sleep(250 * time.Millisecond)
	_ = c.PurgeExpired(ctx)
	n, _ = c.Len(ctx)
	fmt.Printf("len after TTL expiry + Purge: %d\n", n)
}
