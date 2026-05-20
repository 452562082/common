package limiter_test

import (
	"fmt"

	"common/limiter"
)

// ExampleNewTokenBucket throttles per-IP requests to 10 r/s with a burst of 20.
func ExampleNewTokenBucket() {
	tb := limiter.NewTokenBucket(limiter.TokenBucketOptions{
		Rate:    10,
		Burst:   20,
		MaxKeys: 10_000, // cap memory under IP-spoofing
	})
	defer tb.Close()

	const key = "203.0.113.4"
	allowed := 0
	for i := 0; i < 30; i++ {
		if tb.Allow(key) {
			allowed++
		}
	}
	fmt.Println("allowed of 30:", allowed)
	// Output:
	// allowed of 30: 20
}

// ExampleNewSlidingWindow shows a hard "N requests per minute" budget.
func ExampleNewSlidingWindow() {
	sw := limiter.NewSlidingWindow(3, 1<<30) // huge window, illustrative
	defer sw.Close()

	for i := 0; i < 5; i++ {
		fmt.Println("call", i+1, "->", sw.Allow("k"))
	}
	// Output:
	// call 1 -> true
	// call 2 -> true
	// call 3 -> true
	// call 4 -> false
	// call 5 -> false
}
