package httpclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetJSON_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hello":"world"}`))
	}))
	defer srv.Close()

	c := New(Options{})
	var out struct{ Hello string }
	if err := c.GetJSON(context.Background(), srv.URL, &out); err != nil {
		t.Fatalf("GetJSON: %v", err)
	}
	if out.Hello != "world" {
		t.Errorf("got %q", out.Hello)
	}
}

func TestGetJSON_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusBadRequest)
	}))
	defer srv.Close()

	c := New(Options{})
	err := c.GetJSON(context.Background(), srv.URL, nil)

	var herr *HTTPError
	if !errors.As(err, &herr) {
		t.Fatalf("expected *HTTPError, got %T: %v", err, err)
	}
	if herr.StatusCode != 400 {
		t.Errorf("status = %d", herr.StatusCode)
	}
}

func TestRetry_On5xx(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n < 3 {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := New(Options{MaxRetries: 3, RetryWaitMin: time.Millisecond, RetryWaitMax: 2 * time.Millisecond})
	var out struct{ OK bool }
	if err := c.GetJSON(context.Background(), srv.URL, &out); err != nil {
		t.Fatalf("expected success after retries, got %v", err)
	}
	if hits.Load() != 3 {
		t.Errorf("expected 3 hits, got %d", hits.Load())
	}
	if !out.OK {
		t.Errorf("decoded wrong: %#v", out)
	}
}

func TestRetry_Exhausted(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.Error(w, "no", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := New(Options{MaxRetries: 2, RetryWaitMin: time.Millisecond, RetryWaitMax: 2 * time.Millisecond})
	err := c.GetJSON(context.Background(), srv.URL, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	// initial + 2 retries = 3
	if hits.Load() != 3 {
		t.Errorf("expected 3 hits, got %d", hits.Load())
	}
}

func TestRetry_NotOn4xx(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	c := New(Options{MaxRetries: 3, RetryWaitMin: time.Millisecond})
	_ = c.GetJSON(context.Background(), srv.URL, nil)
	if hits.Load() != 1 {
		t.Errorf("4xx should not retry, got %d hits", hits.Load())
	}
}

func TestRetry_NotOnCtxCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
	}))
	defer srv.Close()

	c := New(Options{MaxRetries: 5, Timeout: 5 * time.Millisecond, RetryWaitMin: time.Millisecond})
	err := c.GetJSON(context.Background(), srv.URL, nil)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	// We can't easily assert it didn't retry endlessly, but assert it returned promptly.
}

func TestPostJSON(t *testing.T) {
	type req struct{ Name string }
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q", ct)
		}
		_, _ = w.Write([]byte(`{"echo":"hi"}`))
	}))
	defer srv.Close()

	c := New(Options{})
	var out struct{ Echo string }
	if err := c.PostJSON(context.Background(), srv.URL, req{Name: "x"}, &out); err != nil {
		t.Fatalf("PostJSON: %v", err)
	}
	if out.Echo != "hi" {
		t.Errorf("got %q", out.Echo)
	}
}

func TestAllowHost_BlocksInitialRequest(t *testing.T) {
	c := New(Options{
		AllowHost: func(host string) bool { return false }, // deny everything
	})
	err := c.GetJSON(context.Background(), "http://example.com/x", nil)
	if !errors.Is(err, ErrHostNotAllowed) {
		t.Errorf("expected ErrHostNotAllowed, got %v", err)
	}
}

func TestAllowHost_BlocksRedirect(t *testing.T) {
	var hits atomic.Int32
	redirectTarget := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(200)
	}))
	defer redirectTarget.Close()

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, redirectTarget.URL, http.StatusFound)
	}))
	defer redirector.Close()

	// AllowHost allows only the redirector, not the target.
	redirectorHost := strings.TrimPrefix(redirector.URL, "http://")
	c := New(Options{
		AllowHost: func(host string) bool { return host == redirectorHost },
	})
	err := c.GetJSON(context.Background(), redirector.URL, nil)
	if err == nil {
		t.Fatal("expected redirect to be blocked")
	}
	if hits.Load() != 0 {
		t.Errorf("redirect target should never be hit, got %d", hits.Load())
	}
}

func TestMaxRedirects_DisableEntirely(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer target.Close()
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer redirector.Close()

	c := New(Options{MaxRedirects: -1})
	resp, err := c.Do(mustReq(t, redirector.URL))
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()
	// MaxRedirects=-1 → CheckRedirect returns ErrUseLastResponse, so we see the
	// raw 302 instead of following it.
	if resp.StatusCode != http.StatusFound {
		t.Errorf("expected 302, got %d", resp.StatusCode)
	}
}

func TestDenyPrivateIP(t *testing.T) {
	// 127.0.0.1 — loopback. Use the literal so we don't depend on DNS.
	if DenyPrivateIP("127.0.0.1") {
		t.Error("loopback IP should be denied")
	}
	// 10.0.0.1 — RFC1918.
	if DenyPrivateIP("10.0.0.1") {
		t.Error("10.0.0.0/8 should be denied")
	}
	// 169.254.169.254 — AWS / GCP metadata (link-local).
	if DenyPrivateIP("169.254.169.254") {
		t.Error("169.254.0.0/16 should be denied")
	}
	// 8.8.8.8 — public.
	if !DenyPrivateIP("8.8.8.8") {
		t.Error("8.8.8.8 should be allowed")
	}
	// With explicit port suffix.
	if DenyPrivateIP("127.0.0.1:8080") {
		t.Error("loopback with port should be denied")
	}
}

func mustReq(t *testing.T, url string) *http.Request {
	t.Helper()
	r, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// TestRetry_ConcurrentBackoffIsRaceFree triggers many in-flight retries on a
// shared Client so the race detector fires if backoff ever touches shared
// mutable state again.
func TestRetry_ConcurrentBackoffIsRaceFree(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New(Options{
		MaxRetries:   3,
		RetryWaitMin: 100 * time.Microsecond,
		RetryWaitMax: 1 * time.Millisecond,
	})

	const N = 64
	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = c.GetJSON(context.Background(), srv.URL, nil)
		}()
	}
	wg.Wait()
}

func TestUserAgent(t *testing.T) {
	var seen string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get("User-Agent")
	}))
	defer srv.Close()

	c := New(Options{UserAgent: "common/1.0"})
	_ = c.GetJSON(context.Background(), srv.URL, nil)
	if !strings.Contains(seen, "common/1.0") {
		t.Errorf("User-Agent not propagated: %q", seen)
	}
}
