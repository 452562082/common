package grpcserver

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"common/grpcclient"

	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	a := l.Addr().String()
	_ = l.Close()
	return a
}

func waitListening(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		c, err := net.Dial("tcp", addr)
		if err == nil {
			_ = c.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("server did not become ready")
}

func TestServer_HealthCheck(t *testing.T) {
	srv := New(Options{Addr: freeAddr(t), Logger: quietLogger()})
	done := make(chan error, 1)
	go func() { done <- srv.Start(context.Background()) }()
	waitListening(t, srv.Addr())
	defer func() {
		_ = srv.Shutdown(context.Background())
		<-done
	}()

	conn, err := grpcclient.Dial(context.Background(), srv.Addr(),
		grpcclient.Options{Insecure: true, DialTimeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	hc := healthpb.NewHealthClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	resp, err := hc.Check(ctx, &healthpb.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if resp.Status != healthpb.HealthCheckResponse_SERVING {
		t.Errorf("status = %v", resp.Status)
	}
}

func TestServer_DisableHealth(t *testing.T) {
	srv := New(Options{Addr: freeAddr(t), Logger: quietLogger(), DisableHealth: true})
	if srv.Health() != nil {
		t.Error("expected Health() to be nil when DisableHealth=true")
	}
}

func TestServer_ShutdownIdempotent(t *testing.T) {
	srv := New(Options{Addr: freeAddr(t), Logger: quietLogger()})
	done := make(chan error, 1)
	go func() { done <- srv.Start(context.Background()) }()
	waitListening(t, srv.Addr())

	if err := srv.Shutdown(context.Background()); err != nil {
		t.Errorf("first Shutdown: %v", err)
	}
	// Calling Shutdown after the server is fully stopped should be a no-op.
	if err := srv.Shutdown(context.Background()); err != nil {
		t.Errorf("second Shutdown: %v", err)
	}
	<-done
}
