package errpkg

import (
	"errors"
	"io"
	"testing"
)

func BenchmarkNew(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = New("X", 500, "msg %d", i)
	}
}

func BenchmarkWrap(b *testing.B) {
	cause := io.EOF
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Wrap(cause, "X", 500, "msg")
	}
}

func BenchmarkIs(b *testing.B) {
	e := Wrap(io.EOF, "READ_FAIL", 500, "read")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = errors.Is(e, io.EOF)
	}
}

func BenchmarkNew_NoStack(b *testing.B) {
	orig := StackDepth()
	defer SetStackDepth(orig)
	SetStackDepth(0)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = New("X", 500, "msg")
	}
}
