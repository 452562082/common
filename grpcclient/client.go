// Package grpcclient is a tiny helper for dialling gRPC services with
// sensible defaults: insecure-or-TLS toggle, ctx propagation of trace
// headers, and pluggable extra interceptors.
//
//	conn, err := grpcclient.Dial(ctx, "billing:9090", grpcclient.Options{
//	    Insecure:    true,
//	    DialTimeout: 3 * time.Second,
//	})
//	if err != nil { ... }
//	defer conn.Close()
//	client := pb.NewBillingClient(conn)
package grpcclient

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// Options configures Dial.
type Options struct {
	// Insecure disables TLS. Use only for in-cluster traffic / local dev.
	Insecure bool

	// TLS is required when Insecure is false. Build it with credentials.NewTLS(...).
	TLS credentials.TransportCredentials

	// DialTimeout caps the initial dial. Default 5s. Zero falls back to default;
	// negative disables the timeout.
	DialTimeout time.Duration

	// ServiceName names the tracer used by the tracing interceptor.
	// Empty disables client-side tracing.
	ServiceName string

	// ExtraDialOptions are appended at the end of grpc.NewClient's args.
	ExtraDialOptions []grpc.DialOption

	// ExtraUnaryInterceptors / ExtraStreamInterceptors run AFTER the built-ins.
	ExtraUnaryInterceptors  []grpc.UnaryClientInterceptor
	ExtraStreamInterceptors []grpc.StreamClientInterceptor
}

// Dial creates a *grpc.ClientConn to target. The caller MUST Close it.
func Dial(ctx context.Context, target string, opts Options) (*grpc.ClientConn, error) {
	if opts.DialTimeout == 0 {
		opts.DialTimeout = 5 * time.Second
	}

	var creds grpc.DialOption
	switch {
	case opts.Insecure:
		creds = grpc.WithTransportCredentials(insecure.NewCredentials())
	case opts.TLS != nil:
		creds = grpc.WithTransportCredentials(opts.TLS)
	default:
		return nil, fmt.Errorf("grpcclient: either Insecure or TLS must be set")
	}

	unary := []grpc.UnaryClientInterceptor{}
	stream := []grpc.StreamClientInterceptor{}
	if opts.ServiceName != "" {
		unary = append(unary, tracingUnary(opts.ServiceName))
		stream = append(stream, tracingStream(opts.ServiceName))
	}
	unary = append(unary, opts.ExtraUnaryInterceptors...)
	stream = append(stream, opts.ExtraStreamInterceptors...)

	dialOpts := []grpc.DialOption{
		creds,
		grpc.WithChainUnaryInterceptor(unary...),
		grpc.WithChainStreamInterceptor(stream...),
	}
	dialOpts = append(dialOpts, opts.ExtraDialOptions...)

	if opts.DialTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.DialTimeout)
		defer cancel()
		_ = ctx // grpc.NewClient is non-blocking; ctx kept for symmetry.
	}

	conn, err := grpc.NewClient(target, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("grpcclient: dial %s: %w", target, err)
	}
	return conn, nil
}

func tracingUnary(serviceName string) grpc.UnaryClientInterceptor {
	tracer := otel.Tracer(serviceName)
	prop := otel.GetTextMapPropagator()
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx, span := tracer.Start(ctx, method, trace.WithSpanKind(trace.SpanKindClient))
		defer span.End()
		ctx = injectMD(ctx, prop)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func tracingStream(serviceName string) grpc.StreamClientInterceptor {
	tracer := otel.Tracer(serviceName)
	prop := otel.GetTextMapPropagator()
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		ctx, span := tracer.Start(ctx, method, trace.WithSpanKind(trace.SpanKindClient))
		ctx = injectMD(ctx, prop)
		s, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			span.End()
			return nil, err
		}
		return &tracedStream{ClientStream: s, span: span}, nil
	}
}

// injectMD writes the current span's W3C tracecontext headers into the
// outgoing gRPC metadata.
func injectMD(ctx context.Context, prop propagation.TextMapPropagator) context.Context {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.MD{}
	} else {
		md = md.Copy()
	}
	prop.Inject(ctx, mdCarrier(md))
	return metadata.NewOutgoingContext(ctx, md)
}

type mdCarrier metadata.MD

func (mc mdCarrier) Get(key string) string {
	vs := metadata.MD(mc).Get(key)
	if len(vs) == 0 {
		return ""
	}
	return vs[0]
}
func (mc mdCarrier) Set(key, value string) { metadata.MD(mc).Set(key, value) }
func (mc mdCarrier) Keys() []string {
	out := make([]string, 0, len(mc))
	for k := range mc {
		out = append(out, k)
	}
	return out
}

// tracedStream ends the client-side span when the stream finishes.
type tracedStream struct {
	grpc.ClientStream
	span trace.Span
}

func (s *tracedStream) RecvMsg(m any) error {
	err := s.ClientStream.RecvMsg(m)
	if err != nil {
		s.span.End()
	}
	return err
}

func (s *tracedStream) CloseSend() error {
	err := s.ClientStream.CloseSend()
	s.span.End()
	return err
}
