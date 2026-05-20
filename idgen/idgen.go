// Package idgen offers three flavours of unique ID generation:
//
//   - UUID v4 — random; the most portable choice.
//   - UUID v7 — time-ordered; sortable, friendlier for B-tree indexes.
//   - Snowflake — Twitter-style 64-bit; compact and monotonic per node.
//
// UUID helpers wrap google/uuid. The Snowflake helper wraps bwmarrin/snowflake
// and lets you stamp out IDs from a configured node ID (0–1023).
package idgen

import (
	"fmt"

	"github.com/bwmarrin/snowflake"
	"github.com/google/uuid"
)

// UUID returns a new UUID v4 (random) string in canonical form.
func UUID() string {
	return uuid.NewString()
}

// UUIDv7 returns a new UUID v7 (time-ordered) string.
//
// v7 IDs sort lexicographically by creation time, which is friendlier for
// database indexes than v4. Falls back to UUID() if the local entropy source
// returns an error (extremely unlikely on any production system).
func UUIDv7() string {
	id, err := uuid.NewV7()
	if err != nil {
		return UUID()
	}
	return id.String()
}

// Snowflake is a goroutine-safe ID generator producing 64-bit IDs unique
// across machines with distinct NodeIDs.
type Snowflake struct {
	node *snowflake.Node
}

// NewSnowflake returns a Snowflake bound to the given node ID (0–1023).
func NewSnowflake(nodeID int64) (*Snowflake, error) {
	if nodeID < 0 || nodeID > 1023 {
		return nil, fmt.Errorf("idgen: nodeID %d out of range [0, 1023]", nodeID)
	}
	n, err := snowflake.NewNode(nodeID)
	if err != nil {
		return nil, fmt.Errorf("idgen: new snowflake node: %w", err)
	}
	return &Snowflake{node: n}, nil
}

// Next returns the next Snowflake ID as an int64.
func (s *Snowflake) Next() int64 {
	return s.node.Generate().Int64()
}

// NextString returns the next Snowflake ID formatted as a decimal string.
func (s *Snowflake) NextString() string {
	return s.node.Generate().String()
}

// NextBase58 returns the next Snowflake ID base58-encoded. Useful for
// URL-safe shorter strings.
func (s *Snowflake) NextBase58() string {
	return s.node.Generate().Base58()
}
