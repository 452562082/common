package lru

import (
	"context"
	"strconv"
	"testing"
)

func BenchmarkMemCache_Set(b *testing.B) {
	c := NewMemCache[string, int](MemOptions[string, int]{Capacity: 10_000})
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.Set(ctx, strconv.Itoa(i&0xFFF), i)
	}
}

func BenchmarkMemCache_Get_Hit(b *testing.B) {
	c := NewMemCache[string, int](MemOptions[string, int]{Capacity: 10_000})
	ctx := context.Background()
	for i := 0; i < 1024; i++ {
		_ = c.Set(ctx, strconv.Itoa(i), i)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = c.Get(ctx, strconv.Itoa(i&0x3FF))
	}
}

func BenchmarkMemCache_Get_Miss(b *testing.B) {
	c := NewMemCache[string, int](MemOptions[string, int]{Capacity: 100})
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = c.Get(ctx, "nope")
	}
}

func BenchmarkMemCache_Concurrent(b *testing.B) {
	c := NewMemCache[string, int](MemOptions[string, int]{Capacity: 10_000})
	ctx := context.Background()
	for i := 0; i < 1024; i++ {
		_ = c.Set(ctx, strconv.Itoa(i), i)
	}
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, _, _ = c.Get(ctx, strconv.Itoa(i&0x3FF))
			i++
		}
	})
}
