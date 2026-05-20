// Package nats wraps the official nats.go client with a small, opinionated
// surface for two patterns:
//
//   - Core NATS — fire-and-forget pub/sub. Best when at-most-once delivery
//     and very low overhead are what you need.
//   - JetStream — persistent streams with at-least-once delivery and
//     consumer offsets, similar to Kafka topics + consumer groups but
//     embedded in NATS.
//
// Both use the same *Client and share the same connection.
//
//	c, err := natsx.Open(natsx.Options{URL: "nats://localhost:4222"})
//	if err != nil { ... }
//	defer c.Close()
//
//	// Core pub/sub
//	sub, _ := c.Subscribe(ctx, "billing.events", func(_ context.Context, m *nats.Msg) {
//	    fmt.Println("got", string(m.Data))
//	})
//	defer sub.Unsubscribe()
//	_ = c.Publish(ctx, "billing.events", []byte("hello"))
package nats

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Options configures Open.
type Options struct {
	// URL is the NATS server URL. Required.
	// Example: "nats://localhost:4222" or "nats://u:p@nats:4222".
	URL string

	// Name labels the connection in NATS server logs.
	Name string

	// MaxReconnects defaults to -1 (forever). Zero falls back to that default.
	MaxReconnects int

	// ReconnectWait is the delay between reconnect attempts. Default 2s.
	ReconnectWait time.Duration

	// PingInterval is the server-ping cadence. Default 2 minutes.
	PingInterval time.Duration

	// Username / Password / Token for auth. Mutually exclusive with NKey/JWT,
	// which can be plugged in via ExtraOpts.
	Username string
	Password string
	Token    string

	// ExtraOpts are appended to the nats.Connect arg list, so callers can
	// supply TLS / NKey / custom handlers.
	ExtraOpts []nats.Option
}

// Client wraps a *nats.Conn and (lazily) a JetStream context.
type Client struct {
	conn *nats.Conn
	js   jetstream.JetStream
}

// Open dials the NATS server and returns a Client.
func Open(opts Options) (*Client, error) {
	if opts.URL == "" {
		return nil, errors.New("nats: URL is required")
	}
	if opts.MaxReconnects == 0 {
		opts.MaxReconnects = -1
	}
	if opts.ReconnectWait == 0 {
		opts.ReconnectWait = 2 * time.Second
	}
	if opts.PingInterval == 0 {
		opts.PingInterval = 2 * time.Minute
	}

	natsOpts := []nats.Option{
		nats.MaxReconnects(opts.MaxReconnects),
		nats.ReconnectWait(opts.ReconnectWait),
		nats.PingInterval(opts.PingInterval),
	}
	if opts.Name != "" {
		natsOpts = append(natsOpts, nats.Name(opts.Name))
	}
	switch {
	case opts.Token != "":
		natsOpts = append(natsOpts, nats.Token(opts.Token))
	case opts.Username != "" || opts.Password != "":
		natsOpts = append(natsOpts, nats.UserInfo(opts.Username, opts.Password))
	}
	natsOpts = append(natsOpts, opts.ExtraOpts...)

	conn, err := nats.Connect(opts.URL, natsOpts...)
	if err != nil {
		return nil, fmt.Errorf("nats: connect: %w", err)
	}
	return &Client{conn: conn}, nil
}

// Raw exposes the underlying *nats.Conn for any feature not surfaced here.
func (c *Client) Raw() *nats.Conn { return c.conn }

// Close drains in-flight messages, then closes the connection. Safe to call
// multiple times.
func (c *Client) Close() error {
	if c.conn.IsClosed() {
		return nil
	}
	if err := c.conn.Drain(); err != nil {
		return fmt.Errorf("nats: drain: %w", err)
	}
	return nil
}

// IsConnected reports whether the underlying connection is live.
func (c *Client) IsConnected() bool { return c.conn.IsConnected() }

// ---------- Core pub/sub ----------------------------------------------------

// MsgHandler processes one delivered message. ctx is the parent context the
// caller supplied to Subscribe.
type MsgHandler func(ctx context.Context, msg *nats.Msg)

// Publish sends a single message on subject.
func (c *Client) Publish(_ context.Context, subject string, data []byte) error {
	if err := c.conn.Publish(subject, data); err != nil {
		return fmt.Errorf("nats: publish %s: %w", subject, err)
	}
	return nil
}

// Request publishes a message and waits for a reply, bounded by ctx.
// Use context.WithTimeout if you want a per-request deadline.
func (c *Client) Request(ctx context.Context, subject string, data []byte) (*nats.Msg, error) {
	msg, err := c.conn.RequestWithContext(ctx, subject, data)
	if err != nil {
		return nil, fmt.Errorf("nats: request %s: %w", subject, err)
	}
	return msg, nil
}

// Subscribe registers a callback for subject. Use the returned *nats.Subscription
// to Unsubscribe.
//
// The callback is wrapped in a panic recover — a single bad message must
// never tear down nats's internal dispatch goroutines.
func (c *Client) Subscribe(ctx context.Context, subject string, fn MsgHandler) (*nats.Subscription, error) {
	sub, err := c.conn.Subscribe(subject, safeHandler(ctx, subject, fn))
	if err != nil {
		return nil, fmt.Errorf("nats: subscribe %s: %w", subject, err)
	}
	return sub, nil
}

// QueueSubscribe is the load-balanced variant: every message goes to exactly
// one subscriber per queue group.
func (c *Client) QueueSubscribe(ctx context.Context, subject, queue string, fn MsgHandler) (*nats.Subscription, error) {
	sub, err := c.conn.QueueSubscribe(subject, queue, safeHandler(ctx, subject, fn))
	if err != nil {
		return nil, fmt.Errorf("nats: queue-subscribe %s: %w", subject, err)
	}
	return sub, nil
}

// safeHandler wraps a user MsgHandler so a panic inside it is logged and
// swallowed instead of taking down nats's internal dispatch goroutine.
func safeHandler(ctx context.Context, subject string, fn MsgHandler) func(*nats.Msg) {
	return func(m *nats.Msg) {
		defer func() {
			if r := recover(); r != nil {
				slog.ErrorContext(ctx, "nats: handler panic",
					"subject", subject,
					"err", r,
				)
			}
		}()
		fn(ctx, m)
	}
}

// ---------- JetStream -------------------------------------------------------

// JetStream returns a JetStream handle, lazily creating it on first use.
func (c *Client) JetStream() (jetstream.JetStream, error) {
	if c.js != nil {
		return c.js, nil
	}
	js, err := jetstream.New(c.conn)
	if err != nil {
		return nil, fmt.Errorf("nats: jetstream: %w", err)
	}
	c.js = js
	return c.js, nil
}

// JSPublish publishes durably to a JetStream subject.
func (c *Client) JSPublish(ctx context.Context, subject string, data []byte) (*jetstream.PubAck, error) {
	js, err := c.JetStream()
	if err != nil {
		return nil, err
	}
	ack, err := js.Publish(ctx, subject, data)
	if err != nil {
		return nil, fmt.Errorf("nats: js publish %s: %w", subject, err)
	}
	return ack, nil
}

// ---------- Helpers ---------------------------------------------------------

