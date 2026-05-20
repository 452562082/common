package metrics

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRegistry_CounterIncrement(t *testing.T) {
	r := NewRegistry(Options{Namespace: "ns", Subsystem: "sub"})
	c := r.Counter("hits_total", "test", []string{"endpoint"})
	c.WithLabelValues("/foo").Inc()
	c.WithLabelValues("/foo").Add(2)

	body := scrape(t, r)
	if !strings.Contains(body, `ns_sub_hits_total{endpoint="/foo"} 3`) {
		t.Errorf("missing expected metric line in:\n%s", body)
	}
}

func TestRegistry_HistogramObservation(t *testing.T) {
	r := NewRegistry(Options{Namespace: "test"})
	h := r.Histogram("latency_seconds", "test", []string{"path"}, DefaultLatencyBuckets)
	h.WithLabelValues("/a").Observe(0.123)
	h.WithLabelValues("/a").Observe(0.5)

	body := scrape(t, r)
	if !strings.Contains(body, "test_latency_seconds_count") {
		t.Errorf("missing count line in:\n%s", body)
	}
	if !strings.Contains(body, "test_latency_seconds_sum") {
		t.Errorf("missing sum line in:\n%s", body)
	}
}

func TestRegistry_Gauge(t *testing.T) {
	r := NewRegistry(Options{})
	g := r.Gauge("queue_depth", "test", nil)
	g.WithLabelValues().Set(42)

	body := scrape(t, r)
	if !strings.Contains(body, "queue_depth 42") {
		t.Errorf("expected gauge value; body:\n%s", body)
	}
}

func TestRegistry_ConstLabels(t *testing.T) {
	r := NewRegistry(Options{
		ConstLabels: map[string]string{"service": "billing"},
	})
	c := r.Counter("x_total", "h", nil)
	c.WithLabelValues().Inc()

	body := scrape(t, r)
	if !strings.Contains(body, `service="billing"`) {
		t.Errorf("ConstLabels missing in body:\n%s", body)
	}
}

func TestTimer_ObserveDuration(t *testing.T) {
	r := NewRegistry(Options{})
	h := r.Histogram("op_seconds", "h", []string{"op"}, DefaultLatencyBuckets)

	tm := NewTimer(h.WithLabelValues("work"))
	time.Sleep(time.Millisecond)
	if d := tm.ObserveDuration(); d <= 0 {
		t.Errorf("duration should be positive, got %v", d)
	}

	body := scrape(t, r)
	if !strings.Contains(body, "op_seconds_count") {
		t.Errorf("timer didn't record:\n%s", body)
	}
}

func TestRegistry_GoCollector(t *testing.T) {
	r := NewRegistry(Options{}) // default collectors enabled
	body := scrape(t, r)
	if !strings.Contains(body, "go_goroutines") {
		t.Errorf("expected go_goroutines from default collector; body had:\n%s", body)
	}
}

func TestRegistry_DisableCollectors(t *testing.T) {
	r := NewRegistry(Options{DisableGoCollector: true, DisableProcessCollector: true})
	body := scrape(t, r)
	if strings.Contains(body, "go_goroutines") || strings.Contains(body, "process_cpu_seconds") {
		t.Errorf("collectors should be disabled; body:\n%s", body)
	}
}

func scrape(t *testing.T, r *Registry) string {
	t.Helper()
	srv := httptest.NewServer(r.Handler())
	defer srv.Close()
	resp, err := srv.Client().Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return string(b)
}
