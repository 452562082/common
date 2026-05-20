// Package lru provides a generic, size+TTL bounded LRU cache.
//
// The Cache interface is parameterised on a comparable key type and an
// arbitrary value type. Two implementations are shipped:
//
//   - MemCache: in-process LRU backed by container/list + map.
//   - RedisCache: Redis-backed LRU using a sorted-set + hash (see lru_cache_redis.go).
//
// All operations on RedisCache accept a context.Context. MemCache ignores
// ctx for symmetry with the interface so callers can swap implementations
// freely.
package lru

import (
	"container/list"
	"context"
	"sync"
	"time"
)

// Cache is the common interface implemented by both in-memory and Redis-backed caches.
type Cache[K comparable, V any] interface {
	// Get returns the value for key (and true) or the zero value (and false)
	// when the key is absent or expired. Touching the entry moves it to the
	// front of the LRU.
	Get(ctx context.Context, key K) (V, bool, error)

	// Peek returns the value without touching the LRU.
	Peek(ctx context.Context, key K) (V, bool, error)

	// Set inserts or updates a key. If the cache is full, the least-recently
	// used entry is evicted.
	Set(ctx context.Context, key K, value V) error

	// Delete removes one or more keys. Missing keys are silently ignored.
	Delete(ctx context.Context, keys ...K) error

	// Len reports the current number of entries.
	Len(ctx context.Context) (int, error)

	// Range invokes fn for every live (non-expired) entry. If fn returns
	// false, iteration stops. Iteration order is unspecified.
	Range(ctx context.Context, fn func(K, V) bool) error

	// Clear removes every entry.
	Clear(ctx context.Context) error

	// PurgeExpired drops entries whose TTL has elapsed. A no-op when TTL is 0.
	PurgeExpired(ctx context.Context) error

	// Close releases any underlying resources (network connections, etc.).
	Close() error
}

// EvictionCallback is invoked when an entry is evicted (capacity, manual delete,
// or TTL expiry). Use it for metrics or cascading cleanup.
type EvictionCallback[K comparable, V any] func(key K, value V)

// MemOptions configures NewMemCache.
type MemOptions[K comparable, V any] struct {
	// Capacity is the maximum number of entries before LRU eviction kicks in.
	// 0 means unbounded.
	Capacity int

	// TTL is the maximum age of an entry. 0 means entries never expire.
	TTL time.Duration

	// OnEvict, if set, is invoked whenever an entry leaves the cache.
	OnEvict EvictionCallback[K, V]
}

// MemCache is a size+TTL bounded in-process LRU.
type MemCache[K comparable, V any] struct {
	opts MemOptions[K, V]
	mu   sync.RWMutex
	ll   *list.List
	idx  map[K]*list.Element
}

type memEntry[K comparable, V any] struct {
	key   K
	value V
	added time.Time
}

// NewMemCache returns a new in-process LRU cache.
func NewMemCache[K comparable, V any](opts MemOptions[K, V]) *MemCache[K, V] {
	cap := opts.Capacity
	if cap < 0 {
		cap = 0
	}
	return &MemCache[K, V]{
		opts: opts,
		ll:   list.New(),
		idx:  make(map[K]*list.Element, max(cap, 16)),
	}
}

// Compile-time assertion that *MemCache satisfies Cache.
var _ Cache[string, string] = (*MemCache[string, string])(nil)

func (c *MemCache[K, V]) expired(e *memEntry[K, V]) bool {
	return c.opts.TTL > 0 && time.Since(e.added) > c.opts.TTL
}

func (c *MemCache[K, V]) Get(_ context.Context, key K) (V, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var zero V
	el, ok := c.idx[key]
	if !ok {
		return zero, false, nil
	}
	ent := el.Value.(*memEntry[K, V])
	if c.expired(ent) {
		c.removeElement(el)
		return zero, false, nil
	}
	c.ll.MoveToFront(el)
	return ent.value, true, nil
}

func (c *MemCache[K, V]) Peek(_ context.Context, key K) (V, bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var zero V
	el, ok := c.idx[key]
	if !ok {
		return zero, false, nil
	}
	ent := el.Value.(*memEntry[K, V])
	if c.expired(ent) {
		return zero, false, nil
	}
	return ent.value, true, nil
}

func (c *MemCache[K, V]) Set(_ context.Context, key K, value V) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if el, ok := c.idx[key]; ok {
		ent := el.Value.(*memEntry[K, V])
		ent.value = value
		ent.added = time.Now()
		c.ll.MoveToFront(el)
		return nil
	}
	el := c.ll.PushFront(&memEntry[K, V]{key: key, value: value, added: time.Now()})
	c.idx[key] = el
	if c.opts.Capacity > 0 && c.ll.Len() > c.opts.Capacity {
		c.removeElement(c.ll.Back())
	}
	return nil
}

func (c *MemCache[K, V]) Delete(_ context.Context, keys ...K) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, k := range keys {
		if el, ok := c.idx[k]; ok {
			c.removeElement(el)
		}
	}
	return nil
}

func (c *MemCache[K, V]) Len(_ context.Context) (int, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ll.Len(), nil
}

func (c *MemCache[K, V]) Range(_ context.Context, fn func(K, V) bool) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for el := c.ll.Front(); el != nil; el = el.Next() {
		ent := el.Value.(*memEntry[K, V])
		if c.expired(ent) {
			continue
		}
		if !fn(ent.key, ent.value) {
			return nil
		}
	}
	return nil
}

func (c *MemCache[K, V]) Clear(_ context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.opts.OnEvict != nil {
		for _, el := range c.idx {
			ent := el.Value.(*memEntry[K, V])
			c.opts.OnEvict(ent.key, ent.value)
		}
	}
	c.ll = list.New()
	c.idx = make(map[K]*list.Element)
	return nil
}

func (c *MemCache[K, V]) PurgeExpired(_ context.Context) error {
	if c.opts.TTL <= 0 {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for el := c.ll.Front(); el != nil; {
		next := el.Next()
		if c.expired(el.Value.(*memEntry[K, V])) {
			c.removeElement(el)
		}
		el = next
	}
	return nil
}

// Close is a no-op for MemCache.
func (c *MemCache[K, V]) Close() error { return nil }

func (c *MemCache[K, V]) removeElement(el *list.Element) {
	ent := el.Value.(*memEntry[K, V])
	c.ll.Remove(el)
	delete(c.idx, ent.key)
	if c.opts.OnEvict != nil {
		c.opts.OnEvict(ent.key, ent.value)
	}
}
