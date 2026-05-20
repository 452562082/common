// Package logger provides a thin, opinionated wrapper around log/slog with
// three additions over the stdlib defaults:
//
//   - JSON / text format selectable from config.
//   - Optional file output with size-based rotation (lumberjack).
//   - Context propagation of request-scoped fields (traceID, requestID, ...)
//     so they appear on every log line without explicit re-passing.
//
// The package-level Default logger can also replace the stdlib default with
// SetAsDefault so that any third-party code calling slog.Info("...") inherits
// the same configuration.
//
// # Security: do not log secrets
//
// slog records every key-value attribute verbatim. NEVER pass passwords,
// API keys, session tokens, JWTs, full credit-card numbers, or other secrets
// as log fields:
//
//	// BAD — the password ends up in every log sink (file, stdout, Loki, etc).
//	log.Info("login", "user", uid, "password", pw)
//
//	// OK — log identifiers only, not credentials.
//	log.Info("login", "user", uid)
//
// If you need a secret to flow through some middle layer, redact it at the
// boundary (e.g. replace with "***" or a fingerprint) before it reaches any
// logging call. Audit logs and metrics dashboards routinely outlive the
// processes that wrote them — assume anything you log is forever.
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Format is the on-disk encoding of log records.
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

// Options configures New.
type Options struct {
	// Level is the minimum level that will be emitted. Zero value = Info.
	Level slog.Level

	// Format selects the handler. Zero value = JSON.
	Format Format

	// AddSource includes file:line in each record. Useful in development,
	// non-trivial cost in hot paths.
	AddSource bool

	// Writer overrides the destination. If nil and File is unset, os.Stderr is used.
	Writer io.Writer

	// File enables size-rotated file logging. Ignored when Writer is set.
	File *FileOptions

	// Attrs is appended to every record (e.g. {"service": "billing"}).
	Attrs []slog.Attr
}

// FileOptions controls lumberjack-backed file rotation.
type FileOptions struct {
	Path       string // required
	MaxSizeMB  int    // default 100
	MaxBackups int    // default 7
	MaxAgeDays int    // default 30
	Compress   bool   // gzip rotated files
}

// New returns a configured *slog.Logger.
func New(opts Options) *slog.Logger {
	w := resolveWriter(opts)
	handler := buildHandler(w, opts)
	l := slog.New(handler)
	if len(opts.Attrs) > 0 {
		args := make([]any, 0, len(opts.Attrs))
		for _, a := range opts.Attrs {
			args = append(args, a)
		}
		l = l.With(args...)
	}
	return l
}

func resolveWriter(opts Options) io.Writer {
	if opts.Writer != nil {
		return opts.Writer
	}
	if opts.File != nil && opts.File.Path != "" {
		return newRotatingWriter(opts.File)
	}
	return os.Stderr
}

func newRotatingWriter(f *FileOptions) io.Writer {
	lj := &lumberjack.Logger{
		Filename:   f.Path,
		MaxSize:    100,
		MaxBackups: 7,
		MaxAge:     30,
		Compress:   f.Compress,
	}
	if f.MaxSizeMB > 0 {
		lj.MaxSize = f.MaxSizeMB
	}
	if f.MaxBackups > 0 {
		lj.MaxBackups = f.MaxBackups
	}
	if f.MaxAgeDays > 0 {
		lj.MaxAge = f.MaxAgeDays
	}
	return lj
}

func buildHandler(w io.Writer, opts Options) slog.Handler {
	ho := &slog.HandlerOptions{
		Level:     opts.Level,
		AddSource: opts.AddSource,
	}
	var base slog.Handler
	switch opts.Format {
	case FormatText:
		base = slog.NewTextHandler(w, ho)
	default:
		base = slog.NewJSONHandler(w, ho)
	}
	return &contextHandler{Handler: base}
}

// ParseLevel converts a string like "debug" / "info" / "warn" / "error" to slog.Level.
// Unknown values fall back to Info.
func ParseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// --- Default logger -----------------------------------------------------------

var (
	defaultLogger atomic.Pointer[slog.Logger]
	defaultOnce   sync.Once
)

// Default returns the package-level logger. The first call lazily initialises
// it with sensible defaults (Info / JSON / stderr).
func Default() *slog.Logger {
	defaultOnce.Do(func() {
		defaultLogger.Store(New(Options{}))
	})
	return defaultLogger.Load()
}

// SetDefault replaces the package-level logger.
func SetDefault(l *slog.Logger) {
	defaultLogger.Store(l)
}

// SetAsStdDefault makes the supplied logger the destination for the stdlib's
// log.Default() and slog.Default(). Call this once in main().
func SetAsStdDefault(l *slog.Logger) {
	slog.SetDefault(l)
}

// --- Context-aware fields -----------------------------------------------------

type ctxKey struct{ name string }

var (
	traceIDKey   = ctxKey{"trace_id"}
	requestIDKey = ctxKey{"request_id"}
	userIDKey    = ctxKey{"user_id"}
)

// WithTraceID returns a new context that carries traceID. The logger picks it
// up automatically when logging with that ctx.
func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceIDKey, id)
}

// WithRequestID returns a new context that carries requestID.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// WithUserID returns a new context that carries userID.
func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// TraceID returns the traceID set on ctx, or "" if none.
func TraceID(ctx context.Context) string { return strValue(ctx, traceIDKey) }

// RequestID returns the requestID set on ctx, or "" if none.
func RequestID(ctx context.Context) string { return strValue(ctx, requestIDKey) }

// UserID returns the userID set on ctx, or "" if none.
func UserID(ctx context.Context) string { return strValue(ctx, userIDKey) }

func strValue(ctx context.Context, k ctxKey) string {
	if v, ok := ctx.Value(k).(string); ok {
		return v
	}
	return ""
}

// contextHandler decorates records that came in via slog.*Context with
// ctx-derived attributes.
type contextHandler struct{ slog.Handler }

func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if ctx != nil {
		if v := TraceID(ctx); v != "" {
			r.AddAttrs(slog.String("trace_id", v))
		}
		if v := RequestID(ctx); v != "" {
			r.AddAttrs(slog.String("request_id", v))
		}
		if v := UserID(ctx); v != "" {
			r.AddAttrs(slog.String("user_id", v))
		}
	}
	return h.Handler.Handle(ctx, r)
}

func (h *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &contextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *contextHandler) WithGroup(name string) slog.Handler {
	return &contextHandler{Handler: h.Handler.WithGroup(name)}
}

// String allows %v formatting of Format.
func (f Format) String() string { return string(f) }
