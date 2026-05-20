// Package tracing initialises an OpenTelemetry tracer that exports spans over
// OTLP/HTTP (the modern default; collectors and most backends speak it).
//
// Usage:
//
//	shutdown, err := tracing.Init(ctx, tracing.Options{
//	    ServiceName: "billing-api",
//	    Endpoint:    "otel-collector:4318",
//	    SampleRatio: 0.1,
//	})
//	if err != nil { log.Fatal(err) }
//	defer shutdown(context.Background())
//
//	tr := tracing.Tracer("billing")
//	ctx, span := tr.Start(ctx, "ChargeCard")
//	defer span.End()
//
// Init also wires the global text-map propagator (W3C tracecontext +
// baggage), so any otel instrumentation library (otelhttp, otelsql, ...)
// works automatically.
package tracing

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
	"go.opentelemetry.io/otel/trace"
)

// Options configures Init.
type Options struct {
	// ServiceName populates the standard service.name resource attribute. Required.
	ServiceName string

	// ServiceVersion populates service.version when set.
	ServiceVersion string

	// Environment populates deployment.environment.name when set.
	Environment string

	// Endpoint is the OTLP/HTTP collector endpoint, host:port (no scheme).
	// Empty disables exporting (the tracer still runs; useful for tests).
	Endpoint string

	// Insecure forces plaintext HTTP. Default is HTTPS.
	Insecure bool

	// SampleRatio is in [0, 1]. 0 disables sampling, 1 samples every span.
	// Zero falls back to 1.0; pass a negative value to mean 0.
	SampleRatio float64

	// Headers attached to every export request (e.g. auth tokens).
	Headers map[string]string
}

// Shutdown flushes pending spans and tears down the provider.
type Shutdown func(context.Context) error

// Init configures a global tracer provider and propagator.
// The returned Shutdown should be called during application teardown.
func Init(ctx context.Context, opts Options) (Shutdown, error) {
	if opts.ServiceName == "" {
		return nil, errors.New("tracing: ServiceName is required")
	}

	res, err := buildResource(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("tracing: build resource: %w", err)
	}

	tpOpts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		sdktrace.WithSampler(buildSampler(opts.SampleRatio)),
	}

	if opts.Endpoint != "" {
		exporter, err := buildExporter(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("tracing: build exporter: %w", err)
		}
		tpOpts = append(tpOpts, sdktrace.WithBatcher(exporter))
	}

	tp := sdktrace.NewTracerProvider(tpOpts...)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return func(ctx context.Context) error {
		if err := tp.Shutdown(ctx); err != nil {
			return fmt.Errorf("tracing: shutdown: %w", err)
		}
		return nil
	}, nil
}

func buildResource(ctx context.Context, opts Options) (*resource.Resource, error) {
	attrs := []attribute.KeyValue{semconv.ServiceName(opts.ServiceName)}
	if opts.ServiceVersion != "" {
		attrs = append(attrs, semconv.ServiceVersion(opts.ServiceVersion))
	}
	if opts.Environment != "" {
		attrs = append(attrs, semconv.DeploymentEnvironmentName(opts.Environment))
	}
	return resource.New(ctx, resource.WithAttributes(attrs...))
}

func buildSampler(ratio float64) sdktrace.Sampler {
	switch {
	case ratio < 0 || ratio == 0:
		return sdktrace.NeverSample()
	case ratio >= 1:
		return sdktrace.AlwaysSample()
	default:
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))
	}
}

func buildExporter(ctx context.Context, opts Options) (*otlptrace.Exporter, error) {
	httpOpts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(opts.Endpoint),
	}
	if opts.Insecure {
		httpOpts = append(httpOpts, otlptracehttp.WithInsecure())
	}
	if len(opts.Headers) > 0 {
		httpOpts = append(httpOpts, otlptracehttp.WithHeaders(opts.Headers))
	}
	return otlptracehttp.New(ctx, httpOpts...)
}

// Tracer returns a tracer for the named instrumentation library.
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// SpanFromContext returns the active span on ctx, or a no-op span.
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// TraceIDFromContext returns the active trace ID as a hex string, or "" when
// no span is on ctx. Use this to bridge into logger.WithTraceID.
func TraceIDFromContext(ctx context.Context) string {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return ""
	}
	return sc.TraceID().String()
}

// SpanIDFromContext returns the active span ID as a hex string, or "".
func SpanIDFromContext(ctx context.Context) string {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return ""
	}
	return sc.SpanID().String()
}
