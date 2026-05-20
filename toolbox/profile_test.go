package toolbox

import (
	"strings"
	"testing"
	"time"
)

func TestHumanBytes(t *testing.T) {
	tests := []struct {
		in   uint64
		want string
	}{
		{0, "0B"},
		{1023, "1023B"},
		{1024, "1.00K"},
		{1536, "1.50K"},
		{1024 * 1024, "1.00M"},
		{1024 * 1024 * 1024, "1.00G"},
	}
	for _, tt := range tests {
		if got := humanBytes(tt.in); got != tt.want {
			t.Errorf("humanBytes(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestHumanDuration(t *testing.T) {
	tests := []struct {
		in     time.Duration
		prefix string
	}{
		{0, "0"},
		{500 * time.Nanosecond, "500ns"},
		{500 * time.Microsecond, "500.00"},
		{500 * time.Millisecond, "500.00"},
		{2 * time.Second, "2.00s"},
		{2 * time.Minute, "2.00m"},
		{2 * time.Hour, "2.00h"},
	}
	for _, tt := range tests {
		got := humanDuration(tt.in)
		if !strings.HasPrefix(got, tt.prefix) {
			t.Errorf("humanDuration(%v) = %q, want prefix %q", tt.in, got, tt.prefix)
		}
	}
}

func TestAvgDuration(t *testing.T) {
	got := avgDuration([]time.Duration{time.Second, 2 * time.Second, 3 * time.Second})
	if got != 2*time.Second {
		t.Errorf("avgDuration = %v, want 2s", got)
	}
	if got := avgDuration(nil); got != 0 {
		t.Errorf("avgDuration(nil) = %v, want 0", got)
	}
}

func TestPrintGCSummary(t *testing.T) {
	s := PrintGCSummary()
	if s == "" {
		t.Fatal("PrintGCSummary returned empty string")
	}
	// Either form contains "Alloc:".
	if !strings.Contains(s, "Alloc:") {
		t.Errorf("expected 'Alloc:' in output, got %q", s)
	}
}
