package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := map[string]slog.Level{
		"debug":    slog.LevelDebug,
		"INFO":     slog.LevelInfo,
		"warn":     slog.LevelWarn,
		"warning":  slog.LevelWarn,
		"ERROR":    slog.LevelError,
		"":         slog.LevelInfo,
		"garbage":  slog.LevelInfo,
		" debug ":  slog.LevelDebug,
	}
	for in, want := range tests {
		if got := ParseLevel(in); got != want {
			t.Errorf("ParseLevel(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestContextFields(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{Writer: &buf, Format: FormatJSON})

	ctx := WithTraceID(context.Background(), "trace-1")
	ctx = WithRequestID(ctx, "req-2")
	ctx = WithUserID(ctx, "user-3")

	l.InfoContext(ctx, "hello", "extra", "value")

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("decode: %v (raw=%s)", err, buf.String())
	}
	for _, want := range []struct{ k, v string }{
		{"trace_id", "trace-1"},
		{"request_id", "req-2"},
		{"user_id", "user-3"},
		{"extra", "value"},
		{"msg", "hello"},
	} {
		if rec[want.k] != want.v {
			t.Errorf("field %q = %v, want %v (full=%v)", want.k, rec[want.k], want.v, rec)
		}
	}
}

func TestContextFields_NoneSet(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{Writer: &buf, Format: FormatJSON})

	l.InfoContext(context.Background(), "no fields")
	if strings.Contains(buf.String(), "trace_id") || strings.Contains(buf.String(), "request_id") {
		t.Errorf("unexpected ctx fields in: %s", buf.String())
	}
}

func TestLevelFilter(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{Writer: &buf, Format: FormatJSON, Level: slog.LevelWarn})

	l.Info("info-line")
	l.Warn("warn-line")
	out := buf.String()
	if strings.Contains(out, "info-line") {
		t.Errorf("Info should be filtered, got: %s", out)
	}
	if !strings.Contains(out, "warn-line") {
		t.Errorf("Warn should pass, got: %s", out)
	}
}

func TestTextFormat(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{Writer: &buf, Format: FormatText})
	l.Info("plain-text", "k", "v")
	if !strings.Contains(buf.String(), "plain-text") || !strings.Contains(buf.String(), "k=v") {
		t.Errorf("text format unexpected: %s", buf.String())
	}
}

func TestDefault(t *testing.T) {
	if Default() == nil {
		t.Fatal("Default() returned nil")
	}
	// Replacing default is safe.
	var buf bytes.Buffer
	SetDefault(New(Options{Writer: &buf}))
	Default().Info("test-default")
	if !strings.Contains(buf.String(), "test-default") {
		t.Errorf("SetDefault did not take effect: %s", buf.String())
	}
}

func TestWithAttrs(t *testing.T) {
	var buf bytes.Buffer
	l := New(Options{
		Writer: &buf,
		Format: FormatJSON,
		Attrs:  []slog.Attr{slog.String("service", "test")},
	})
	l.Info("hi")
	if !strings.Contains(buf.String(), `"service":"test"`) {
		t.Errorf("expected service=test, got %s", buf.String())
	}
}
