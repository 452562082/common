package utils

import (
	"bytes"
	"strings"
	"testing"
)

func TestStringToBytes_BytesToString_RoundTrip(t *testing.T) {
	cases := []string{"", "x", "hello, 世界", strings.Repeat("a", 4096)}
	for _, s := range cases {
		b := StringToBytes(s)
		if string(b) != s {
			t.Errorf("StringToBytes(%q) → %q", s, b)
		}
		if got := BytesToString(b); got != s {
			t.Errorf("BytesToString round-trip: got %q want %q", got, s)
		}
	}
}

func TestAcquireRelease(t *testing.T) {
	buf := AcquireByteBuffer()
	buf.WriteString("hello")
	if buf.String() != "hello" {
		t.Fatalf("buf content: %q", buf.String())
	}
	ReleaseByteBuffer(buf)
	// After release, the pool may give us a fresh buffer, but reset must have run.
	buf2 := AcquireByteBuffer()
	if buf2.Len() != 0 {
		t.Errorf("expected empty buffer after release, got len=%d", buf2.Len())
	}
}

func TestReadAllToByteBuffer(t *testing.T) {
	src := strings.NewReader("payload")
	buf := AcquireByteBuffer()
	defer ReleaseByteBuffer(buf)

	got, err := ReadAllToByteBuffer(src, buf)
	if err != nil {
		t.Fatalf("ReadAllToByteBuffer: %v", err)
	}
	if !bytes.Equal(got, []byte("payload")) {
		t.Errorf("got %q", got)
	}
}
