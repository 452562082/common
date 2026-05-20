package idgen

import "testing"

func BenchmarkUUIDv4(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = UUID()
	}
}

func BenchmarkUUIDv7(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = UUIDv7()
	}
}

func BenchmarkSnowflake_Next(b *testing.B) {
	s, _ := NewSnowflake(1)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.Next()
	}
}

func BenchmarkSnowflake_Parallel(b *testing.B) {
	s, _ := NewSnowflake(1)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = s.Next()
		}
	})
}
