package breaker

import (
	"context"
	"testing"
)

func BenchmarkDo_Closed(b *testing.B) {
	br := New(Options{Name: "bench"})
	ctx := context.Background()
	noop := func(ctx context.Context) error { return nil }
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = br.Do(ctx, noop)
	}
}

func BenchmarkDo_Parallel(b *testing.B) {
	br := New(Options{Name: "bench"})
	ctx := context.Background()
	noop := func(ctx context.Context) error { return nil }
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = br.Do(ctx, noop)
		}
	})
}
