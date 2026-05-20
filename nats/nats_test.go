package nats

import (
	"context"
	"testing"
	"time"
)

// Live tests need a running nats-server; CI runs them against a container.
// Here we exercise the validation surface.

func TestOpen_RequiresURL(t *testing.T) {
	if _, err := Open(context.Background(), Options{}); err == nil {
		t.Error("expected error for empty URL")
	}
}

func TestOpen_BadURL(t *testing.T) {
	if _, err := Open(context.Background(), Options{URL: "nats://127.0.0.1:1"}); err == nil {
		t.Error("expected error dialling closed port")
	}
}

func TestOpen_CtxCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	// Dial a black-hole address (TEST-NET-1) so connect blocks; ctx must fire.
	_, err := Open(ctx, Options{URL: "nats://192.0.2.1:4222"})
	if err == nil {
		t.Fatal("expected ctx-cancel error")
	}
}
