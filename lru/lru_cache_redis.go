package lru

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// Marshaler turns a value of type V into bytes for Redis storage.
type Marshaler[V any] func(V) ([]byte, error)

// Unmarshaler reconstitutes a value of type V from its Redis-stored bytes.
type Unmarshaler[V any] func([]byte) (V, error)

// KeyEncoder turns a typed key into the Redis field/member string.
type KeyEncoder[K comparable] func(K) (string, error)

// KeyDecoder reconstitutes a typed key from the Redis field/member string.
type KeyDecoder[K comparable] func(string) (K, error)

// RedisOptions configures NewRedisCache.
type RedisOptions[K comparable, V any] struct {
	// Client is a pre-built *redis.Client. If nil, RedisOptions.Addr et al. are used.
	Client *redis.Client

	// Name is used as the prefix for the underlying Redis keys
	// ("<name>:lru" and "<name>:map"). Required.
	Name string

	// Addr is the "host:port" of the Redis server. Used only when Client is nil.
	Addr     string
	Password string
	DB       int
	PoolSize int

	// Capacity is the maximum number of entries kept in the cache. 0 means unbounded.
	Capacity int

	// TTL is the maximum age of an entry. 0 means entries never expire.
	TTL time.Duration

	EncodeKey   KeyEncoder[K]
	DecodeKey   KeyDecoder[K]
	EncodeValue Marshaler[V]
	DecodeValue Unmarshaler[V]
}

// RedisCache is a Cache implementation backed by a Redis sorted-set (LRU order)
// and a hash (key → value).
type RedisCache[K comparable, V any] struct {
	opts   RedisOptions[K, V]
	client *redis.Client
	owned  bool // true if we created the client and must Close() it
}

// NewRedisCache returns a Cache backed by Redis.
func NewRedisCache[K comparable, V any](opts RedisOptions[K, V]) (*RedisCache[K, V], error) {
	if opts.Name == "" {
		return nil, errors.New("lru: RedisOptions.Name is required")
	}
	if opts.EncodeKey == nil || opts.DecodeKey == nil {
		return nil, errors.New("lru: RedisOptions key codec is required")
	}
	if opts.EncodeValue == nil || opts.DecodeValue == nil {
		return nil, errors.New("lru: RedisOptions value codec is required")
	}

	c := &RedisCache[K, V]{opts: opts}
	if opts.Client != nil {
		c.client = opts.Client
	} else {
		if opts.PoolSize == 0 {
			opts.PoolSize = 10
		}
		c.client = redis.NewClient(&redis.Options{
			Addr:     opts.Addr,
			Password: opts.Password,
			DB:       opts.DB,
			PoolSize: opts.PoolSize,
		})
		c.owned = true
	}
	return c, nil
}

// Compile-time assertion that *RedisCache satisfies Cache.
var _ Cache[string, string] = (*RedisCache[string, string])(nil)

func (c *RedisCache[K, V]) lruKey() string { return c.opts.Name + ":lru" }
func (c *RedisCache[K, V]) mapKey() string { return c.opts.Name + ":map" }

func (c *RedisCache[K, V]) Get(ctx context.Context, key K) (V, bool, error) {
	var zero V
	field, err := c.opts.EncodeKey(key)
	if err != nil {
		return zero, false, fmt.Errorf("lru: encode key: %w", err)
	}

	script := redis.NewScript(`
		local mapKey, lruKey = KEYS[1], KEYS[2]
		local field, ts = ARGV[1], ARGV[2]
		local v = redis.call('HGET', mapKey, field)
		if v == false then return nil end
		redis.call('ZADD', lruKey, ts, field)
		return v
	`)
	res, err := script.Run(ctx, c.client,
		[]string{c.mapKey(), c.lruKey()},
		field, strconv.FormatInt(time.Now().Unix(), 10),
	).Result()
	if errors.Is(err, redis.Nil) {
		return zero, false, nil
	}
	if err != nil {
		return zero, false, fmt.Errorf("lru: redis eval get: %w", err)
	}
	if res == nil {
		return zero, false, nil
	}
	bs, _ := res.(string)
	value, err := c.opts.DecodeValue([]byte(bs))
	if err != nil {
		return zero, false, fmt.Errorf("lru: decode value: %w", err)
	}
	return value, true, nil
}

func (c *RedisCache[K, V]) Peek(ctx context.Context, key K) (V, bool, error) {
	var zero V
	field, err := c.opts.EncodeKey(key)
	if err != nil {
		return zero, false, fmt.Errorf("lru: encode key: %w", err)
	}
	b, err := c.client.HGet(ctx, c.mapKey(), field).Bytes()
	if errors.Is(err, redis.Nil) {
		return zero, false, nil
	}
	if err != nil {
		return zero, false, fmt.Errorf("lru: redis hget: %w", err)
	}
	value, err := c.opts.DecodeValue(b)
	if err != nil {
		return zero, false, fmt.Errorf("lru: decode value: %w", err)
	}
	return value, true, nil
}

func (c *RedisCache[K, V]) Set(ctx context.Context, key K, value V) error {
	field, err := c.opts.EncodeKey(key)
	if err != nil {
		return fmt.Errorf("lru: encode key: %w", err)
	}
	b, err := c.opts.EncodeValue(value)
	if err != nil {
		return fmt.Errorf("lru: encode value: %w", err)
	}

	script := redis.NewScript(`
		local mapKey, lruKey = KEYS[1], KEYS[2]
		local field, value, ts, cap = ARGV[1], ARGV[2], ARGV[3], tonumber(ARGV[4])
		redis.call('HSET', mapKey, field, value)
		redis.call('ZADD', lruKey, ts, field)
		if cap == 0 then return 0 end
		local size = redis.call('ZCARD', lruKey)
		if size <= cap then return 0 end
		local delTo = size - cap - 1
		local del = redis.call('ZRANGE', lruKey, 0, delTo)
		redis.call('ZREMRANGEBYRANK', lruKey, 0, delTo)
		for i = 1, #del do redis.call('HDEL', mapKey, del[i]) end
		return 0
	`)
	if _, err := script.Run(ctx, c.client,
		[]string{c.mapKey(), c.lruKey()},
		field, string(b),
		strconv.FormatInt(time.Now().Unix(), 10),
		strconv.Itoa(c.opts.Capacity),
	).Result(); err != nil {
		return fmt.Errorf("lru: redis eval set: %w", err)
	}
	return nil
}

func (c *RedisCache[K, V]) Delete(ctx context.Context, keys ...K) error {
	if len(keys) == 0 {
		return nil
	}
	fields := make([]string, len(keys))
	members := make([]any, len(keys))
	for i, k := range keys {
		s, err := c.opts.EncodeKey(k)
		if err != nil {
			return fmt.Errorf("lru: encode key: %w", err)
		}
		fields[i] = s
		members[i] = s
	}
	if _, err := c.client.HDel(ctx, c.mapKey(), fields...).Result(); err != nil {
		return fmt.Errorf("lru: redis hdel: %w", err)
	}
	if _, err := c.client.ZRem(ctx, c.lruKey(), members...).Result(); err != nil {
		return fmt.Errorf("lru: redis zrem: %w", err)
	}
	return nil
}

func (c *RedisCache[K, V]) Len(ctx context.Context) (int, error) {
	n, err := c.client.HLen(ctx, c.mapKey()).Result()
	if err != nil {
		return 0, fmt.Errorf("lru: redis hlen: %w", err)
	}
	return int(n), nil
}

func (c *RedisCache[K, V]) Range(ctx context.Context, fn func(K, V) bool) error {
	pairs, err := c.client.HGetAll(ctx, c.mapKey()).Result()
	if err != nil {
		return fmt.Errorf("lru: redis hgetall: %w", err)
	}
	for fieldStr, valStr := range pairs {
		key, err := c.opts.DecodeKey(fieldStr)
		if err != nil {
			return fmt.Errorf("lru: decode key: %w", err)
		}
		value, err := c.opts.DecodeValue([]byte(valStr))
		if err != nil {
			return fmt.Errorf("lru: decode value: %w", err)
		}
		if !fn(key, value) {
			return nil
		}
	}
	return nil
}

func (c *RedisCache[K, V]) Clear(ctx context.Context) error {
	if _, err := c.client.Del(ctx, c.mapKey(), c.lruKey()).Result(); err != nil {
		return fmt.Errorf("lru: redis del: %w", err)
	}
	return nil
}

func (c *RedisCache[K, V]) PurgeExpired(ctx context.Context) error {
	if c.opts.TTL <= 0 {
		return nil
	}
	cutoff := strconv.FormatInt(time.Now().Add(-c.opts.TTL).Unix(), 10)
	script := redis.NewScript(`
		local mapKey, lruKey, cutoff = KEYS[1], KEYS[2], ARGV[1]
		local del = redis.call('ZRANGEBYSCORE', lruKey, 0, cutoff)
		for i = 1, #del do redis.call('HDEL', mapKey, del[i]) end
		redis.call('ZREMRANGEBYSCORE', lruKey, 0, cutoff)
		return #del
	`)
	if _, err := script.Run(ctx, c.client,
		[]string{c.mapKey(), c.lruKey()},
		cutoff,
	).Result(); err != nil {
		return fmt.Errorf("lru: redis eval purgeExpired: %w", err)
	}
	return nil
}

// Close releases the Redis client (only when we created it).
func (c *RedisCache[K, V]) Close() error {
	if !c.owned {
		return nil
	}
	if err := c.client.Close(); err != nil {
		return fmt.Errorf("lru: redis close: %w", err)
	}
	return nil
}

// StringKeyCodec is a convenience encoder/decoder pair for string keys.
func StringKeyCodec() (KeyEncoder[string], KeyDecoder[string]) {
	return func(s string) (string, error) { return s, nil },
		func(s string) (string, error) { return s, nil }
}
