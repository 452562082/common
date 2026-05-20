// Package redis is a thin wrapper around go-redis/v9.
//
// It exists to:
//
//   - Centralise client construction with sensible defaults and an explicit
//     Ping-on-open.
//   - Offer a unified Client type that hides the standalone/cluster/sentinel
//     distinction behind go-redis's UniversalClient interface.
//   - Expose a Pinger interface so the graceful/health stack can probe Redis.
//
// For everyday calls just use the embedded redis.UniversalClient API:
//
//	c, err := redisx.Open(ctx, redisx.Options{Addrs: []string{"localhost:6379"}})
//	if err != nil { ... }
//	defer c.Close()
//
//	if err := c.Set(ctx, "k", "v", 0).Err(); err != nil { ... }
package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Mode selects which kind of redis topology to talk to. Defaults to Standalone.
type Mode string

const (
	ModeStandalone Mode = "standalone"
	ModeCluster    Mode = "cluster"
	ModeSentinel   Mode = "sentinel"
)

// Options configures Open.
type Options struct {
	// Mode picks the topology. Defaults to Standalone.
	Mode Mode

	// Addrs is the list of "host:port" entries. For Standalone, only the
	// first is used. For Cluster/Sentinel, all are used.
	Addrs []string

	// Username / Password are forwarded to the underlying client.
	Username string
	Password string

	// DB is the database index (Standalone / Sentinel only).
	DB int

	// MasterName is the Sentinel master name (Sentinel mode only).
	MasterName string

	// PoolSize defaults to 10 per CPU. ZeroValue → go-redis default.
	PoolSize int

	// DialTimeout / ReadTimeout / WriteTimeout default to 5s / 3s / 3s.
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	// PingTimeout caps the initial connectivity check. 0 = skip the check.
	PingTimeout time.Duration
}

// Client wraps a redis.UniversalClient so callers don't have to care about
// Cluster / Sentinel / Standalone at the call site.
type Client struct {
	redis.UniversalClient
}

// Open dials redis according to opts and verifies connectivity.
func Open(ctx context.Context, opts Options) (*Client, error) {
	if len(opts.Addrs) == 0 {
		return nil, errors.New("redis: Addrs is required")
	}
	if opts.PingTimeout == 0 {
		opts.PingTimeout = 5 * time.Second
	}
	if opts.DialTimeout == 0 {
		opts.DialTimeout = 5 * time.Second
	}
	if opts.ReadTimeout == 0 {
		opts.ReadTimeout = 3 * time.Second
	}
	if opts.WriteTimeout == 0 {
		opts.WriteTimeout = 3 * time.Second
	}

	uniOpts := &redis.UniversalOptions{
		Addrs:        opts.Addrs,
		Username:     opts.Username,
		Password:     opts.Password,
		DB:           opts.DB,
		MasterName:   opts.MasterName,
		PoolSize:     opts.PoolSize,
		DialTimeout:  opts.DialTimeout,
		ReadTimeout:  opts.ReadTimeout,
		WriteTimeout: opts.WriteTimeout,
	}

	var c redis.UniversalClient
	switch opts.Mode {
	case "", ModeStandalone:
		c = redis.NewClient(uniOpts.Simple())
	case ModeCluster:
		c = redis.NewClusterClient(uniOpts.Cluster())
	case ModeSentinel:
		if opts.MasterName == "" {
			return nil, errors.New("redis: Sentinel mode requires MasterName")
		}
		c = redis.NewFailoverClient(uniOpts.Failover())
	default:
		return nil, fmt.Errorf("redis: unknown Mode %q", opts.Mode)
	}

	if opts.PingTimeout > 0 {
		pingCtx, cancel := context.WithTimeout(ctx, opts.PingTimeout)
		defer cancel()
		if err := c.Ping(pingCtx).Err(); err != nil {
			_ = c.Close()
			return nil, fmt.Errorf("redis: ping: %w", err)
		}
	}
	return &Client{UniversalClient: c}, nil
}

// Close terminates the connection pool.
func (c *Client) Close() error {
	if err := c.UniversalClient.Close(); err != nil {
		return fmt.Errorf("redis: close: %w", err)
	}
	return nil
}

// Pinger lets external machinery (health checks, graceful shutdown) verify
// Redis connectivity without depending on the concrete *Client.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Ping satisfies the Pinger interface.
func (c *Client) Ping(ctx context.Context) error {
	if err := c.UniversalClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis: ping: %w", err)
	}
	return nil
}

// IsNil returns true when err is the go-redis sentinel for "no value".
// Handy when distinguishing missing keys from real errors.
//
//	v, err := c.Get(ctx, "k").Result()
//	if redisx.IsNil(err) { ... }
func IsNil(err error) bool { return errors.Is(err, redis.Nil) }

// Nil is re-exported so callers can build on it.
var Nil = redis.Nil
