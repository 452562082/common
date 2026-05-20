package utils

import (
	"strings"
	"testing"
)

var benchString = strings.Repeat("hello world ", 64)

func BenchmarkStringToBytes(b *testing.B) {
	s := benchString
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = StringToBytes(s)
	}
}

func BenchmarkStringToBytes_Std(b *testing.B) {
	// Baseline: stdlib []byte(s) conversion (allocates).
	s := benchString
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = []byte(s)
	}
}

func BenchmarkBytesToString(b *testing.B) {
	bs := []byte(benchString)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = BytesToString(bs)
	}
}

func BenchmarkAcquireRelease(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := AcquireByteBuffer()
		buf.WriteString("payload")
		ReleaseByteBuffer(buf)
	}
}
