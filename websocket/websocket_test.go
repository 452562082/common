package websocket

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
)

func echoHandler(t *testing.T, s *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := s.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		go func() {
			defer c.Close()
			for {
				_, p, err := c.ReadMessage()
				if err != nil {
					return
				}
				if err := c.Send(p); err != nil {
					return
				}
			}
		}()
	}
}

func TestConn_EchoRoundTrip(t *testing.T) {
	s := NewServer(Options{})
	srv := httptest.NewServer(echoHandler(t, s))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"
	c, _, err := gws.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()

	if err := c.WriteMessage(gws.TextMessage, []byte("hello")); err != nil {
		t.Fatal(err)
	}
	_, p, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(p) != "hello" {
		t.Errorf("echo body = %q", p)
	}
}

func TestConn_CloseIdempotent(t *testing.T) {
	s := NewServer(Options{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := s.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		// Two Close calls — must not panic or error.
		if err := c.Close(); err != nil {
			t.Errorf("first Close: %v", err)
		}
		if err := c.Close(); err != nil {
			t.Errorf("second Close: %v", err)
		}
	}))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"
	c, _, err := gws.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
}

func TestConn_SendAfterClose(t *testing.T) {
	s := NewServer(Options{})
	got := make(chan error, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := s.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		_ = c.Close()
		got <- c.Send([]byte("nope"))
	}))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"
	c, _, err := gws.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	select {
	case err := <-got:
		if !errors.Is(err, ErrClosed) {
			t.Errorf("expected ErrClosed, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for handler")
	}
}

func TestCheckOrigin_DefaultRejectsBrowserCrossSite(t *testing.T) {
	cb := buildCheckOrigin(nil) // default: no AllowedOrigins

	// Non-browser client (no Origin) — must pass.
	r := httptest.NewRequest("GET", "/ws", nil)
	if !cb(r) {
		t.Error("client without Origin should be allowed")
	}

	// Browser cross-site — must be rejected.
	r = httptest.NewRequest("GET", "/ws", nil)
	r.Header.Set("Origin", "https://evil.example")
	if cb(r) {
		t.Error("unknown Origin should be rejected by default")
	}
}

func TestCheckOrigin_AllowedOrigins(t *testing.T) {
	cb := buildCheckOrigin([]string{"https://app.example.com"})

	r := httptest.NewRequest("GET", "/ws", nil)
	r.Header.Set("Origin", "https://app.example.com")
	if !cb(r) {
		t.Error("whitelisted origin must pass")
	}

	r.Header.Set("Origin", "https://evil.example")
	if cb(r) {
		t.Error("non-whitelisted origin must fail")
	}
}

func TestCheckOrigin_Wildcard(t *testing.T) {
	cb := buildCheckOrigin([]string{"*"})
	r := httptest.NewRequest("GET", "/ws", nil)
	r.Header.Set("Origin", "https://anything.example")
	if !cb(r) {
		t.Error("'*' should allow any origin")
	}
}

func TestUpgrade_BrowserCrossSiteRejected(t *testing.T) {
	s := NewServer(Options{}) // default options — no AllowedOrigins
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := s.Upgrade(w, r, nil)
		if err == nil {
			t.Error("upgrade should have been rejected")
		}
	}))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"
	header := http.Header{}
	header.Set("Origin", "https://evil.example")
	_, _, err := gws.DefaultDialer.Dial(url, header)
	if err == nil {
		t.Error("dial with foreign Origin should fail")
	}
}

func TestHub_BroadcastAndRemove(t *testing.T) {
	s := NewServer(Options{})
	hub := NewHub()

	var mu sync.Mutex
	received := map[string][][]byte{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := s.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		hub.Add(c)
	}))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"
	dial := func(name string) *gws.Conn {
		c, _, err := gws.DefaultDialer.Dial(url, nil)
		if err != nil {
			t.Fatalf("dial %s: %v", name, err)
		}
		go func() {
			for {
				_, p, err := c.ReadMessage()
				if err != nil {
					return
				}
				mu.Lock()
				received[name] = append(received[name], p)
				mu.Unlock()
			}
		}()
		return c
	}

	a := dial("a")
	defer a.Close()
	b := dial("b")
	defer b.Close()

	// Wait until both connections are registered with the hub.
	for i := 0; i < 50 && hub.Len() < 2; i++ {
		time.Sleep(10 * time.Millisecond)
	}
	if hub.Len() != 2 {
		t.Fatalf("hub size = %d", hub.Len())
	}

	sent, dropped := hub.Broadcast([]byte("hi"))
	if sent != 2 || dropped != 0 {
		t.Errorf("Broadcast = (%d, %d), want (2, 0)", sent, dropped)
	}

	// Give the writes a moment to flush.
	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	gotA, gotB := len(received["a"]), len(received["b"])
	mu.Unlock()
	if gotA == 0 || gotB == 0 {
		t.Errorf("not all clients got the message: a=%d b=%d", gotA, gotB)
	}
}
