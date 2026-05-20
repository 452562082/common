package zk

import (
	"errors"
	"fmt"
	"log/slog"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/go-zookeeper/zk"
)

const defaultSessionTimeout = 2 * time.Second

type createType int32

const (
	createNodeData     createType = 0
	createNodeChildren createType = 1
)

type znode struct {
	typ    createType
	path   string
	data   []byte
	active bool
}

// Server manages ZooKeeper service-registration and config-publishing nodes,
// re-creating them automatically after session loss.
type Server struct {
	conn        *zk.Conn
	events      <-chan zk.Event
	registryMap map[string]*znode
	mu          sync.RWMutex
	done        chan struct{}
}

// NewServer connects to the given ZK ensemble and returns a Server ready to
// publish config / register ephemeral nodes. A background goroutine watches
// for session events and re-creates registered nodes on reconnect.
func NewServer(zkHosts []string) (*Server, error) {
	conn, ev, err := zk.Connect(zkHosts, defaultSessionTimeout)
	if err != nil {
		return nil, fmt.Errorf("zk: connect: %w", err)
	}
	s := &Server{
		conn:        conn,
		events:      ev,
		registryMap: make(map[string]*znode),
		done:        make(chan struct{}),
	}
	go s.loop()
	return s, nil
}

func (s *Server) String() string {
	return fmt.Sprintf("zk.Server sid=%d", s.conn.SessionID())
}

// Close terminates the watcher loop and tears down the ZK connection.
// Returns nil; the error return exists for io.Closer-style symmetry.
func (s *Server) Close() error {
	close(s.done)
	s.conn.Close()
	return nil
}

func (s *Server) loop() {
	for {
		select {
		case <-s.done:
			return
		case evt, ok := <-s.events:
			if !ok {
				return
			}
			if evt.Type != zk.EventSession {
				continue
			}
			s.handleSessionEvent(evt)
		}
	}
}

func (s *Server) handleSessionEvent(evt zk.Event) {
	switch evt.State {
	case zk.StateHasSession:
		slog.Info("zk: session ready", "server", s.conn.Server())
		s.mu.Lock()
		defer s.mu.Unlock()
		for p, node := range s.registryMap {
			if node.active {
				continue
			}
			switch node.typ {
			case createNodeData:
				if err := s.serviceConfig(p, node.data, true); err != nil {
					slog.Error("zk: recreate config node", "path", p, "err", err)
					continue
				}
			case createNodeChildren:
				parts := strings.Split(p, "/")
				root := strings.Join(parts[:len(parts)-1], "/")
				if err := s.serviceRegistry(root, parts[len(parts)-1], node.data, true); err != nil {
					slog.Error("zk: recreate ephemeral node", "path", p, "err", err)
					continue
				}
			}
			slog.Info("zk: recreated node", "path", p)
		}
	case zk.StateDisconnected:
		slog.Warn("zk: disconnected", "server", s.conn.Server())
		s.mu.Lock()
		for _, node := range s.registryMap {
			node.active = false
		}
		s.mu.Unlock()
	case zk.StateConnected:
		slog.Info("zk: connected", "server", s.conn.Server())
	case zk.StateExpired:
		slog.Warn("zk: session expired", "server", s.conn.Server())
	}
}

// PublishConfig creates a persistent node holding configuration data.
// Clients can watch this path to receive config updates.
// If createParents is true, missing parent nodes are created automatically.
func (s *Server) PublishConfig(servicePath string, data []byte, createParents bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.serviceConfig(servicePath, data, createParents)
}

func (s *Server) serviceConfig(servicePath string, data []byte, createParents bool) error {
	servicePath = path.Clean(servicePath)
	if err := s.ensureParents(servicePath, createParents); err != nil {
		return err
	}

	if existing, ok := s.registryMap[servicePath]; ok && existing.active {
		return nil
	}
	s.registryMap[servicePath] = &znode{
		typ:  createNodeData,
		path: servicePath,
		data: data,
	}

	_, err := s.conn.Create(servicePath, data, 0, zk.WorldACL(zk.PermAll))
	if err != nil {
		if errors.Is(err, zk.ErrNodeExists) {
			if _, err = s.conn.Set(servicePath, data, -1); err != nil {
				return fmt.Errorf("zk: set %s: %w", servicePath, err)
			}
		} else {
			return fmt.Errorf("zk: create %s: %w", servicePath, err)
		}
	}
	s.registryMap[servicePath].active = true
	return nil
}

// Register creates an ephemeral child node under rootPath, typically used for
// service discovery (the node disappears when the session ends).
// If createParents is true, missing parent nodes are created automatically.
func (s *Server) Register(rootPath, serviceHost string, data []byte, createParents bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.serviceRegistry(rootPath, serviceHost, data, createParents)
}

func (s *Server) serviceRegistry(rootPath, serviceHost string, data []byte, createParents bool) error {
	abs := path.Join(rootPath, serviceHost)
	if err := s.ensureParents(abs, createParents); err != nil {
		return err
	}

	if existing, ok := s.registryMap[abs]; ok && existing.active {
		return nil
	}
	s.registryMap[abs] = &znode{
		typ:  createNodeChildren,
		path: abs,
		data: data,
	}

	if _, err := s.conn.Create(abs, data, zk.FlagEphemeral, zk.WorldACL(zk.PermAll)); err != nil {
		return fmt.Errorf("zk: create ephemeral %s: %w", abs, err)
	}
	s.registryMap[abs].active = true
	return nil
}

func (s *Server) ensureParents(p string, create bool) error {
	for _, parent := range parentPaths(p) {
		exist, _, err := s.conn.Exists(parent)
		if err != nil {
			return fmt.Errorf("zk: exists %s: %w", parent, err)
		}
		if exist {
			continue
		}
		if !create {
			return fmt.Errorf("zk: missing parent node %s", parent)
		}
		if _, err = s.conn.Create(parent, []byte(parent), 0, zk.WorldACL(zk.PermAll)); err != nil && !errors.Is(err, zk.ErrNodeExists) {
			return fmt.Errorf("zk: create parent %s: %w", parent, err)
		}
	}
	return nil
}

func parentPaths(p string) []string {
	p = "/" + strings.Trim(p, "/")
	parts := strings.Split(p, "/")
	if len(parts) <= 2 {
		return nil
	}
	out := make([]string, 0, len(parts)-2)
	for i := 1; i < len(parts)-1; i++ {
		out = append(out, "/"+strings.Join(parts[1:i+1], "/"))
	}
	return out
}
