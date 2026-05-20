package etcd

import (
	"context"
	"testing"
)

// Live etcd tests require a running cluster, which we don't spin up here.
// These tests cover the validation surface; integration tests live in CI
// against a real etcd container.

func TestOpen_RequiresEndpoints(t *testing.T) {
	if _, err := Open(context.Background(), Options{}); err == nil {
		t.Error("expected error for empty endpoints")
	}
}

func TestErrNotAcquiredExported(t *testing.T) {
	if ErrNotAcquired == nil || ErrNotAcquired.Error() == "" {
		t.Error("ErrNotAcquired should be a usable sentinel")
	}
}

func TestEventTypeConstants(t *testing.T) {
	if EventPut == EventDelete {
		t.Error("EventPut and EventDelete must be distinct")
	}
}
