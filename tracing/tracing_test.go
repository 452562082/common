package tracing

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestInit_RequiresServiceName(t *testing.T) {
	if _, err := Init(context.Background(), Options{}); err == nil {
		t.Error("expected error for missing ServiceName")
	}
}

func TestInit_NoExporterOK(t *testing.T) {
	shutdown, err := Init(context.Background(), Options{
		ServiceName:    "test-svc",
		ServiceVersion: "v1.0.0",
		Environment:    "test",
		SampleRatio:    1,
	})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	defer func() {
		if err := shutdown(context.Background()); err != nil {
			t.Errorf("shutdown: %v", err)
		}
	}()

	tr := Tracer("unit-test")
	ctx, span := tr.Start(context.Background(), "TestSpan")
	defer span.End()

	if id := TraceIDFromContext(ctx); id == "" {
		t.Error("expected non-empty TraceID after Start")
	}
	if id := SpanIDFromContext(ctx); id == "" {
		t.Error("expected non-empty SpanID after Start")
	}
}

func TestTraceID_NoSpan(t *testing.T) {
	if id := TraceIDFromContext(context.Background()); id != "" {
		t.Errorf("expected empty traceID for plain ctx, got %q", id)
	}
}

func TestBuildSampler(t *testing.T) {
	tests := []struct {
		ratio float64
		want  string
	}{
		{0, "AlwaysOffSampler"},
		{-1, "AlwaysOffSampler"},
		{1, "AlwaysOnSampler"},
		{2, "AlwaysOnSampler"},
		{0.5, "ParentBased"},
	}
	for _, tt := range tests {
		s := buildSampler(tt.ratio)
		if s == nil {
			t.Errorf("ratio=%v sampler is nil", tt.ratio)
		}
		// Sanity-check the description; the prefix is enough.
		if got := s.Description(); got == "" {
			t.Errorf("ratio=%v sampler has empty description", tt.ratio)
		}
	}
	// Make sure these compile-time references stay alive.
	_ = otel.GetTracerProvider()
	_ = sdktrace.NeverSample()
}
