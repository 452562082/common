package nats

import (
	"testing"
)

// Live tests need a running nats-server; CI runs them against a container.
// Here we exercise the validation surface.

func TestOpen_RequiresURL(t *testing.T) {
	if _, err := Open(Options{}); err == nil {
		t.Error("expected error for empty URL")
	}
}

func TestOpen_BadURL(t *testing.T) {
	if _, err := Open(Options{URL: "nats://127.0.0.1:1"}); err == nil {
		t.Error("expected error dialling closed port")
	}
}
