package httpserver

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// freeAddr asks the OS for an unused TCP port on 127.0.0.1.
func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	_ = l.Close()
	return addr
}

// startInBackground starts srv.Start in a goroutine and waits until the
// listener is accepting connections.
func startInBackground(t *testing.T, srv *Server) (done chan error) {
	t.Helper()
	done = make(chan error, 1)
	go func() { done <- srv.Start(context.Background()) }()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		c, err := net.Dial("tcp", srv.Addr())
		if err == nil {
			_ = c.Close()
			return done
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("server did not become ready")
	return done
}

func TestServer_StartShutdown(t *testing.T) {
	srv := New(Options{Addr: freeAddr(t), Logger: quietLogger()})
	srv.Router().Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("pong"))
	})

	done := startInBackground(t, srv)

	resp, err := http.Get("http://" + srv.Addr() + "/ping")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "pong" {
		t.Errorf("body = %q", body)
	}

	if err := srv.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	if err := <-done; err != nil && !errors.Is(err, http.ErrServerClosed) {
		t.Errorf("Start returned %v", err)
	}
}

func TestServer_RecoverMiddleware(t *testing.T) {
	srv := New(Options{Addr: freeAddr(t), Logger: quietLogger()})
	srv.Router().Get("/boom", func(w http.ResponseWriter, r *http.Request) {
		panic("nope")
	})

	done := startInBackground(t, srv)
	defer func() {
		_ = srv.Shutdown(context.Background())
		<-done
	}()

	resp, err := http.Get("http://" + srv.Addr() + "/boom")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d", resp.StatusCode)
	}
}

func TestServer_HealthCheck(t *testing.T) {
	srv := New(Options{
		Addr:        freeAddr(t),
		Logger:      quietLogger(),
		HealthCheck: func(ctx context.Context) error { return nil },
	})

	done := startInBackground(t, srv)
	defer func() {
		_ = srv.Shutdown(context.Background())
		<-done
	}()

	resp, err := http.Get("http://" + srv.Addr() + "/healthz")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d", resp.StatusCode)
	}
}

func TestServer_MaxBodyBytes_Limits(t *testing.T) {
	srv := New(Options{
		Addr:         freeAddr(t),
		Logger:       quietLogger(),
		MaxBodyBytes: 16,
	})
	srv.Router().Post("/upload", func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	done := startInBackground(t, srv)
	defer func() {
		_ = srv.Shutdown(context.Background())
		<-done
	}()

	// Under the limit — should succeed.
	resp, err := http.Post("http://"+srv.Addr()+"/upload", "application/octet-stream",
		strings.NewReader("short"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("small body should succeed; got %d", resp.StatusCode)
	}

	// Over the limit — handler should see MaxBytesError on read.
	big := strings.Repeat("x", 1024)
	resp, err = http.Post("http://"+srv.Addr()+"/upload", "application/octet-stream",
		strings.NewReader(big))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Errorf("large body should have been rejected; got %d", resp.StatusCode)
	}
}

func TestServer_HealthCheck_Failure(t *testing.T) {
	srv := New(Options{
		Addr:        freeAddr(t),
		Logger:      quietLogger(),
		HealthCheck: func(ctx context.Context) error { return errors.New("dial tcp 10.1.2.3:3306: refused") },
	})

	done := startInBackground(t, srv)
	defer func() {
		_ = srv.Shutdown(context.Background())
		<-done
	}()

	resp, err := http.Get("http://" + srv.Addr() + "/healthz")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d", resp.StatusCode)
	}
	// The internal error MUST NOT appear in the response body — that would
	// leak infrastructure details (host:port, driver-specific messages, etc).
	if strings.Contains(string(body), "10.1.2.3") || strings.Contains(string(body), "refused") {
		t.Errorf("response leaked internal error: %s", body)
	}
	if !strings.Contains(string(body), "unhealthy") {
		t.Errorf("expected generic unhealthy body, got: %s", body)
	}
}
