// Package middleware bundles the small set of HTTP middleware used by
// httpserver.Server's default stack: access logging, panic recovery, request-ID
// propagation, CORS, and Prometheus / OpenTelemetry instrumentation.
package middleware

import (
	"net/http"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"common/logger"

	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
)

// AccessLogOptions tunes AccessLogWithOptions.
type AccessLogOptions struct {
	// LogQueryKeys is the allowlist of query parameters whose values should be
	// logged verbatim. Anything not on the list is omitted (the key is
	// preserved for context, redacted as "***").
	//
	// Empty / nil: the query string is dropped entirely — safest default,
	// since URLs in the wild routinely carry tokens, session IDs, and PII.
	LogQueryKeys []string
}

// AccessLog logs one line per request using slog.
//
// The line includes method, path, status, bytes, latency, trace_id and
// request_id (from chi.RequestID).
//
// Important: the raw query string is NOT logged. URLs commonly carry tokens
// (callback URLs, presigned URLs, OAuth code exchanges, ...) and unsafe
// secret leakage into log aggregators is one of the most common audit
// findings. Use AccessLogWithOptions if you need specific query keys logged.
func AccessLog(log *slog.Logger) func(http.Handler) http.Handler {
	return AccessLogWithOptions(log, AccessLogOptions{})
}

// AccessLogWithOptions is AccessLog with an explicit query-key allowlist.
//
//	r.Use(middleware.AccessLogWithOptions(log, middleware.AccessLogOptions{
//	    LogQueryKeys: []string{"page", "sort"},
//	}))
func AccessLogWithOptions(log *slog.Logger, opts AccessLogOptions) func(http.Handler) http.Handler {
	allow := make(map[string]struct{}, len(opts.LogQueryKeys))
	for _, k := range opts.LogQueryKeys {
		allow[k] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()

			reqID := middleware.GetReqID(r.Context())
			ctx := r.Context()
			if reqID != "" {
				ctx = logger.WithRequestID(ctx, reqID)
				r = r.WithContext(ctx)
			}

			defer func() {
				attrs := []any{
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Int("status", ww.Status()),
					slog.Int("bytes", ww.BytesWritten()),
					slog.Duration("elapsed", time.Since(start)),
				}
				if r.URL.RawQuery != "" {
					if redacted := redactQuery(r.URL.Query(), allow); redacted != "" {
						attrs = append(attrs, slog.String("query", redacted))
					}
				}
				log.InfoContext(r.Context(), "http access", attrs...)
			}()
			next.ServeHTTP(ww, r)
		})
	}
}

// redactQuery renders a URL query: values for keys in allow are kept;
// every other value is replaced by "***". Returns "" if nothing remained
// after redaction (i.e. allow was empty).
func redactQuery(q map[string][]string, allow map[string]struct{}) string {
	if len(allow) == 0 {
		return ""
	}
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	first := true
	for _, k := range keys {
		for _, v := range q[k] {
			if !first {
				sb.WriteByte('&')
			}
			first = false
			sb.WriteString(k)
			sb.WriteByte('=')
			if _, ok := allow[k]; ok {
				sb.WriteString(v)
			} else {
				sb.WriteString("***")
			}
		}
	}
	return sb.String()
}

// MaxBody wraps r.Body in http.MaxBytesReader so any handler that reads it
// past the limit gets EOF + a typed *http.MaxBytesError. Combine with
// Server.Options.MaxBodyBytes to cap memory use from huge POSTs.
func MaxBody(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil && r.ContentLength != 0 {
				r.Body = http.MaxBytesReader(w, r.Body, limit)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Recover catches panics, logs them with stack, and returns a 500 response.
func Recover(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					if rec == http.ErrAbortHandler {
						panic(rec)
					}
					log.ErrorContext(r.Context(), "http panic",
						"err", rec,
						"path", r.URL.Path,
						"stack", string(debug.Stack()),
					)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(`{"error":"internal error"}`))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// CORS is a permissive cross-origin handler. Pass the allowed-origin list
// (or {"*"} for fully open).
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	allowAll := false
	allow := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		if o == "*" {
			allowAll = true
			continue
		}
		allow[o] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if allowAll {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				} else if _, ok := allow[origin]; ok {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Vary", "Origin")
				}
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-Id")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Tracing extracts the trace context from inbound headers and starts a server
// span for every request, naming it "<METHOD> <route>".
//
// It also threads the trace ID into the slog context (logger.WithTraceID) so
// AccessLog and any subsequent log call carries it automatically.
func Tracing(serviceName string) func(http.Handler) http.Handler {
	tracer := otel.Tracer(serviceName)
	prop := otel.GetTextMapPropagator()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := prop.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
			route := r.URL.Path
			ctx, span := tracer.Start(ctx, r.Method+" "+route, trace.WithSpanKind(trace.SpanKindServer))
			defer span.End()

			if tid := span.SpanContext().TraceID(); tid.IsValid() {
				ctx = logger.WithTraceID(ctx, tid.String())
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// PrometheusMetrics exposes request count + latency. The two metrics are
// registered against the supplied registerer (e.g. metrics.Registry.Raw()).
//
// Cardinality: method × status only. The path is intentionally NOT a label —
// templated routes are hard to extract here without a router-aware glue layer,
// and unbounded paths explode cardinality. Add a custom middleware if you
// need per-route metrics.
func PrometheusMetrics(reg prometheus.Registerer, namespace, subsystem string) func(http.Handler) http.Handler {
	requests := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "http_requests_total",
		Help:      "HTTP requests handled, partitioned by method and status code.",
	}, []string{"method", "code"})

	latency := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "http_request_duration_seconds",
		Help:      "HTTP request latency in seconds.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method"})

	reg.MustRegister(requests, latency)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()
			next.ServeHTTP(ww, r)
			elapsed := time.Since(start).Seconds()

			code := normaliseStatusCode(ww.Status())
			requests.WithLabelValues(r.Method, code).Inc()
			latency.WithLabelValues(r.Method).Observe(elapsed)
		})
	}
}

func normaliseStatusCode(code int) string {
	switch {
	case code == 0:
		return "200"
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500 && code < 600:
		return "5xx"
	default:
		return strings.Repeat("?", 1)
	}
}
