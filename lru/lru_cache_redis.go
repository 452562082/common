package lru

import (
	"bytes"
	"fmt"
	"strconv"
	"sync"
	"time"

	"goal/log"
	"goal/redis"
)

type CacheRedis struct {
	sync.RWMutex
	Name             string
	Limits           int // 0 means unlimits
	Timeout          int64
	client           *redis.Client
	ValueMarshaler   func(interface{}) ([]byte, error)
	ValueUnmarshaler func([]byte) (interface{}, error)
	KeyMarshaler     func(interface{}) (string, error)
	KeyUnmarshaler   func(string) (interface{}, error)

	removeCallback       func(Key, interface{})
	removeOldestCallback func(Key, interface{})

	memcacheEnable bool
	memcache       map[Key]interface{}

	cmdbuf  *bytes.Buffer
	cmdbuf2 *bytes.Buffer
}

func NewCacheRedis(limits int, timeout int, name string, redis_ip string, redis_port int, redis_pass string, redis_db int,
	valueMarshaler func(interface{}) ([]byte, error),
	valueUnmarshaler func([]byte) (interface{}, error),
	keyMarshaler func(interface{}) (string, error),
	keyUnmarshaler func(string) (interface{}, error),
	memcacheEnable bool) Cache {
	c := &CacheRedis{
		Limits:           limits,
		Name:             name,
		Timeout:          int64(timeout),
		client:           &redis.Client{Addr: fmt.Sprintf("%s:%d", redis_ip, redis_port), Password: redis_pass, Db: redis_db},
		ValueMarshaler:   valueMarshaler,
		ValueUnmarshaler: valueUnmarshaler,
		KeyMarshaler:     keyMarshaler,
		KeyUnmarshaler:   keyUnmarshaler,
		memcacheEnable:   memcacheEnable,
	}

	c.memcacheReInit()
	c.cmdbuf = new(bytes.Buffer)
	c.cmdbuf2 = new(bytes.Buffer)

	return c
}

func (c *CacheRedis) Length() int {
	c.RLock()
	defer c.RUnlock()

	length, err := c.client.Hlen(c.mapName())
	if err != nil {
		log.Warnf("CacheRedis.Length: redis.Hlen: %s", err.Error())
		return -1
	}
	return length
}
func (c *CacheRedis) SetTimeout(timeout int64) {
	c.Timeout = timeout
}

func (c *CacheRedis) lruName() string {
	return c.Name + ":lru"
}

func (c *CacheRedis) mapName() string {
	return c.Name + ":map"
}

func (c *CacheRedis) Add(key Key, value interface{}) error {
	c.Lock()
	defer c.Unlock()
	if log.MinSeverity <= log.DEBUG {
		log.Debug("lru_cache_redis Add:", c.Name, key, value)
	}

	keyString, err := c.KeyMarshaler(key)
	if err != nil {
		return err
	}
	b, err := c.ValueMarshaler(value)
	if err != nil {
		return err
	}

	valueString := string(b)
	currentTime := strconv.FormatInt(time.Now().Unix(), 10)
	limits := strconv.FormatInt(int64(c.Limits), 10)
	script := `local keyString = KEYS[3];
		local valueString = KEYS[4];
		local currentTime = KEYS[5];
		local limits = tonumber(KEYS[6]);
		redis.call('HSET',KEYS[1],keyString,valueString);
		redis.call('ZADD',KEYS[2],currentTime,keyString);
		local size=redis.call('ZCARD',KEYS[2]);
		if limits == 0 or size<=limits then return 0; end;
		local delIndex=size-limits-1;
		local delkeys=redis.call('ZRANGE',KEYS[2],0,delIndex);
		redis.call('ZREMRANGEBYRANK',KEYS[2],0,delIndex);
		for i=1,table.getn(delkeys) do redis.call('HDEL',KEYS[1],delkeys[i]) end;
		return 0`
	_, err = c.client.Eval(script, 6, c.mapName(), c.lruName(), keyString, valueString, currentTime, limits)
	if err != nil {
		return err
	}

	c.memcacheSet(key, value)

	return nil
}

// modify value, without changing the LRU queue
func (c *CacheRedis) Modify(key Key, value interface{}) (bool, error) {
	c.Lock()
	defer c.Unlock()
	if log.MinSeverity <= log.DEBUG {
		log.Debug("lru_cache_redis Modify:", c.Name, key, value)
	}

	keyString, err := c.KeyMarshaler(key)
	if err != nil {
		return false, err
	}
	b, err := c.ValueMarshaler(value)
	if err != nil {
		return false, err
	}

	script := `local target = KEYS[2];
		local value = KEYS[3];
		local ex=redis.call('HEXISTS',KEYS[1],target);
		if ex == 0 then return 0 end;
		redis.call('HSET',KEYS[1],target,value);
		return 1`
	ret, err := c.client.Eval(script, 3, c.mapName(), keyString, string(b))
	if err != nil {
		return false, err
	}
	if ret != nil {
		if i, ok := ret.(int64); ok && i == 0 {
			if log.MinSeverity <= log.DEBUG {
				log.Debug("lru_cache_redis Modify:", c.Name, key, "was not exists.")
			}
			return false, nil
		}
	}

	c.memcacheSet(key, value)

	return true, nil
}

// looks up a key's value from the cache, move item to the front of LRU queue
func (c *CacheRedis) Get(key Key) (interface{}, bool, error) {
	c.RLock()
	val, ok := c.memcacheGet(key)
	c.RUnlock()
	if ok {
		return val, true, nil
	}

	c.Lock()
	defer c.Unlock()

	keyString, err := c.KeyMarshaler(key)
	if err != nil {
		return nil, false, err
	}

	currentTime := strconv.FormatInt(time.Now().Unix(), 10)
	script := `local keyString = KEYS[3];
		local currentTime = KEYS[4];
		local value=redis.call('HGET',KEYS[1],keyString);
		redis.call('ZADD',KEYS[2],currentTime,keyString);
		return value`
	ret, err := c.client.Eval(script, 4, c.mapName(), c.lruName(), keyString, currentTime)
	if ret == nil {
		c.memcacheSet(key, nil)
		return nil, false, nil
	}
	b := ret.([]byte)
	value, err := c.ValueUnmarshaler(b)
	if err != nil {
		return nil, false, err
	}
	// log.Debug("lru_cache_redis Get:", c.Name, key, value)

	c.memcacheSet(key, value)

	return value, true, nil
}

// looks up a key's value from the cache, without changing the LRU queue
func (c *CacheRedis) Peek(key Key) (interface{}, bool, error) {
	c.RLock()
	val, ok := c.memcacheGet(key)
	c.RUnlock()
	if ok {
		return val, true, nil
	}

	c.Lock()
	defer c.Unlock()

	keyString, err := c.KeyMarshaler(key)
	if err != nil {
		return nil, false, err
	}
	b, _ := c.client.Hget(c.mapName(), keyString)
	if b == nil {
		c.memcacheSet(key, nil)
		return nil, false, nil
	}

	value, err := c.ValueUnmarshaler(b)
	if err != nil {
		return nil, false, err
	}
	// log.Debug("lru_cache_redis Peek:", c.Name, key, value)

	c.memcacheSet(key, value)

	return value, true, nil
}

func (c *CacheRedis) UpdateTimestamp(key Key, timestamp int64) (bool, error) {
	c.Lock()
	defer c.Unlock()

	keyString, err := c.KeyMarshaler(key)
	if err != nil {
		return false, err
	}
	added, err := c.client.Zadd(c.lruName(), keyString, float64(time.Now().Unix()))
	if err != nil {
		return false, err
	}
	return !added, nil
}

func (c *CacheRedis) GetAll(getAliveFunc func(Key, interface{})) error {
	c.RLock()
	defer c.RUnlock()

	script := `return redis.call('HGETALL',KEYS[1]);`
	// log.Debug("lru_cache_redis Purge: send command start", time.Now())
	res, err := c.client.Eval(script, 1, c.mapName())
	// log.Debug("lru_cache_redis Purge: send command finish", time.Now())
	if err != nil {
		return err
	}
	if getAliveFunc != nil {
		bs := res.([][]byte)
		// log.Debug("lru_cache_redis Purge: for cycle start", time.Now())
		for i := 0; i < len(bs)/2; i++ {
			b := bs[2*i]
			vb := bs[2*i+1]
			keyString := string(b)

			var value interface{}
			if string(vb) == "" {
				// log.Warn("lru_cache_redis Purge:", c.Name, "key do not exist", keyString)
				value = nil
			} else {
				value, err = c.ValueUnmarshaler(vb)
				if err != nil {
					log.Warn("lru_cache_redis GetAll:", c.Name, "c.ValueUnmarshaler", err)
					continue
				}
			}

			key, err := c.KeyUnmarshaler(keyString)
			if err != nil {
				log.Warn("lru_cache_redis GetAll:", c.Name, "c.KeyUnmarshaler", err)
				continue
			}
			getAliveFunc(key, value)
		}
		// log.Debug("lru_cache_redis Purge: for cycle finish", time.Now())
	}
	return nil
}

//TODO(K'): getExpiredFunc() was not called in this function
func (c *CacheRedis) Purge(getAliveFunc, getExpiredFunc func(Key, interface{})) error {
	c.Lock()
	defer c.Unlock()

	c.memcacheReInit()

	cacheTimeout := strconv.FormatInt(c.Timeout, 10)
	timeoutTime := strconv.FormatInt(time.Now().Unix()-c.Timeout, 10)
	script := `local cacheTimeout = tonumber(KEYS[3]);
		local timeoutTime = KEYS[4];
		if cacheTimeout ~= 0 then 
		local delKeys=redis.call('ZRANGEBYSCORE',KEYS[2],0,timeoutTime);
		for i=1,table.getn(delKeys) do redis.call('HDEL',KEYS[1],delKeys[i]) end;
		redis.call('ZREMRANGEBYSCORE',KEYS[2],0,timeoutTime);
		end;
		local keys=redis.call('ZREVRANGE',KEYS[2],0,-1);
		local returnArray = {};
		for i=1,table.getn(keys)
		do
		returnArray[2*i-1]=keys[i];
		returnArray[2*i]=redis.call('HGET',KEYS[1],keys[i]);
		end;
		return returnArray`
	// log.Debug("lru_cache_redis Purge: send command start", time.Now())
	res, err := c.client.Eval(script, 4, c.mapName(), c.lruName(), cacheTimeout, timeoutTime)
	// log.Debug("lru_cache_redis Purge: send command finish", time.Now())
	if err != nil {
		return err
	}

	bs := res.([][]byte)
	// log.Debug("lru_cache_redis Purge: for cycle start", time.Now())
	for i := 0; i < len(bs)/2; i++ {
		b := bs[2*i]
		vb := bs[2*i+1]
		keyString := string(b)

		var value interface{}
		if string(vb) == "" {
			// log.Warn("lru_cache_redis Purge:", c.Name, "key do not exist", keyString)
			value = nil
		} else {
			value, err = c.ValueUnmarshaler(vb)
			if err != nil {
				log.Warn("lru_cache_redis Purge:", c.Name, "c.ValueUnmarshaler", err)
				continue
			}
		}

		key, err := c.KeyUnmarshaler(keyString)
		if err != nil {
			log.Warn("lru_cache_redis Purge:", c.Name, "c.KeyUnmarshaler", err)
			continue
		}

		c.memcacheSet(key, value)

		if getAliveFunc != nil {
			getAliveFunc(key, value)
		}
	}

	return nil
}

// Remove removes the provided key from the cache.
func (c *CacheRedis) Remove(keys ...Key) error {
	c.Lock()
	defer c.Unlock()
	// log.Debug("lru_cache_redis Remove:", c.Name, key)

	var err error
	keyStrings := make([]string, len(keys))
	for i, key := range keys {
		if keyString, err := c.KeyMarshaler(key); err != nil {
			return err
		} else {
			keyStrings[i] = keyString
		}
	}

	_, err = c.client.Hdel(c.mapName(), keyStrings...)
	if err != nil {
		return err
	}
	_, err = c.client.Zrem(c.lruName(), keyStrings...)
	if err != nil {
		return err
	}

	c.memcacheRemove(keys...)

	return nil
}

func (c *CacheRedis) RemoveAll() error {
	c.Lock()
	defer c.Unlock()

	_, err := c.client.Del(c.mapName(), c.lruName())
	log.Debug("lru_cache_redis RemoveAll:", c.Name)
	if err != nil {
		return err
	}
	return nil
}

func (c *CacheRedis) SetRemoveCallback(removeCallback func(Key, interface{})) {
	c.removeCallback = removeCallback
}

func (c *CacheRedis) SetRemoveOldestCallback(removeOldestCallback func(Key, interface{})) {
	c.removeOldestCallback = removeOldestCallback
}

// RemoveOldest removes the oldest item from the lru.
func (c *CacheRedis) RemoveOldest() error {
	c.Lock()
	defer c.Unlock()

	return c.removeOldest()
}

func (c *CacheRedis) removeOldest() error {
	// log.Debug("lru_cache_redis removeOldest:", c.Name)
	limits := strconv.FormatInt(int64(c.Limits), 10)
	script := `local limits=tonumber(KEY[3]);
		local size=redis.call('ZCARD',KEYS[2]);
		if size<=limits then return 0; end;
		local delIndex=size-limits-1;
		local delKeys=redis.call('ZRANGE',KEYS[2],0,delIndex);
		redis.call('ZREMRANGEBYRANK',KEYS[2],0,delIndex);
		for i=1,table.getn(delKeys) do redis.call('HDEL',KEYS[1],delKeys[i]) end;
		return 0`
	_, err := c.client.Eval(script, 3, c.mapName(), c.lruName(), limits)
	return err
}

func (c *CacheRedis) ClearExpireKeys() error {
	c.Lock()
	defer c.Unlock()

	cacheTimeout := strconv.FormatInt(c.Timeout, 10)
	timeoutTime := strconv.FormatInt(time.Now().Unix()-c.Timeout, 10)
	script := `local cacheTimeout = tonumber(KEYS[3]);
		local timeoutTime = KEYS[4];
		if cacheTimeout ~= 0 then 
			local delKeys=redis.call('ZRANGEBYSCORE',KEYS[2],0,timeoutTime);
			for i=1,table.getn(delKeys) do 
				redis.call('HDEL',KEYS[1],delKeys[i]) 
			end;
			redis.call('ZREMRANGEBYSCORE',KEYS[2],0,timeoutTime);
		end;
		return 0`
	_, err := c.client.Eval(script, 4, c.mapName(), c.lruName(), cacheTimeout, timeoutTime)
	if err != nil {
		return err
	}

	return nil
}

// func (c *CacheRedis) Keys() []Key {
// 	c.Lock()
// 	defer c.Unlock()

// 	keys := make([]Key, 0, c.ll.Len())
// 	for e := c.ll.Front(); e != nil; e = e.Next() {
// 		keys = append(keys, e.Value.(*entry).key)
// 	}
// 	return keys
// }

func (c *CacheRedis) MultiAdd(keys []Key, values []interface{}) error {
	c.Lock()
	defer c.Unlock()

	if len(keys) != len(values) {
		return fmt.Errorf("MultiAdd: keys' length not equal to values' length")
	}

	if len(keys) == 0 {
		return nil
	}

	score := strconv.FormatFloat(float64(time.Now().Unix()), 'f', -1, 64)
	argc := 1 + 2*len(keys)
	c.client.ResetCommandBuffer(c.cmdbuf, "HMSET", argc)
	c.client.AppendCommandBufferWithString(c.cmdbuf, c.mapName())

	c.client.ResetCommandBuffer(c.cmdbuf2, "ZADD", argc)
	c.client.AppendCommandBufferWithString(c.cmdbuf2, c.lruName())

	var value interface{}
	for i, key := range keys {
		value = values[i]
		keyString, err := c.KeyMarshaler(key)
		if err != nil {
			return err
		}
		b, err := c.ValueMarshaler(value) //xxxxx: 818MB
		if err != nil {
			return err
		}

		c.client.AppendCommandBufferWithString(c.cmdbuf, keyString)
		c.client.AppendCommandBufferWithBytes(c.cmdbuf, b)

		c.client.AppendCommandBufferWithString(c.cmdbuf2, score)
		c.client.AppendCommandBufferWithString(c.cmdbuf2, keyString)
	}

	if err := c.client.SendCommandBuffer(c.cmdbuf); err != nil {
		return err
	}
	if err := c.client.SendCommandBuffer(c.cmdbuf2); err != nil {
		return err
	}

	script := `local limits = tonumber(KEYS[3]);
		local size = redis.call('ZCARD',KEYS[2]);
		if limits > 0 and size > limits then 
			local delIndex=size-limits-1;
			local delkeys=redis.call('ZRANGE',KEYS[2], 0, delIndex);
			redis.call('ZREMRANGEBYRANK', KEYS[2], 0, delIndex);
			for i=1,table.getn(delkeys) do redis.call('HDEL',KEYS[1],delkeys[i]) end;
		end; 
		return 0;`
	if _, err := c.client.Eval(script, 3, c.mapName(), c.lruName(), strconv.FormatInt(int64(c.Limits), 10)); err != nil {
		return err
	}

	return nil
}

func (c *CacheRedis) MultiModify(keys []Key, values []interface{}) error {
	c.Lock()
	defer c.Unlock()

	if len(keys) != len(values) {
		return fmt.Errorf("MultiModify: keys' length not equal to values' length")
	}

	if len(keys) == 0 {
		return nil
	}

	args := make([]string, 1+2*len(keys))
	args[0] = c.mapName()
	var value interface{}
	for i, key := range keys {
		value = values[i]
		keyString, err := c.KeyMarshaler(key)
		if err != nil {
			return err
		}
		b, err := c.ValueMarshaler(value)
		if err != nil {
			return err
		}
		args[2*i+1] = keyString
		args[2*i+2] = string(b)
	}

	if err := c.client.SendCommand("HMSET", args...); err != nil {
		return err
	}

	return nil
}

func (c *CacheRedis) MultiGet(keys []Key) ([]interface{}, error) {
	c.Lock()
	defer c.Unlock()

	if len(keys) == 0 {
		return nil, nil
	}

	fields := make([]string, len(keys))
	for i, key := range keys {
		keyString, err := c.KeyMarshaler(key)
		if err != nil {
			return nil, err
		}
		fields[i] = keyString
	}

	if datas, err := c.client.Hmget(c.mapName(), fields...); err != nil {
		return nil, err
	} else {
		values := make([]interface{}, len(datas))
		args2 := make([]string, 1+2*len(keys))
		args2[0] = c.lruName()
		score := strconv.FormatFloat(float64(time.Now().Unix()), 'f', -1, 64)

		for i, data := range datas {
			if data == nil {
				values[i] = nil
				continue
			}
			if value, err := c.ValueUnmarshaler(data); err != nil {
				log.Warnf("MultiGet: ValueUnmarshaler: %s", err.Error())
				values[i] = nil
				continue
			} else {
				values[i] = value
				args2[2*i+1] = score
				args2[2*i+2] = fields[i]
			}
		}
		if len(args2) > 0 {
			if err := c.client.SendCommand("ZADD", args2...); err != nil {
				return nil, err
			}
		}

		return values, nil
	}
}

func (c *CacheRedis) memcacheReInit() {
	if c.memcacheEnable {
		c.memcache = make(map[Key]interface{})
	}
}

func (c *CacheRedis) memcacheGet(key Key) (interface{}, bool) {
	if c.memcacheEnable && c.memcache != nil {
		if value, ok := c.memcache[key]; ok {
			// log.Debug("lru_cache_redis memcacheGet:", key, updateTimestamp)
			return value, true
		}
	}

	return nil, false
}

func (c *CacheRedis) memcacheSet(key Key, value interface{}) {
	if c.memcacheEnable && c.memcache != nil {
		c.memcache[key] = value
	}
}

func (c *CacheRedis) memcacheRemove(keys ...Key) {
	if c.memcacheEnable && c.memcache != nil {
		for _, key := range keys {
			delete(c.memcache, key)
		}
	}
}
