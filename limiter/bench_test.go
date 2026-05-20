package limiter

import (
	"strconv"
	"testing"
	"time"
)

func BenchmarkTokenBucket_Allow_SameKey(b *testing.B) {
	tb := NewTokenBucket(TokenBucketOptions{Rate: 1e9, Burst: 1 << 30})
	defer tb.Close()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tb.Allow("k")
	}
}

func BenchmarkTokenBucket_Allow_ManyKeys(b *testing.B) {
	tb := NewTokenBucket(TokenBucketOptions{Rate: 1e9, Burst: 1 << 30, MaxKeys: 100_000})
	defer tb.Close()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tb.Allow(strconv.Itoa(i & 0xFFFF))
	}
}

func BenchmarkTokenBucket_Allow_Parallel(b *testing.B) {
	tb := NewTokenBucket(TokenBucketOptions{Rate: 1e9, Burst: 1 << 30})
	defer tb.Close()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = tb.Allow("shared")
		}
	})
}

func BenchmarkSlidingWindow_Allow(b *testing.B) {
	// Realistic limit — production code rarely needs millions per window.
	sw := NewSlidingWindow(10_000, time.Hour)
	defer sw.Close()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sw.Allow("k")
	}
}
