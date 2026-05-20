// Package metrics is a thin Prometheus client wrapper.
//
// A typical setup looks like:
//
//	reg := metrics.NewRegistry(metrics.Options{
//	    Namespace: "billing",
//	    Subsystem: "api",
//	})
//	requests := reg.Counter("requests_total", "Total HTTP requests.", []string{"method", "code"})
//	latency  := reg.Histogram("request_seconds", "Request latency in seconds.", []string{"method"}, metrics.DefaultLatencyBuckets)
//
//	// Expose at /metrics:
//	http.Handle("/metrics", reg.Handler())
//
// The registry is isolated from the prom default global, so multiple
// independent registries can coexist in the same process (handy for tests).
package metrics

import (
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// DefaultLatencyBuckets is a reasonable bucket set for request latency in
// seconds: 1ms → 10s, ~10 buckets.
var DefaultLatencyBuckets = []float64{
	0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10,
}

// Options configures NewRegistry.
type Options struct {
	Namespace string
	Subsystem string

	// ConstLabels are attached to every metric registered through this registry.
	ConstLabels prometheus.Labels

	// DisableGoCollector skips the built-in Go runtime collector.
	DisableGoCollector bool

	// DisableProcessCollector skips the built-in process collector.
	DisableProcessCollector bool
}

// Registry wraps prometheus.Registry with helpers that automatically apply
// the namespace/subsystem/labels from Options.
type Registry struct {
	opts Options
	reg  *prometheus.Registry
	mu   sync.Mutex
}

// NewRegistry returns an initialised Registry.
func NewRegistry(opts Options) *Registry {
	reg := prometheus.NewRegistry()
	if !opts.DisableGoCollector {
		reg.MustRegister(collectors.NewGoCollector())
	}
	if !opts.DisableProcessCollector {
		reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	}
	return &Registry{opts: opts, reg: reg}
}

// Raw returns the underlying *prometheus.Registry so callers can plug in
// arbitrary collectors.
func (r *Registry) Raw() *prometheus.Registry { return r.reg }

// Handler returns an http.Handler that serves the Prometheus text exposition
// format, scoped to this registry only.
func (r *Registry) Handler() http.Handler {
	return promhttp.HandlerFor(r.reg, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// Counter registers and returns a new *prometheus.CounterVec.
// labels may be nil for a label-less counter.
func (r *Registry) Counter(name, help string, labels []string) *prometheus.CounterVec {
	c := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace:   r.opts.Namespace,
		Subsystem:   r.opts.Subsystem,
		Name:        name,
		Help:        help,
		ConstLabels: r.opts.ConstLabels,
	}, labels)
	r.register(c)
	return c
}

// Gauge registers and returns a new *prometheus.GaugeVec.
func (r *Registry) Gauge(name, help string, labels []string) *prometheus.GaugeVec {
	g := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   r.opts.Namespace,
		Subsystem:   r.opts.Subsystem,
		Name:        name,
		Help:        help,
		ConstLabels: r.opts.ConstLabels,
	}, labels)
	r.register(g)
	return g
}

// Histogram registers and returns a new *prometheus.HistogramVec.
// Pass nil for buckets to use prometheus's defaults.
func (r *Registry) Histogram(name, help string, labels []string, buckets []float64) *prometheus.HistogramVec {
	h := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace:   r.opts.Namespace,
		Subsystem:   r.opts.Subsystem,
		Name:        name,
		Help:        help,
		ConstLabels: r.opts.ConstLabels,
		Buckets:     buckets,
	}, labels)
	r.register(h)
	return h
}

// Summary registers and returns a new *prometheus.SummaryVec.
// Prefer Histogram unless you specifically need client-side quantiles.
func (r *Registry) Summary(name, help string, labels []string, objectives map[float64]float64) *prometheus.SummaryVec {
	if objectives == nil {
		objectives = map[float64]float64{
			0.5:  0.05,
			0.9:  0.01,
			0.99: 0.001,
		}
	}
	s := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:   r.opts.Namespace,
		Subsystem:   r.opts.Subsystem,
		Name:        name,
		Help:        help,
		ConstLabels: r.opts.ConstLabels,
		Objectives:  objectives,
	}, labels)
	r.register(s)
	return s
}

// Register lets callers add custom collectors.
func (r *Registry) Register(c prometheus.Collector) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.reg.Register(c)
}

func (r *Registry) register(c prometheus.Collector) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reg.MustRegister(c)
}

// Timer measures elapsed time and reports it to a histogram on Stop.
//
//	t := metrics.NewTimer(latency.WithLabelValues("GET"))
//	defer t.ObserveDuration()
type Timer struct {
	start time.Time
	obs   prometheus.Observer
}

// NewTimer starts a new timer.
func NewTimer(obs prometheus.Observer) *Timer {
	return &Timer{start: time.Now(), obs: obs}
}

// ObserveDuration reports elapsed seconds to the underlying observer.
func (t *Timer) ObserveDuration() time.Duration {
	d := time.Since(t.start)
	t.obs.Observe(d.Seconds())
	return d
}
