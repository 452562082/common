// Package etcd is a small wrapper around go.etcd.io/etcd/client/v3.
//
// It covers the three most common use cases for etcd in microservice
// codebases:
//
//   - KV store: Get / Put / Delete / List with prefix.
//   - Service discovery: Register an ephemeral key tied to a lease that
//     auto-renews while the process is alive, plus Discover / Watch helpers
//     for clients to react to the set of live instances.
//   - Distributed lock: Lock / TryLock backed by concurrency.Mutex.
//
// Each capability is a separate struct so callers can wire only what they need.
package etcd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

// Options configures Open.
type Options struct {
	// Endpoints is the list of etcd endpoints, e.g. ["etcd-0:2379", "etcd-1:2379"].
	Endpoints []string

	// Username / Password enable basic auth. Both empty = no auth.
	Username string
	Password string

	// DialTimeout caps the initial connect. Default 5s.
	DialTimeout time.Duration

	// RequestTimeout caps each individual KV / lease call. Default 5s.
	RequestTimeout time.Duration
}

// Client wraps a *clientv3.Client and exposes ergonomic helpers.
type Client struct {
	raw  *clientv3.Client
	opts Options
}

// Open dials the cluster and returns a Client.
func Open(ctx context.Context, opts Options) (*Client, error) {
	if len(opts.Endpoints) == 0 {
		return nil, errors.New("etcd: Endpoints is required")
	}
	if opts.DialTimeout == 0 {
		opts.DialTimeout = 5 * time.Second
	}
	if opts.RequestTimeout == 0 {
		opts.RequestTimeout = 5 * time.Second
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   opts.Endpoints,
		Username:    opts.Username,
		Password:    opts.Password,
		DialTimeout: opts.DialTimeout,
		Context:     ctx,
	})
	if err != nil {
		return nil, fmt.Errorf("etcd: connect: %w", err)
	}
	return &Client{raw: cli, opts: opts}, nil
}

// Raw returns the underlying *clientv3.Client for advanced features.
func (c *Client) Raw() *clientv3.Client { return c.raw }

// Close releases the connection.
func (c *Client) Close() error {
	if err := c.raw.Close(); err != nil {
		return fmt.Errorf("etcd: close: %w", err)
	}
	return nil
}

// ---------- KV ---------------------------------------------------------------

// Get returns the value for key, or ("", false, nil) when it doesn't exist.
func (c *Client) Get(ctx context.Context, key string) (string, bool, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	resp, err := c.raw.Get(ctx, key)
	if err != nil {
		return "", false, fmt.Errorf("etcd: get %s: %w", key, err)
	}
	if len(resp.Kvs) == 0 {
		return "", false, nil
	}
	return string(resp.Kvs[0].Value), true, nil
}

// Put writes value under key. Use PutWithLease for ephemeral keys.
func (c *Client) Put(ctx context.Context, key, value string) error {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	if _, err := c.raw.Put(ctx, key, value); err != nil {
		return fmt.Errorf("etcd: put %s: %w", key, err)
	}
	return nil
}

// Delete removes a key. Missing keys are not errors.
func (c *Client) Delete(ctx context.Context, key string) error {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	if _, err := c.raw.Delete(ctx, key); err != nil {
		return fmt.Errorf("etcd: delete %s: %w", key, err)
	}
	return nil
}

// DeletePrefix removes every key starting with prefix.
func (c *Client) DeletePrefix(ctx context.Context, prefix string) (deleted int64, err error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	resp, err := c.raw.Delete(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return 0, fmt.Errorf("etcd: delete prefix %s: %w", prefix, err)
	}
	return resp.Deleted, nil
}

// List returns key/value pairs whose key starts with prefix.
func (c *Client) List(ctx context.Context, prefix string) (map[string]string, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	resp, err := c.raw.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("etcd: list %s: %w", prefix, err)
	}
	out := make(map[string]string, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		out[string(kv.Key)] = string(kv.Value)
	}
	return out, nil
}

// ---------- Lease / Service registration ------------------------------------

// Registration keeps a lease alive in the background. When Close is called the
// lease is revoked, removing the key immediately rather than waiting for TTL.
type Registration struct {
	client  *Client
	leaseID clientv3.LeaseID
	cancel  context.CancelFunc
	done    chan struct{}
	key     string
}

// Register associates key=value with a fresh lease and starts a keep-alive
// loop. The key disappears when Close is called or the process loses its
// session.
func (c *Client) Register(ctx context.Context, key, value string, ttl time.Duration) (*Registration, error) {
	if ttl < time.Second {
		ttl = time.Second
	}
	leaseResp, err := c.raw.Grant(ctx, int64(ttl/time.Second))
	if err != nil {
		return nil, fmt.Errorf("etcd: grant lease: %w", err)
	}
	if _, err := c.raw.Put(ctx, key, value, clientv3.WithLease(leaseResp.ID)); err != nil {
		_, _ = c.raw.Revoke(context.Background(), leaseResp.ID)
		return nil, fmt.Errorf("etcd: put with lease: %w", err)
	}

	keepCtx, cancel := context.WithCancel(context.Background())
	ch, err := c.raw.KeepAlive(keepCtx, leaseResp.ID)
	if err != nil {
		cancel()
		_, _ = c.raw.Revoke(context.Background(), leaseResp.ID)
		return nil, fmt.Errorf("etcd: keepalive: %w", err)
	}

	r := &Registration{
		client:  c,
		leaseID: leaseResp.ID,
		cancel:  cancel,
		done:    make(chan struct{}),
		key:     key,
	}
	go func() {
		defer close(r.done)
		for range ch {
			// drain
		}
	}()
	return r, nil
}

// Close revokes the lease (which deletes the registered key) and stops the
// keepalive loop. Safe to call multiple times.
func (r *Registration) Close(ctx context.Context) error {
	r.cancel()
	<-r.done
	if _, err := r.client.raw.Revoke(ctx, r.leaseID); err != nil {
		return fmt.Errorf("etcd: revoke %s: %w", r.key, err)
	}
	return nil
}

// LeaseID is mostly useful for tests / diagnostics.
func (r *Registration) LeaseID() clientv3.LeaseID { return r.leaseID }

// ---------- Watch -----------------------------------------------------------

// Event mirrors clientv3.Event without exposing the upstream type.
type Event struct {
	Type   EventType
	Key    string
	Value  string
}

// EventType is either Put or Delete.
type EventType int

const (
	EventPut    EventType = 1
	EventDelete EventType = 2
)

// WatchPrefix streams every change under prefix to fn. The call returns when
// ctx is cancelled.
//
// fn is invoked under a panic-recover so a single bad event doesn't tear
// down the watch goroutine; panics are logged via slog.
func (c *Client) WatchPrefix(ctx context.Context, prefix string, fn func(Event)) error {
	ch := c.raw.Watch(ctx, prefix, clientv3.WithPrefix())
	for w := range ch {
		if err := w.Err(); err != nil {
			return fmt.Errorf("etcd: watch %s: %w", prefix, err)
		}
		for _, e := range w.Events {
			ev := Event{
				Type:  EventType(e.Type) + 1, // shift: clientv3 Put=0, Delete=1
				Key:   string(e.Kv.Key),
				Value: string(e.Kv.Value),
			}
			safeWatchInvoke(ctx, prefix, fn, ev)
		}
	}
	return nil
}

func safeWatchInvoke(ctx context.Context, prefix string, fn func(Event), ev Event) {
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "etcd: watch handler panic",
				"prefix", prefix,
				"key", ev.Key,
				"err", r,
			)
		}
	}()
	fn(ev)
}

// ---------- Distributed lock ------------------------------------------------

// Mutex is a session-scoped distributed lock backed by concurrency.Mutex.
type Mutex struct {
	session *concurrency.Session
	mu      *concurrency.Mutex
	key     string
}

// Lock blocks until the key can be acquired, or ctx is done.
func (c *Client) Lock(ctx context.Context, key string, ttl time.Duration) (*Mutex, error) {
	if ttl < time.Second {
		ttl = 10 * time.Second
	}
	sess, err := concurrency.NewSession(c.raw, concurrency.WithTTL(int(ttl/time.Second)))
	if err != nil {
		return nil, fmt.Errorf("etcd: new session: %w", err)
	}
	m := concurrency.NewMutex(sess, key)
	if err := m.Lock(ctx); err != nil {
		_ = sess.Close()
		return nil, fmt.Errorf("etcd: lock %s: %w", key, err)
	}
	return &Mutex{session: sess, mu: m, key: key}, nil
}

// TryLock attempts to acquire without blocking. Returns ErrNotAcquired when
// someone else holds the lock.
func (c *Client) TryLock(ctx context.Context, key string, ttl time.Duration) (*Mutex, error) {
	if ttl < time.Second {
		ttl = 10 * time.Second
	}
	sess, err := concurrency.NewSession(c.raw, concurrency.WithTTL(int(ttl/time.Second)))
	if err != nil {
		return nil, fmt.Errorf("etcd: new session: %w", err)
	}
	m := concurrency.NewMutex(sess, key)
	if err := m.TryLock(ctx); err != nil {
		_ = sess.Close()
		if errors.Is(err, concurrency.ErrLocked) {
			return nil, ErrNotAcquired
		}
		return nil, fmt.Errorf("etcd: trylock %s: %w", key, err)
	}
	return &Mutex{session: sess, mu: m, key: key}, nil
}

// Unlock releases the lock and closes the session.
func (m *Mutex) Unlock(ctx context.Context) error {
	defer m.session.Close()
	if err := m.mu.Unlock(ctx); err != nil {
		return fmt.Errorf("etcd: unlock %s: %w", m.key, err)
	}
	return nil
}

// ErrNotAcquired is returned by TryLock when the lock is held by someone else.
var ErrNotAcquired = errors.New("etcd: lock not acquired")

// ---------- Internal --------------------------------------------------------

func (c *Client) withTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	if c.opts.RequestTimeout <= 0 {
		return parent, func() {}
	}
	return context.WithTimeout(parent, c.opts.RequestTimeout)
}
