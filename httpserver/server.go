// Package httpserver provides a small starter for HTTP services: a chi-based
// router pre-wired with logger / recovery / request-id / metrics / tracing
// middleware, plus a Start/Shutdown lifecycle that plays well with the
// graceful package.
//
// Minimal example:
//
//	srv := httpserver.New(httpserver.Options{
//	    Addr: ":8080",
//	})
//	srv.Router().Get("/hello", func(w http.ResponseWriter, r *http.Request) {
//	    w.Write([]byte("hi"))
//	})
//
//	if err := srv.Start(ctx); err != nil { log.Fatal(err) }
//	defer srv.Shutdown(context.Background())
package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"common/httpserver/middleware"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

// Options configures New.
type Options struct {
	// Addr is the listen address, e.g. ":8080". Required.
	Addr string

	// ReadTimeout / WriteTimeout / IdleTimeout cap a single connection.
	// Zero falls back to sensible defaults.
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration

	// ShutdownTimeout caps the graceful drain on Shutdown(). Default 30s.
	ShutdownTimeout time.Duration

	// MaxBodyBytes caps the size of any incoming request body, in bytes.
	// Defaults to 4 MiB. Set to -1 to disable the limit (not recommended for
	// internet-facing services). Set to 0 to keep the default.
	MaxBodyBytes int64

	// Logger is used by the request-logging middleware. nil = slog.Default().
	Logger *slog.Logger

	// DisableDefaults skips installation of the built-in middleware stack
	// (request-id, recovery, logger). Use this to install your own order.
	DisableDefaults bool

	// MetricsHandler, if non-nil, is registered at MetricsPath.
	MetricsHandler http.Handler
	MetricsPath    string // default "/metrics"

	// HealthCheck, if set, is registered at HealthPath. The handler should
	// return nil for "healthy".
	HealthCheck func(ctx context.Context) error
	HealthPath  string // default "/healthz"
}

// Server bundles a chi.Router + *http.Server with start/shutdown helpers.
type Server struct {
	opts   Options
	router *chi.Mux
	server *http.Server
	log    *slog.Logger
}

// New returns a Server configured per opts.
// It does not start listening yet; call Start to do that.
func New(opts Options) *Server {
	applyHTTPDefaults(&opts)

	log := opts.Logger
	if log == nil {
		log = slog.Default()
	}

	r := chi.NewRouter()
	if !opts.DisableDefaults {
		r.Use(chimw.RequestID)
		r.Use(chimw.RealIP)
		r.Use(middleware.Recover(log))
		if opts.MaxBodyBytes > 0 {
			r.Use(middleware.MaxBody(opts.MaxBodyBytes))
		}
		r.Use(middleware.AccessLog(log))
	}
	if opts.MetricsHandler != nil {
		r.Method(http.MethodGet, opts.MetricsPath, opts.MetricsHandler)
	}
	if opts.HealthCheck != nil {
		r.Get(opts.HealthPath, healthHandler(opts.HealthCheck, log))
	}

	return &Server{
		opts:   opts,
		router: r,
		log:    log,
		server: &http.Server{
			Addr:         opts.Addr,
			Handler:      r,
			ReadTimeout:  opts.ReadTimeout,
			WriteTimeout: opts.WriteTimeout,
			IdleTimeout:  opts.IdleTimeout,
		},
	}
}

func applyHTTPDefaults(o *Options) {
	if o.ReadTimeout == 0 {
		o.ReadTimeout = 10 * time.Second
	}
	if o.WriteTimeout == 0 {
		o.WriteTimeout = 30 * time.Second
	}
	if o.IdleTimeout == 0 {
		o.IdleTimeout = 90 * time.Second
	}
	if o.ShutdownTimeout == 0 {
		o.ShutdownTimeout = 30 * time.Second
	}
	switch {
	case o.MaxBodyBytes == 0:
		o.MaxBodyBytes = 4 << 20 // 4 MiB
	case o.MaxBodyBytes < 0:
		o.MaxBodyBytes = 0 // explicit "off"
	}
	if o.MetricsPath == "" {
		o.MetricsPath = "/metrics"
	}
	if o.HealthPath == "" {
		o.HealthPath = "/healthz"
	}
}

// Router returns the underlying chi.Mux so callers can mount routes / nested
// routers / additional middleware.
func (s *Server) Router() *chi.Mux { return s.router }

// HTTPServer returns the wrapped *http.Server. Mostly useful for tests.
func (s *Server) HTTPServer() *http.Server { return s.server }

// Addr returns the listen address as configured.
func (s *Server) Addr() string { return s.opts.Addr }

// Start binds the listener and serves requests. It blocks until the server
// exits — successful Shutdown returns nil, any other error is returned wrapped.
//
// This signature is designed to plug into graceful.App.Add directly.
func (s *Server) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.opts.Addr)
	if err != nil {
		return fmt.Errorf("httpserver: listen %s: %w", s.opts.Addr, err)
	}
	s.log.Info("httpserver: listening", "addr", ln.Addr().String())
	if err := s.server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("httpserver: serve: %w", err)
	}
	return nil
}

// Shutdown drains in-flight requests, bounded by Options.ShutdownTimeout.
// Pass an external ctx if you want a tighter bound.
func (s *Server) Shutdown(ctx context.Context) error {
	deadline, cancel := context.WithTimeout(ctx, s.opts.ShutdownTimeout)
	defer cancel()
	if err := s.server.Shutdown(deadline); err != nil {
		return fmt.Errorf("httpserver: shutdown: %w", err)
	}
	return nil
}

// healthHandler returns a public-facing health endpoint that NEVER leaks the
// error returned by the check (which typically contains an internal IP, db
// host:port, or other infrastructure detail). Failures are slog-ged so that
// operators still have full information server-side.
func healthHandler(check func(ctx context.Context) error, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := check(r.Context()); err != nil {
			log.ErrorContext(r.Context(), "httpserver: health check failed", "err", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"unhealthy"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}
}
