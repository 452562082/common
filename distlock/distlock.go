// Package distlock implements a single-instance Redis distributed lock.
//
// The lock uses SET NX PX <ttl> for acquisition and a Lua compare-and-delete
// for safe release. An optional auto-renew goroutine extends the TTL while
// the lock is held, so callers don't lose the lock during a long operation
// that out-runs the initial TTL.
//
// Caveats vs. Redlock:
//
//   - This is the simple, single-master pattern. It is correct against a
//     single Redis primary; in failover scenarios a brief split-brain window
//     is possible. For stricter guarantees implement the multi-instance
//     Redlock algorithm (or use redsync), at the cost of operational
//     complexity.
//   - The unlock script is the only safe way to release: it checks the
//     owner token before deleting, so a process holding a stale lock can't
//     accidentally drop someone else's.
package distlock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrNotAcquired is returned when TryLock cannot get the lock.
var ErrNotAcquired = errors.New("distlock: lock not acquired")

// ErrNotHeld is returned by Unlock / Refresh when the lock is no longer ours
// (released, expired, or stolen).
var ErrNotHeld = errors.New("distlock: lock not held by this owner")

// Options configures Locker.
type Options struct {
	// TTL is the lock's lease duration. Default 30s.
	TTL time.Duration

	// AutoRefresh, when > 0, runs a background goroutine that extends the
	// lock's TTL every AutoRefresh. Pick something like TTL/3.
	AutoRefresh time.Duration

	// AcquireTimeout caps how long Lock blocks waiting. Zero means use ctx only.
	AcquireTimeout time.Duration

	// PollInterval is the delay between Lock retries while waiting.
	// Default 100ms.
	PollInterval time.Duration

	// KeyPrefix is prepended to every lock key, useful for namespacing.
	// Default "lock:".
	KeyPrefix string
}

// Locker creates locks against a redis client.
type Locker struct {
	client redis.UniversalClient
	opts   Options
}

// New returns a Locker. The redis client may be a *redis.Client,
// *redis.ClusterClient, or any UniversalClient.
func New(client redis.UniversalClient, opts Options) *Locker {
	if opts.TTL == 0 {
		opts.TTL = 30 * time.Second
	}
	if opts.PollInterval == 0 {
		opts.PollInterval = 100 * time.Millisecond
	}
	if opts.KeyPrefix == "" {
		opts.KeyPrefix = "lock:"
	}
	return &Locker{client: client, opts: opts}
}

// Lock holds one acquired lock. Unlock should always be called (typically via defer).
type Lock struct {
	locker *Locker
	key    string
	token  string

	stopRenew chan struct{}
	renewDone chan struct{}
	once      sync.Once
}

// TryLock attempts a single acquisition. Returns ErrNotAcquired immediately
// if someone else holds the lock.
func (l *Locker) TryLock(ctx context.Context, key string) (*Lock, error) {
	token, err := randomToken()
	if err != nil {
		return nil, fmt.Errorf("distlock: gen token: %w", err)
	}
	fullKey := l.opts.KeyPrefix + key

	ok, err := l.client.SetNX(ctx, fullKey, token, l.opts.TTL).Result()
	if err != nil {
		return nil, fmt.Errorf("distlock: setnx: %w", err)
	}
	if !ok {
		return nil, ErrNotAcquired
	}
	return l.newLock(fullKey, token), nil
}

// Lock blocks until the lock is acquired or ctx (or AcquireTimeout) elapses.
func (l *Locker) Lock(ctx context.Context, key string) (*Lock, error) {
	if l.opts.AcquireTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, l.opts.AcquireTimeout)
		defer cancel()
	}

	t := time.NewTicker(l.opts.PollInterval)
	defer t.Stop()
	for {
		lock, err := l.TryLock(ctx, key)
		if err == nil {
			return lock, nil
		}
		if !errors.Is(err, ErrNotAcquired) {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("distlock: %w", ctx.Err())
		case <-t.C:
		}
	}
}

func (l *Locker) newLock(fullKey, token string) *Lock {
	lock := &Lock{locker: l, key: fullKey, token: token}
	if l.opts.AutoRefresh > 0 {
		lock.stopRenew = make(chan struct{})
		lock.renewDone = make(chan struct{})
		go lock.renewLoop()
	}
	return lock
}

// unlockScript safely deletes the key only if its value matches token.
var unlockScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
else
	return 0
end
`)

// refreshScript safely extends the TTL only if the value still matches.
var refreshScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("PEXPIRE", KEYS[1], ARGV[2])
else
	return 0
end
`)

// Unlock releases the lock. It is safe to call multiple times.
// Returns ErrNotHeld if the lock had already expired or been stolen.
func (lk *Lock) Unlock(ctx context.Context) error {
	var firstErr error
	lk.once.Do(func() {
		if lk.stopRenew != nil {
			close(lk.stopRenew)
			<-lk.renewDone
		}
		n, err := unlockScript.Run(ctx, lk.locker.client, []string{lk.key}, lk.token).Int()
		if err != nil {
			firstErr = fmt.Errorf("distlock: unlock: %w", err)
			return
		}
		if n == 0 {
			firstErr = ErrNotHeld
		}
	})
	return firstErr
}

// Refresh extends the TTL by the locker's configured TTL.
// Returns ErrNotHeld if the lock is no longer ours.
func (lk *Lock) Refresh(ctx context.Context) error {
	n, err := refreshScript.Run(ctx, lk.locker.client,
		[]string{lk.key},
		lk.token,
		int64(lk.locker.opts.TTL/time.Millisecond),
	).Int()
	if err != nil {
		return fmt.Errorf("distlock: refresh: %w", err)
	}
	if n == 0 {
		return ErrNotHeld
	}
	return nil
}

// Key returns the full Redis key the lock occupies (including prefix).
func (lk *Lock) Key() string { return lk.key }

// Token returns the random owner token (mostly useful for tests/debug).
func (lk *Lock) Token() string { return lk.token }

func (lk *Lock) renewLoop() {
	defer close(lk.renewDone)
	t := time.NewTicker(lk.locker.opts.AutoRefresh)
	defer t.Stop()
	for {
		select {
		case <-lk.stopRenew:
			return
		case <-t.C:
			ctx, cancel := context.WithTimeout(context.Background(), lk.locker.opts.TTL/2)
			if err := lk.Refresh(ctx); err != nil {
				cancel()
				return // either gone or fatal — stop renewing
			}
			cancel()
		}
	}
}

func randomToken() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
