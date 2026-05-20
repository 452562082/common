// Package grpcserver provides a small starter for gRPC services with the same
// shape as the httpserver package: a Server with a Start/Shutdown lifecycle
// that plugs into graceful.App, pre-wired unary and stream interceptors for
// logger / recovery / tracing / metrics, and an optional health service.
//
// Typical use:
//
//	srv := grpcserver.New(grpcserver.Options{Addr: ":9090"})
//	pb.RegisterMyServer(srv.GRPC(), &myHandler{})
//	if err := srv.Start(ctx); err != nil { log.Fatal(err) }
//	defer srv.Shutdown(ctx)
package grpcserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"runtime/debug"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// Options configures New.
type Options struct {
	// Addr is the listen address, e.g. ":9090". Required.
	Addr string

	// MaxRecvMsgSize / MaxSendMsgSize override grpc defaults.
	// Zero falls back to grpc's defaults.
	MaxRecvMsgSize int
	MaxSendMsgSize int

	// ShutdownTimeout caps GracefulStop. Default 30s.
	ShutdownTimeout time.Duration

	// Logger is used by the logging interceptor. nil = slog.Default().
	Logger *slog.Logger

	// ServiceName is the OpenTelemetry tracer name. Empty disables tracing.
	ServiceName string

	// DisableHealth skips registration of the standard health service.
	DisableHealth bool

	// EnableReflection registers grpc reflection (useful for grpcurl).
	EnableReflection bool

	// ExtraServerOptions are appended to the constructor's args. Use this to
	// add TLS credentials, keepalive params, custom interceptors, ...
	ExtraServerOptions []grpc.ServerOption

	// ExtraUnaryInterceptors / ExtraStreamInterceptors run AFTER the built-ins.
	ExtraUnaryInterceptors  []grpc.UnaryServerInterceptor
	ExtraStreamInterceptors []grpc.StreamServerInterceptor
}

// Server bundles a *grpc.Server with start/shutdown helpers.
type Server struct {
	opts   Options
	server *grpc.Server
	health *health.Server
	log    *slog.Logger
}

// New builds a Server. The gRPC server is not started until Start is called.
func New(opts Options) *Server {
	applyGRPCDefaults(&opts)

	log := opts.Logger
	if log == nil {
		log = slog.Default()
	}

	unary := []grpc.UnaryServerInterceptor{
		recoverUnary(log),
		loggerUnary(log),
	}
	stream := []grpc.StreamServerInterceptor{
		recoverStream(log),
		loggerStream(log),
	}
	if opts.ServiceName != "" {
		unary = append(unary, tracingUnary(opts.ServiceName))
		stream = append(stream, tracingStream(opts.ServiceName))
	}
	unary = append(unary, opts.ExtraUnaryInterceptors...)
	stream = append(stream, opts.ExtraStreamInterceptors...)

	serverOpts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(unary...),
		grpc.ChainStreamInterceptor(stream...),
	}
	if opts.MaxRecvMsgSize > 0 {
		serverOpts = append(serverOpts, grpc.MaxRecvMsgSize(opts.MaxRecvMsgSize))
	}
	if opts.MaxSendMsgSize > 0 {
		serverOpts = append(serverOpts, grpc.MaxSendMsgSize(opts.MaxSendMsgSize))
	}
	serverOpts = append(serverOpts, opts.ExtraServerOptions...)

	s := grpc.NewServer(serverOpts...)
	srv := &Server{opts: opts, server: s, log: log}

	if !opts.DisableHealth {
		srv.health = health.NewServer()
		healthpb.RegisterHealthServer(s, srv.health)
		srv.health.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	}
	if opts.EnableReflection {
		reflection.Register(s)
	}
	return srv
}

func applyGRPCDefaults(o *Options) {
	if o.ShutdownTimeout == 0 {
		o.ShutdownTimeout = 30 * time.Second
	}
}

// GRPC returns the underlying *grpc.Server so callers can Register handlers.
func (s *Server) GRPC() *grpc.Server { return s.server }

// Health returns the health service (or nil when DisableHealth was set).
// Use it to flip per-service serving status during graceful drain.
func (s *Server) Health() *health.Server { return s.health }

// Addr returns the configured listen address.
func (s *Server) Addr() string { return s.opts.Addr }

// Start binds the listener and serves. Blocks until Shutdown is called.
// Designed to plug directly into graceful.App.Add.
func (s *Server) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.opts.Addr)
	if err != nil {
		return fmt.Errorf("grpcserver: listen %s: %w", s.opts.Addr, err)
	}
	s.log.Info("grpcserver: listening", "addr", ln.Addr().String())
	if err := s.server.Serve(ln); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return fmt.Errorf("grpcserver: serve: %w", err)
	}
	return nil
}

// Shutdown initiates a graceful stop. If the deadline ctx fires first, falls
// back to forceful Stop.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.health != nil {
		s.health.Shutdown()
	}
	done := make(chan struct{})
	go func() {
		s.server.GracefulStop()
		close(done)
	}()

	timeout := s.opts.ShutdownTimeout
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		s.server.Stop()
		return fmt.Errorf("grpcserver: shutdown ctx: %w", ctx.Err())
	case <-t.C:
		s.server.Stop()
		return fmt.Errorf("grpcserver: graceful stop timed out after %s", timeout)
	}
}

// ----------------------------------------------------------------------------
// Built-in interceptors
// ----------------------------------------------------------------------------

func recoverUnary(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.ErrorContext(ctx, "grpcserver: panic",
					"method", info.FullMethod,
					"err", r,
					"stack", string(debug.Stack()),
				)
				err = status.Error(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}

func recoverStream(log *slog.Logger) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				log.ErrorContext(ss.Context(), "grpcserver: panic (stream)",
					"method", info.FullMethod,
					"err", r,
					"stack", string(debug.Stack()),
				)
				err = status.Error(codes.Internal, "internal server error")
			}
		}()
		return handler(srv, ss)
	}
}

func loggerUnary(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		log.InfoContext(ctx, "grpcserver: rpc",
			"method", info.FullMethod,
			"code", codeOf(err).String(),
			"elapsed", time.Since(start),
		)
		return resp, err
	}
}

func loggerStream(log *slog.Logger) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		err := handler(srv, ss)
		log.InfoContext(ss.Context(), "grpcserver: stream",
			"method", info.FullMethod,
			"code", codeOf(err).String(),
			"elapsed", time.Since(start),
		)
		return err
	}
}

func codeOf(err error) codes.Code {
	if err == nil {
		return codes.OK
	}
	return status.Code(err)
}

func tracingUnary(serviceName string) grpc.UnaryServerInterceptor {
	tracer := otel.Tracer(serviceName)
	prop := otel.GetTextMapPropagator()
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		ctx = prop.Extract(ctx, mdCarrier(ctx))
		ctx, span := tracer.Start(ctx, info.FullMethod, trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()
		return handler(ctx, req)
	}
}

func tracingStream(serviceName string) grpc.StreamServerInterceptor {
	tracer := otel.Tracer(serviceName)
	prop := otel.GetTextMapPropagator()
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := prop.Extract(ss.Context(), mdCarrier(ss.Context()))
		ctx, span := tracer.Start(ctx, info.FullMethod, trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()
		return handler(srv, &wrappedStream{ServerStream: ss, ctx: ctx})
	}
}

// wrappedStream lets us swap the ctx in a server stream so the started span is
// visible to downstream code that calls ss.Context().
type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context { return w.ctx }

// mdCarrier adapts incoming metadata.MD to TextMapCarrier.
func mdCarrier(ctx context.Context) propagation.TextMapCarrier {
	md, _ := metadata.FromIncomingContext(ctx)
	return metadataCarrier(md)
}

type metadataCarrier metadata.MD

func (mc metadataCarrier) Get(key string) string {
	vs := metadata.MD(mc).Get(key)
	if len(vs) == 0 {
		return ""
	}
	return vs[0]
}
func (mc metadataCarrier) Set(key, value string)  { metadata.MD(mc).Set(key, value) }
func (mc metadataCarrier) Keys() []string {
	out := make([]string, 0, len(mc))
	for k := range mc {
		out = append(out, k)
	}
	return out
}
