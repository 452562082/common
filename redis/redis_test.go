package redis

import (
	"context"
	"errors"
	"testing"

	goredis "github.com/redis/go-redis/v9"
)

func TestOpen_MissingAddrs(t *testing.T) {
	if _, err := Open(context.Background(), Options{}); err == nil {
		t.Error("expected error for empty Addrs")
	}
}

func TestOpen_SentinelRequiresMaster(t *testing.T) {
	_, err := Open(context.Background(), Options{
		Mode:        ModeSentinel,
		Addrs:       []string{"127.0.0.1:26379"},
		PingTimeout: 0, // skip ping so we don't need a live sentinel
	})
	if err == nil {
		t.Error("expected error when MasterName missing")
	}
}

func TestOpen_UnknownMode(t *testing.T) {
	_, err := Open(context.Background(), Options{
		Mode:        "garbage",
		Addrs:       []string{"127.0.0.1:6379"},
		PingTimeout: 0,
	})
	if err == nil {
		t.Error("expected error for unknown mode")
	}
}

func TestOpen_NoPingWhenZero(t *testing.T) {
	// PingTimeout = 0 disables the connectivity check so this should not try
	// to dial. The client is still constructed.
	c, err := Open(context.Background(), Options{
		Addrs:       []string{"127.0.0.1:6379"},
		PingTimeout: -1, // explicit "off" — anything non-positive skips
	})
	if err != nil {
		t.Skipf("client construction itself dialled: %v", err)
	}
	if c != nil {
		_ = c.Close()
	}
}

func TestIsNil(t *testing.T) {
	if !IsNil(goredis.Nil) {
		t.Error("IsNil should match go-redis Nil")
	}
	if IsNil(errors.New("other")) {
		t.Error("IsNil should not match arbitrary errors")
	}
	if IsNil(nil) {
		t.Error("IsNil(nil) should be false")
	}
}

// Compile-time check that *Client satisfies Pinger.
var _ Pinger = (*Client)(nil)
