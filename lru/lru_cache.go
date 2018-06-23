package lru

import (
	"container/list"
	"encoding/binary"
	"fmt"
	"github.com/coocood/freecache"
	"kuaishangtong/common/utils/log"
	"sync"
	"time"
	"unsafe"
)

type Cache interface {
	Length() int64
	Add(key Key, value interface{}) error
	Modify(key Key, value interface{}) (bool, error)
	Get(key Key) (interface{}, bool, error)
	Peek(key Key) (interface{}, bool, error)
	Purge(getAliveFunc, getExpiredFunc func(Key, interface{})) error
	GetAll(getAliveFunc func(Key, interface{})) error
	Remove(key ...Key) error
	RemoveAll() error
	RemoveOldest() error
	SetTimeout(timeout int64)
	UpdateTimestamp(Key, int64) error
	SetRemoveCallback(removeCallback func(Key, interface{}))
	SetRemoveOldestCallback(removeOldestCallback func(Key, interface{}))
	ClearExpireKeys() error

	// multi operations
	//MultiAdd([]Key, []interface{}) error
	//MultiModify([]Key, []interface{}) error
	//MultiGet(keys []Key) ([]interface{}, error)
}

type CacheMem struct {
	Name string
	sync.RWMutex
	Limits  int // 0 means unlimits
	Timeout int64
	cache   *freecache.Cache

	lru *list.List

	pool *sync.Pool

	removeCallback       func(Key, interface{})
	removeOldestCallback func(Key, interface{})
}

// A Key may be any value that is comparable.
// See http://golang.org/ref/spec#Comparison_operators
type Key interface{}

type entry struct {
	key       Key
	value     interface{}
	timestamp int64
}

func (e *entry) reset(key Key, value interface{}, timestamp int64) {
	e.key = key
	e.value = value
	e.timestamp = timestamp
}

// New creates a new CacheMem.
// If maxEntries is zero, the cache has no limit and it's assumed
// that eviction is done by the caller.
func NewCacheMem(name string, limits int, timeout int) Cache {
	return &CacheMem{
		Name:    name,
		Limits:  limits,
		Timeout: int64(timeout),
		lru:     list.New(),
		cache:   freecache.NewCache(limits * 100),

		pool: &sync.Pool{
			New: func() interface{} {
				return new(entry)
			},
		},
	}
}

func (c *CacheMem) alloc(k Key, value interface{}, timestamp int64) (e *entry) {
	e = c.pool.Get().(*entry)
	e.reset(k, value, timestamp)
	return e
}

func (c *CacheMem) recycle(entry *entry) {
	c.pool.Put(entry)
}

func (c *CacheMem) Length() int64 {
	c.Lock()
	defer c.Unlock()
	if c.cache == nil {
		return 0
	}
	return int64(c.lru.Len())
}
func (c *CacheMem) SetTimeout(timeout int64) {
	c.Timeout = timeout
}

func (c *CacheMem) cacheSearch(key Key) (e *list.Element, ok bool) {
	keybytes := []byte(key.(string))

	valuebytes, _ := c.cache.Get(keybytes)
	if valuebytes != nil {
		e = (*list.Element)(unsafe.Pointer(uintptr(binary.LittleEndian.Uint64(valuebytes))))
		ok = true
		return
	}

	return
}

func (c *CacheMem) cacheSet(key Key, e *list.Element) {
	keybytes := []byte(key.(string))

	var valueBuf []byte = make([]byte, 8)
	binary.LittleEndian.PutUint64(valueBuf, uint64(uintptr(unsafe.Pointer(e))))
	if err := c.cache.Set(keybytes, valueBuf, 0); err != nil {
		log.Warnf("CacheMem.cacheSet: %s", err.Error())
	}

}

func (c *CacheMem) cacheDel(key Key) {
	keybytes := []byte(key.(string))

	c.cache.Del(keybytes)
}

func (c *CacheMem) Add(key Key, value interface{}) (err error) {
	c.Lock()
	defer c.Unlock()

	var e *list.Element

	if e, ok := c.cacheSearch(key); ok {
		c.lru.MoveToFront(e)
		e.Value.(*entry).value = value
		e.Value.(*entry).timestamp = time.Now().Unix()
		return
	}

	e = c.lru.PushFront(c.alloc(key, value, time.Now().Unix()))
	c.cacheSet(key, e)
	if c.Limits != 0 && c.lru.Len() > c.Limits {
		c.removeOldest()
	}
	return
}

// modify value, without changing the LRU queue
func (c *CacheMem) Modify(key Key, value interface{}) (exist bool, err error) {
	c.Lock()
	defer c.Unlock()

	var e *list.Element

	e, exist = c.cacheSearch(key)
	if exist {
		e.Value.(*entry).value = value
	}

	return exist, nil
}

// looks up a key's value from the cache, move item to the front of LRU queue
func (c *CacheMem) Get(key Key) (interface{}, bool, error) {
	c.Lock()
	defer c.Unlock()
	if e, ok := c.cacheSearch(key); ok {
		c.lru.MoveToFront(e)
		e.Value.(*entry).timestamp = time.Now().Unix()
		value := e.Value.(*entry).value
		// log.Debug("lru Get:", c.Name, key, value)
		return value, true, nil
	}
	return nil, false, nil
}

// looks up a key's value from the cache, without changing the LRU queue
func (c *CacheMem) Peek(key Key) (interface{}, bool, error) {
	c.RLock()
	defer c.RUnlock()

	if e, hit := c.cacheSearch(key); hit {
		value := e.Value.(*entry).value
		// log.Debug("lru Peek:", c.Name, key, value)
		return value, true, nil
	}
	return nil, false, nil
}

func (c *CacheMem) CheckTimeStampRemain(key Key) (timeRemain int64, ok bool, err error) {
	c.RLock()
	defer c.RUnlock()

	if e, hit := c.cacheSearch(key); hit {
		now := time.Now().Unix()
		timestamp := e.Value.(*entry).timestamp
		if c.Timeout != 0 {
			if now-timestamp >= c.Timeout {
				return -1, false, nil
			} else {
				return c.Timeout - (now - timestamp), true, nil
			}
		} else {
			return 0, true, nil
		}

	}
	return -1, false, nil
}

func (c *CacheMem) CheckTimeStamp(key Key) (bool, error) {
	c.RLock()
	defer c.RUnlock()

	if e, hit := c.cacheSearch(key); hit {
		now := time.Now().Unix()
		timestamp := e.Value.(*entry).timestamp
		if c.Timeout != 0 && now-timestamp >= c.Timeout {
			return false, nil
		}
		return true, nil
	}
	return false, nil
}

func (c *CacheMem) UpdateTimestamp(key Key, timestamp int64) error {
	c.Lock()
	defer c.Unlock()

	if e, hit := c.cacheSearch(key); hit {
		c.lru.MoveToFront(e)
		e.Value.(*entry).timestamp = timestamp
		//return true, nil
	}
	//return false, nil
	return nil
}

func (c *CacheMem) GetAll(getAliveFunc func(Key, interface{})) error {
	c.RLock()
	defer c.RUnlock()

	for e := c.lru.Front(); e != nil; {
		key := e.Value.(*entry).key
		value := e.Value.(*entry).value
		e = e.Next()
		if getAliveFunc != nil {
			getAliveFunc(key, value)
		}
	}
	return nil
}

func (c *CacheMem) Purge(getAliveFunc, getExpiredFunc func(Key, interface{})) (err error) {
	c.Lock()
	defer c.Unlock()

	for e := c.lru.Front(); e != nil; {
		key := e.Value.(*entry).key
		value := e.Value.(*entry).value
		timestamp := e.Value.(*entry).timestamp
		if c.Timeout != 0 && (time.Now().Unix()-timestamp) > c.Timeout {
			if ele, hit := c.cacheSearch(key); hit {
				e = c.removeElement(ele)
			}

			if getExpiredFunc != nil {
				getExpiredFunc(key, value)
			}
		} else {
			e = e.Next()
			if getAliveFunc != nil {
				getAliveFunc(key, value)
			}
		}
	}
	return
}

// Remove removes the provided key from the cache.
func (c *CacheMem) Remove(keys ...Key) (err error) {
	c.Lock()
	defer c.Unlock()

	for _, key := range keys {
		if e, hit := c.cacheSearch(key); hit {
			c.removeElement(e)
		}
	}
	return
}

func (c *CacheMem) RemoveAll() error {
	c.Lock()
	defer c.Unlock()

	for e := c.lru.Front(); e != nil; {
		e = c.removeElement(e)
	}

	return nil
}

func (c *CacheMem) SetRemoveCallback(removeCallback func(Key, interface{})) {
	c.removeCallback = removeCallback
}

func (c *CacheMem) SetRemoveOldestCallback(removeOldestCallback func(Key, interface{})) {
	c.removeOldestCallback = removeOldestCallback
}

// RemoveOldest removes the oldest item from the cache.
func (c *CacheMem) RemoveOldest() (err error) {
	c.Lock()
	defer c.Unlock()

	c.removeOldest()
	return
}
func (c *CacheMem) removeOldest() {
	if c.cache == nil {
		return
	}
	e := c.lru.Back()
	if e != nil {
		if c.removeOldestCallback != nil {
			c.removeOldestCallback(e.Value.(*entry).key, e.Value.(*entry).value)
		}
		c.removeElement(e)
	}
}

func (c *CacheMem) removeElement(e *list.Element) *list.Element {
	if c.removeCallback != nil {
		c.removeCallback(e.Value.(*entry).key, e.Value.(*entry).value)
	}
	eNext := e.Next()
	c.lru.Remove(e)
	kv := e.Value.(*entry)
	c.cacheDel(kv.key)
	c.recycle(kv)
	return eNext
}

func (c *CacheMem) ClearExpireKeys() error {
	c.Lock()
	defer c.Unlock()

	if c.Timeout == 0 {
		return nil
	}
	for e := c.lru.Front(); e != nil; {
		key := e.Value.(*entry).key
		timestamp := e.Value.(*entry).timestamp
		if (time.Now().Unix() - timestamp) > c.Timeout {
			if ele, hit := c.cacheSearch(key); hit {
				e = c.removeElement(ele)
			}
		} else {
			e = e.Next()
		}
	}
	return nil
}

func (c *CacheMem) MultiAdd(keys []Key, values []interface{}) (err error) {
	c.Lock()
	defer c.Unlock()

	if len(keys) != len(values) {
		return fmt.Errorf("MultiAdd: keys' length not equal to values' length")
	}

	if len(keys) == 0 {
		return nil
	}

	var e *list.Element
	var value interface{}
	for i, key := range keys {
		value = values[i]
		if e, ok := c.cacheSearch(key); ok {
			c.lru.MoveToFront(e)
			e.Value.(*entry).value = value
			e.Value.(*entry).timestamp = time.Now().Unix()
			continue
		}

		e = c.lru.PushFront(&entry{key, value, time.Now().Unix()})
		c.cacheSet(key, e)
		if c.Limits != 0 && c.lru.Len() > c.Limits {
			c.removeOldest()
		}

	}
	return
}

func (c *CacheMem) MultiModify(keys []Key, values []interface{}) (err error) {
	c.Lock()
	defer c.Unlock()

	if len(keys) != len(values) {
		return fmt.Errorf("MultiModify: keys' length not equal to values' length")
	}

	if len(keys) == 0 {
		return nil
	}

	var value interface{}
	for i, key := range keys {
		value = values[i]
		if e, ok := c.cacheSearch(key); ok {
			e.Value.(*entry).value = value
		}
	}
	return
}

func (c *CacheMem) MultiGet(keys []Key) ([]interface{}, error) {
	c.Lock()
	defer c.Unlock()

	values := make([]interface{}, len(keys))
	for i, key := range keys {
		if e, ok := c.cacheSearch(key); ok {
			c.lru.MoveToFront(e)
			e.Value.(*entry).timestamp = time.Now().Unix()
			values[i] = e.Value.(*entry).value
		} else {
			values[i] = nil
		}
	}

	return values, nil
}
