package zk

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/go-zookeeper/zk"
)

const (
	clientDefaultSessionTimeout = 2 * time.Second
	clientChannelBuffer         = 1
	clientWatchBackoff          = time.Second
)

// Client watches a single ZooKeeper node for data and children changes.
//
// Both watcher goroutines push the latest value onto buffered channels each
// time it changes. If a consumer is slow, the latest value is conflated
// (older buffered values are dropped) rather than blocking the watcher.
type Client struct {
	path     string
	conn     *zk.Conn
	data     chan []byte
	children chan []string

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewClient connects to zkHosts, ensures nodePath exists (creating it with
// defaultData when writeDefault is true), and starts watcher goroutines.
func NewClient(zkHosts []string, nodePath string, defaultData []byte, writeDefault bool) (*Client, error) {
	nodePath = "/" + strings.Trim(nodePath, "/")

	conn, _, err := zk.Connect(zkHosts, clientDefaultSessionTimeout)
	if err != nil {
		return nil, fmt.Errorf("zk: connect: %w", err)
	}

	exist, _, err := conn.Exists(nodePath)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("zk: exists %s: %w", nodePath, err)
	}
	if !exist {
		if !writeDefault {
			conn.Close()
			return nil, fmt.Errorf("zk: node %s does not exist", nodePath)
		}
		if _, err = conn.Create(nodePath, defaultData, 0, zk.WorldACL(zk.PermAll)); err != nil && !errors.Is(err, zk.ErrNodeExists) {
			conn.Close()
			return nil, fmt.Errorf("zk: create %s: %w", nodePath, err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	c := &Client{
		path:     nodePath,
		conn:     conn,
		data:     make(chan []byte, clientChannelBuffer),
		children: make(chan []string, clientChannelBuffer),
		ctx:      ctx,
		cancel:   cancel,
	}

	c.wg.Add(2)
	go c.watchData()
	go c.watchChildren()
	return c, nil
}

func (c *Client) String() string {
	return fmt.Sprintf("zk.Client sid=%d path=%s", c.conn.SessionID(), c.path)
}

// Data returns a channel that receives the node's data on every change.
// Old values are dropped if the consumer cannot keep up.
func (c *Client) Data() <-chan []byte { return c.data }

// Children returns a channel that receives the node's children on every change.
// Old values are dropped if the consumer cannot keep up.
func (c *Client) Children() <-chan []string { return c.children }

// ChildrenOnce reads the current children list without setting a watch.
func (c *Client) ChildrenOnce(node string) ([]string, *zk.Stat, error) {
	return c.conn.Children(node)
}

// Close stops the watcher goroutines and tears down the ZK connection.
// It is safe to call multiple times. The error return is reserved for future
// drivers that report close failures; the current implementation always
// returns nil.
func (c *Client) Close() error {
	c.cancel()
	c.conn.Close()
	c.wg.Wait()
	return nil
}

// publishData pushes v onto the data channel, dropping a stale buffered value
// rather than blocking. Returns false if the client is shutting down.
func (c *Client) publishData(v []byte) bool {
	for {
		select {
		case <-c.ctx.Done():
			return false
		case c.data <- v:
			return true
		default:
			// channel full; drop the oldest buffered value and retry.
			select {
			case <-c.data:
			default:
			}
		}
	}
}

func (c *Client) publishChildren(v []string) bool {
	for {
		select {
		case <-c.ctx.Done():
			return false
		case c.children <- v:
			return true
		default:
			select {
			case <-c.children:
			default:
			}
		}
	}
}

func (c *Client) watchData() {
	defer c.wg.Done()
	for {
		if c.ctx.Err() != nil {
			return
		}
		data, _, events, err := c.conn.GetW(c.path)
		if err != nil {
			slog.Error("zk: watch data", "path", c.path, "err", err)
			if !c.sleep(clientWatchBackoff) {
				return
			}
			continue
		}
		if !c.publishData(data) {
			return
		}

		select {
		case <-c.ctx.Done():
			return
		case evt := <-events:
			if evt.Err != nil {
				slog.Error("zk: data watch event", "path", c.path, "err", evt.Err)
				continue
			}
			// Any node-data change reaches here; the next GetW re-arms the watch
			// and re-fetches the value, so non-data events fall through to the
			// next iteration safely.
			_ = evt
		}
	}
}

func (c *Client) watchChildren() {
	defer c.wg.Done()
	for {
		if c.ctx.Err() != nil {
			return
		}
		children, _, events, err := c.conn.ChildrenW(c.path)
		if err != nil {
			slog.Error("zk: watch children", "path", c.path, "err", err)
			if !c.sleep(clientWatchBackoff) {
				return
			}
			continue
		}
		if !c.publishChildren(children) {
			return
		}

		select {
		case <-c.ctx.Done():
			return
		case evt := <-events:
			if evt.Err != nil {
				slog.Error("zk: children watch event", "path", c.path, "err", evt.Err)
				continue
			}
			_ = evt
		}
	}
}

func (c *Client) sleep(d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-c.ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
