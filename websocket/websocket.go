// Package websocket layers a small Conn + Hub abstraction on top of
// gorilla/websocket.
//
// Conn:
//   - Per-connection write goroutine with a bounded send channel; the rest of
//     the codebase enqueues bytes via Send without worrying about gorilla's
//     concurrent-write restriction.
//   - Built-in ping/pong keepalive — the connection closes automatically
//     when the peer goes silent.
//
// Hub:
//   - Tracks connected clients, broadcasts to all or to a subset.
//   - Drops slow consumers that can't keep up with the broadcast rate
//     (back-pressure has no good answer for fan-out).
package websocket

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Options for the underlying gorilla Upgrader.
type Options struct {
	ReadBufferSize  int
	WriteBufferSize int

	// AllowedOrigins whitelists Origin header values (browser-facing case).
	// Each entry is matched against the request's Origin header exactly.
	// The special value "*" allows all origins — convenient but disables
	// Cross-Site WebSocket Hijacking protection, so use it deliberately.
	//
	// If both AllowedOrigins and CheckOrigin are unset, browser-style upgrades
	// (those that send an Origin header) are REJECTED. Non-browser clients
	// that omit Origin (gRPC bridges, server-to-server, native apps) are
	// always allowed.
	AllowedOrigins []string

	// CheckOrigin overrides AllowedOrigins with arbitrary logic.
	CheckOrigin func(r *http.Request) bool

	// SendBuffer is the per-connection outbound queue size. Default 64.
	SendBuffer int

	// PingPeriod is the ping cadence. Default 30s.
	PingPeriod time.Duration

	// PongTimeout — peer must respond within this. Default 60s.
	PongTimeout time.Duration

	// WriteTimeout caps each write call. Default 10s.
	WriteTimeout time.Duration

	// MaxMessageSize caps inbound frames. Default 1 MiB.
	MaxMessageSize int64
}

// buildCheckOrigin returns the gorilla CheckOrigin callback applied when the
// caller didn't supply one. The default policy is:
//
//   - Requests without an Origin header (non-browser clients) pass through.
//   - Origin "*" in AllowedOrigins disables checking entirely.
//   - Otherwise the Origin must exactly match one of AllowedOrigins.
//
// This is conservative on purpose: browsers always send Origin, so any
// cross-site connection attempt is rejected unless the operator opts in.
func buildCheckOrigin(allowed []string) func(*http.Request) bool {
	allowAll := false
	set := make(map[string]struct{}, len(allowed))
	for _, o := range allowed {
		if o == "*" {
			allowAll = true
			continue
		}
		set[o] = struct{}{}
	}
	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // non-browser client
		}
		if allowAll {
			return true
		}
		_, ok := set[origin]
		return ok
	}
}

// Server bundles an Upgrader and the runtime options shared by every Conn it
// creates.
type Server struct {
	upgrader websocket.Upgrader
	opts     Options
}

// NewServer returns a Server with sensible defaults applied.
func NewServer(opts Options) *Server {
	applyDefaults(&opts)
	return &Server{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  opts.ReadBufferSize,
			WriteBufferSize: opts.WriteBufferSize,
			CheckOrigin:     opts.CheckOrigin,
		},
		opts: opts,
	}
}

func applyDefaults(o *Options) {
	if o.ReadBufferSize == 0 {
		o.ReadBufferSize = 1024
	}
	if o.WriteBufferSize == 0 {
		o.WriteBufferSize = 1024
	}
	if o.CheckOrigin == nil {
		o.CheckOrigin = buildCheckOrigin(o.AllowedOrigins)
	}
	if o.SendBuffer == 0 {
		o.SendBuffer = 64
	}
	if o.PingPeriod == 0 {
		o.PingPeriod = 30 * time.Second
	}
	if o.PongTimeout == 0 {
		o.PongTimeout = 60 * time.Second
	}
	if o.WriteTimeout == 0 {
		o.WriteTimeout = 10 * time.Second
	}
	if o.MaxMessageSize == 0 {
		o.MaxMessageSize = 1 << 20 // 1 MiB
	}
}

// Upgrade upgrades an HTTP request to a Conn. The caller is responsible for
// invoking Conn.Close when done with it.
func (s *Server) Upgrade(w http.ResponseWriter, r *http.Request, respHeader http.Header) (*Conn, error) {
	ws, err := s.upgrader.Upgrade(w, r, respHeader)
	if err != nil {
		return nil, fmt.Errorf("websocket: upgrade: %w", err)
	}
	ws.SetReadLimit(s.opts.MaxMessageSize)
	_ = ws.SetReadDeadline(time.Now().Add(s.opts.PongTimeout))
	ws.SetPongHandler(func(string) error {
		_ = ws.SetReadDeadline(time.Now().Add(s.opts.PongTimeout))
		return nil
	})
	c := &Conn{
		ws:    ws,
		send:  make(chan []byte, s.opts.SendBuffer),
		opts:  s.opts,
		close: make(chan struct{}),
	}
	go c.writeLoop()
	return c, nil
}

// Conn is a single client connection. All sends go through a single goroutine
// so the caller never has to coordinate writes.
type Conn struct {
	ws    *websocket.Conn
	send  chan []byte
	opts  Options
	once  sync.Once
	close chan struct{}
}

// Send queues data for delivery. Returns ErrClosed if the connection has been
// closed, or ErrSlowConsumer if the send queue is full (the connection is
// then closed and the next call returns ErrClosed).
func (c *Conn) Send(data []byte) error {
	select {
	case <-c.close:
		return ErrClosed
	default:
	}
	select {
	case c.send <- data:
		return nil
	default:
		_ = c.Close()
		return ErrSlowConsumer
	}
}

// ReadMessage reads the next data frame from the peer.
// Returns ErrClosed once the connection is shut down.
func (c *Conn) ReadMessage() (messageType int, payload []byte, err error) {
	mt, p, err := c.ws.ReadMessage()
	if err != nil {
		return mt, p, err
	}
	return mt, p, nil
}

// Close terminates the connection. Safe to call multiple times.
func (c *Conn) Close() error {
	var firstErr error
	c.once.Do(func() {
		close(c.close)
		_ = c.ws.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(c.opts.WriteTimeout))
		if err := c.ws.Close(); err != nil {
			firstErr = fmt.Errorf("websocket: close: %w", err)
		}
	})
	return firstErr
}

// RemoteAddr returns the peer address (useful for logging).
func (c *Conn) RemoteAddr() string { return c.ws.RemoteAddr().String() }

func (c *Conn) writeLoop() {
	ticker := time.NewTicker(c.opts.PingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-c.close:
			return
		case data, ok := <-c.send:
			if !ok {
				return
			}
			_ = c.ws.SetWriteDeadline(time.Now().Add(c.opts.WriteTimeout))
			if err := c.ws.WriteMessage(websocket.TextMessage, data); err != nil {
				_ = c.Close()
				return
			}
		case <-ticker.C:
			_ = c.ws.SetWriteDeadline(time.Now().Add(c.opts.WriteTimeout))
			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				_ = c.Close()
				return
			}
		}
	}
}

// ---------- Hub --------------------------------------------------------------

// Hub tracks a set of Conns and fans out broadcasts.
type Hub struct {
	mu      sync.RWMutex
	clients map[*Conn]struct{}
}

// NewHub returns an empty Hub.
func NewHub() *Hub { return &Hub{clients: make(map[*Conn]struct{})} }

// Add registers c with the hub.
func (h *Hub) Add(c *Conn) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

// Remove unregisters c. Calling Remove on a missing Conn is a no-op.
func (h *Hub) Remove(c *Conn) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
}

// Len returns the number of connected clients.
func (h *Hub) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// Broadcast sends data to every connected client. Slow consumers are dropped
// (their Send returns ErrSlowConsumer which closes their conn).
//
// Returns the number of clients reached and the number dropped due to back-pressure.
func (h *Hub) Broadcast(data []byte) (sent, dropped int) {
	h.mu.RLock()
	clients := make([]*Conn, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.RUnlock()
	for _, c := range clients {
		if err := c.Send(data); err != nil {
			h.Remove(c)
			dropped++
			continue
		}
		sent++
	}
	return sent, dropped
}

// ErrClosed signals operations on a closed connection.
var ErrClosed = errors.New("websocket: connection closed")

// ErrSlowConsumer signals that the per-conn send queue overflowed.
var ErrSlowConsumer = errors.New("websocket: slow consumer")
