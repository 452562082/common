// Package zk wraps the go-zookeeper client for two common patterns:
//
//   - Client watches a single node and pushes data/children changes onto
//     buffered channels, conflating stale values rather than blocking.
//   - Server publishes config (persistent) and registry (ephemeral) nodes,
//     transparently re-creating them after session loss.
//
// Both types log connection-lifecycle events via log/slog and wrap underlying
// errors with %w.
package zk
