// Package mongo is a thin wrapper around the official MongoDB Go driver.
//
// It centralises connection construction with sensible defaults plus an
// explicit ping-on-open, and exposes a Pinger interface so the rest of the
// stack (graceful, health) can probe MongoDB.
//
// For everyday operations, use the embedded *mongo.Database directly:
//
//	c, err := mongox.Open(ctx, mongox.Options{
//	    URI:      "mongodb://localhost:27017",
//	    Database: "billing",
//	})
//	if err != nil { ... }
//	defer c.Close(ctx)
//
//	users := c.DB().Collection("users")
//	_ = users.InsertOne(ctx, bson.M{"name": "Ada"})
package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// Options configures Open.
type Options struct {
	// URI is the MongoDB connection string. Required.
	// Example: mongodb://user:pass@host1:27017,host2:27017/?replicaSet=rs0
	URI string

	// Database is the default database returned by DB(). Required.
	Database string

	// AppName is sent to MongoDB and shows up in the slow-query log.
	AppName string

	// ConnectTimeout caps the initial connect. Default 10s.
	ConnectTimeout time.Duration

	// PingTimeout caps the connectivity check after dial. Default 5s.
	// Zero disables the check.
	PingTimeout time.Duration

	// MaxPoolSize / MinPoolSize override the driver defaults.
	MaxPoolSize uint64
	MinPoolSize uint64
}

// Client wraps a *mongo.Client and pins a default database.
type Client struct {
	raw *mongo.Client
	db  *mongo.Database
}

// Open connects to MongoDB, verifies connectivity, and returns a Client.
func Open(ctx context.Context, opts Options) (*Client, error) {
	if opts.URI == "" {
		return nil, errors.New("mongo: URI is required")
	}
	if opts.Database == "" {
		return nil, errors.New("mongo: Database is required")
	}
	if opts.ConnectTimeout == 0 {
		opts.ConnectTimeout = 10 * time.Second
	}
	if opts.PingTimeout == 0 {
		opts.PingTimeout = 5 * time.Second
	}

	clientOpts := options.Client().ApplyURI(opts.URI).
		SetConnectTimeout(opts.ConnectTimeout)
	if opts.AppName != "" {
		clientOpts = clientOpts.SetAppName(opts.AppName)
	}
	if opts.MaxPoolSize > 0 {
		clientOpts = clientOpts.SetMaxPoolSize(opts.MaxPoolSize)
	}
	if opts.MinPoolSize > 0 {
		clientOpts = clientOpts.SetMinPoolSize(opts.MinPoolSize)
	}

	cli, err := mongo.Connect(clientOpts)
	if err != nil {
		return nil, fmt.Errorf("mongo: connect: %w", err)
	}

	if opts.PingTimeout > 0 {
		pingCtx, cancel := context.WithTimeout(ctx, opts.PingTimeout)
		defer cancel()
		if err := cli.Ping(pingCtx, readpref.Primary()); err != nil {
			_ = cli.Disconnect(context.Background())
			return nil, fmt.Errorf("mongo: ping: %w", err)
		}
	}
	return &Client{raw: cli, db: cli.Database(opts.Database)}, nil
}

// Raw returns the underlying *mongo.Client for advanced features (sessions,
// transactions, multi-DB workloads).
func (c *Client) Raw() *mongo.Client { return c.raw }

// DB returns the pinned default database.
func (c *Client) DB() *mongo.Database { return c.db }

// Collection is a shortcut for c.DB().Collection(name).
func (c *Client) Collection(name string) *mongo.Collection {
	return c.db.Collection(name)
}

// Ping verifies the primary is reachable.
func (c *Client) Ping(ctx context.Context) error {
	if err := c.raw.Ping(ctx, readpref.Primary()); err != nil {
		return fmt.Errorf("mongo: ping: %w", err)
	}
	return nil
}

// Close disconnects from the cluster.
func (c *Client) Close(ctx context.Context) error {
	if err := c.raw.Disconnect(ctx); err != nil {
		return fmt.Errorf("mongo: disconnect: %w", err)
	}
	return nil
}

// IsDuplicateKey reports whether err is a duplicate-key (E11000) error.
func IsDuplicateKey(err error) bool {
	return mongo.IsDuplicateKeyError(err)
}

// IsNoDocuments reports whether err is the canonical "no documents in result"
// error returned by FindOne / similar.
func IsNoDocuments(err error) bool {
	return errors.Is(err, mongo.ErrNoDocuments)
}
