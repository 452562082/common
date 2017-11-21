package lru

import (
	"compass/lru"
	. "compass/protocol"
	"fmt"
	"goal/log"
	"testing"
	"time"
)

func init() {
	log.MinSeverity = log.INFO
}

func newCacheRedis() lru.Cache {
	return lru.NewCacheRedis(1000, 3600, "redisCache", "127.0.0.1", 8380, "", 0,
		MarshalProbeValue, UnMarshalProbeValue, MarshalKey, UnMarshalKey, false)
}

func newProbeValue(rtt int) (*ProbeValue, error) {
	//return &ProbeValue{
	//	Type:  "1",
	//	Uri:   "",
	//	RTT:   rtt,
	//	Route: []string{"0.0.0.0:0", "172.16.1.42"},
	//}

	ppv, err := CreateProbeValue(rtt)
	if err != nil {
		return nil, err
	}

	return ppv, nil
}

func TestCacheRedis_Add(t *testing.T) {
	var cache lru.Cache = newCacheRedis()

	var key string = "key"
	val, err := newProbeValue(1)
	if err != nil {
		t.Fatal("TestCacheRedis_Add, newProbeValue failed:", err)
	}

	err = cache.Add(key, val)
	if err != nil {
		t.Fatal("TestCacheRedis_Add, CacheMem.Add failed:", err)
	}

	v, exist, err := cache.Get(key)
	if err != nil {
		t.Fatal("TestCacheRedis_Add, CacheMem.Get failed:", err)
	}

	if !exist {
		t.Fatal("TestCacheRedis_Add, can not get", key, "value")
	}

	value, ok := v.(*ProbeValue)
	if !ok {
		t.Fatal("TestCacheRedis_Add, value type is not ProbeValue point")
	}

	if value.RTT != val.RTT {
		t.Fatal("TestCacheRedis_Add, get value expect rtt", val.RTT, ", but rtt", value.RTT)
	}
}

func BenchmarkCacheRedis_Add(b *testing.B) {
	var cache lru.Cache = newCacheRedis()
	var keyhead string = "key"
	for i := 0; i < b.N; i++ {
		pv, err := newProbeValue(i)
		if err != nil {
			b.Fatal("BenchmarkCacheRedis_Add, newProbeValue failed:", err)
		}
		err = cache.Add(lru.Key(fmt.Sprintf("%s%d", keyhead, i)), pv)
		if err != nil {
			b.Fatal("BenchmarkCacheRedis_Add, CacheMem.Add failed:", err)
		}
	}
}

func TestCacheRedis_MultiAdd(t *testing.T) {
	var cache lru.Cache = newCacheRedis()

	var keys []lru.Key = []lru.Key{"key1", "key2"}

	val1, err := newProbeValue(111)
	if err != nil {
		t.Fatal("TestCacheRedis_MultiAdd, newProbeValue failed:", err)
	}
	val2, err := newProbeValue(222)
	if err != nil {
		t.Fatal("TestCacheRedis_MultiAdd, newProbeValue failed:", err)
	}

	var vals []interface{} = []interface{}{val1, val2}
	err = cache.MultiAdd(keys, vals)
	if err != nil {
		t.Fatal("TestCacheRedis_MultiAdd, CacheMem.MultiAdd failed:", err)
	}

	var getAliveFunc func(lru.Key, interface{}) = func(k lru.Key, v interface{}) {
		fmt.Println("Alive:", k, v.(*ProbeValue).RTT)
	}

	err = cache.GetAll(getAliveFunc)
	if err != nil {
		t.Fatal("TestCacheRedis_MultiAdd, CacheMem.GetAll failed:", err)
	}
}

func BenchmarkCacheRedis_MultiAdd(b *testing.B) {
	var cache lru.Cache = newCacheRedis()

	var keys []lru.Key = []lru.Key{"key1", "key2"}
	val1, err := newProbeValue(111)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_MultiAdd, newProbeValue failed:", err)
	}
	val2, err := newProbeValue(222)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_MultiAdd, newProbeValue failed:", err)
	}
	var vals []interface{} = []interface{}{val1, val2}

	for i := 0; i < b.N; i++ {
		err := cache.MultiAdd(keys, vals)
		if err != nil {
			b.Fatal("TestCacheRedis_MultiAdd, CacheMem.MultiAdd failed:", err)
		}
	}
}

func TestCacheRedis_Modify(t *testing.T) {

	var cache lru.Cache = newCacheRedis()
	var key string = "modify"
	val, err := newProbeValue(111)
	if err != nil {
		t.Fatal("TestCacheRedis_Modify, newProbeValue failed:", err)
	}
	m_val, err := newProbeValue(222)

	if err != nil {
		t.Fatal("TestCacheRedis_Modify, newProbeValue failed:", err)
	}

	err = cache.Add(key, val)
	if err != nil {
		t.Fatal("TestCacheRedis_Modify, CacheMem.Add failed:", err)
	}

	v, exist, err := cache.Get(key)
	if err != nil {
		t.Fatal("TestCacheRedis_Modify, CacheMem.Get failed:", err)
	}

	if !exist {
		t.Fatal("TestCacheRedis_Modify, can not get \"", key, "\" value")
	}

	value, ok := v.(*ProbeValue)
	if !ok {
		t.Fatal("TestCacheRedis_Modify, value type is not ProbeValue point")
	}

	if value.RTT != val.RTT {
		t.Fatal("TestCacheRedis_Modify, get value expect rtt", val.RTT, ", but rtt", value.RTT)
	}

	exist, err = cache.Modify(key, m_val)
	if err != nil {
		t.Fatal("TestCacheRedis_Modify, CacheMem.Modify failed:", err)
	}

	if !exist {

		t.Fatal("TestCacheRedis_Modify, CacheMem.Modify failed: can not find", key)
	}

	v, exist, err = cache.Get(key)
	if err != nil {
		t.Fatal("TestCacheRedis_Modify, CacheMem.Get failed:", err)
	}

	if !exist {
		t.Fatal("TestCacheRedis_Modify, can not get", key, "value")
	}

	value, ok = v.(*ProbeValue)
	if !ok {
		t.Fatal("TestCacheRedis_Modify, value type is not ProbeValue point")
	}

	if value.RTT != m_val.RTT {
		t.Fatal("TestCacheRedis_Modify, get value expect rtt", m_val.RTT, ", but rtt", value.RTT)
	}
}

func BenchmarkCacheRedis_Modify(t *testing.B) {
	var cache lru.Cache = newCacheRedis()
	var key string = "key"
	val, err := newProbeValue(1)
	if err != nil {
		t.Fatal("BenchmarkCacheRedis_Modify, newProbeValue failed:", err)
	}
	var v interface{}
	var exist bool
	err = cache.Add(key, val)
	if err != nil {
		t.Fatal("BenchmarkCacheRedis_Modify, CacheMem.Add failed:", err)
	}

	v, exist, err = cache.Get(key)
	if err != nil {
		t.Fatal("BenchmarkCacheRedis_Modify, CacheMem.Get failed:", err)
	}

	if !exist {
		t.Fatal("BenchmarkCacheRedis_Modify, can not get", key, "value rtt", val.RTT)
	}

	if v.(*ProbeValue).RTT != val.RTT {
		t.Fatal("TestCacheRedis_Modify, get value expect rtt", val.RTT, ", but rtt", v.(*ProbeValue).RTT)
	}

	for i := 0; i < t.N; i++ {
		pv, err := newProbeValue(i)
		if err != nil {
			t.Fatal("BenchmarkCacheRedis_Modify, newProbeValue failed:", err)
		}
		exist, err = cache.Modify(key, pv)
		if err != nil {
			t.Fatal("BenchmarkCacheRedis_Modify, CacheMem.Modify failed:", err)
		}

		if !exist {
			t.Fatal("BenchmarkCacheRedis_Modify, can not get", key, "value rtt", i)
		}
	}
}

func TestCacheRedis_MultiModify(t *testing.T) {
	var cache lru.Cache = newCacheRedis()

	var keys []lru.Key = []lru.Key{"key1", "key2"}

	val1, err := newProbeValue(111)
	if err != nil {
		t.Fatal("TestCacheRedis_MultiModify, newProbeValue failed:", err)
	}
	val2, err := newProbeValue(222)
	if err != nil {
		t.Fatal("TestCacheRedis_MultiModify, newProbeValue failed:", err)
	}

	var vals1 []interface{} = []interface{}{val1, val2}

	val3, err := newProbeValue(1111)
	if err != nil {
		t.Fatal("TestCacheRedis_MultiModify, newProbeValue failed:", err)
	}
	val4, err := newProbeValue(2222)
	if err != nil {
		t.Fatal("TestCacheRedis_MultiModify, newProbeValue failed:", err)
	}

	var vals2 []interface{} = []interface{}{val3, val4}

	err = cache.MultiAdd(keys, vals1)
	if err != nil {
		t.Fatal("TestCacheRedis_MultiModify, CacheMem.MultiAdd failed:", err)
	}

	err = cache.MultiModify(keys, vals2)
	if err != nil {
		t.Fatal("TestCacheRedis_MultiModify, CacheMem.MultiModify failed:", err)
	}
}

func BenchmarkCacheRedis_MultiModify(b *testing.B) {
	var cache lru.Cache = newCacheRedis()

	var keys []lru.Key = []lru.Key{"key1", "key2"}
	val1, err := newProbeValue(111)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_MultiModify, newProbeValue failed:", err)
	}
	val2, err := newProbeValue(222)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_MultiModify, newProbeValue failed:", err)
	}

	var vals1 []interface{} = []interface{}{val1, val2}

	val3, err := newProbeValue(1111)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_MultiModify, newProbeValue failed:", err)
	}
	val4, err := newProbeValue(2222)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_MultiModify, newProbeValue failed:", err)
	}

	var vals2 []interface{} = []interface{}{val3, val4}
	err = cache.MultiAdd(keys, vals1)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_MultiModify, CacheMem.MultiAdd failed:", err)
	}

	for i := 0; i < b.N; i++ {
		err = cache.MultiModify(keys, vals2)
		if err != nil {
			b.Fatal("BenchmarkCacheRedis_MultiModify, CacheMem.MultiModify failed:", err)
		}
	}
}

func TestCacheRedis_Get(t *testing.T) {
	var cache lru.Cache = newCacheRedis()

	var key string = "key"
	val, err := newProbeValue(111)
	if err != nil {
		t.Fatal("TestCacheRedis_Get, newProbeValue failed:", err)
	}

	err = cache.Add(key, val)
	if err != nil {
		t.Fatal("TestCacheRedis_Get, CacheMem.Add failed:", err)
	}

	v, exist, err := cache.Get(key)
	if err != nil {
		t.Fatal("TestCacheRedis_Get, CacheMem.Get failed:", err)
	}

	if !exist {
		t.Fatal("TestCacheRedis_Get, can not get", key, "value")
	}

	value, ok := v.(*ProbeValue)
	if !ok {
		t.Fatal("TestCacheRedis_Get, value type is not ProbeValue point")
	}

	if value.RTT != val.RTT {
		t.Fatal("TestCacheRedis_Get, get value expect rtt", val.RTT, ", but rtt", value.RTT)
	}
}

func BenchmarkCacheRedis_Get(b *testing.B) {
	var cache lru.Cache = newCacheRedis()

	var key string = "key"
	val, err := newProbeValue(111)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_Get, newProbeValue failed:", err)
	}
	err = cache.Add(key, val)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_Get, CacheMem.Add failed:", err)
	}

	for i := 0; i < b.N; i++ {

		v, exist, err := cache.Get(key)
		if err != nil {
			b.Fatal("BenchmarkCacheRedis_Get, CacheMem.Get failed:", err)
		}

		if !exist {
			b.Fatal("BenchmarkCacheRedis_Get, can not get", key, "value")
		}

		value, ok := v.(*ProbeValue)
		if !ok {
			b.Fatal("BenchmarkCacheRedis_Get, value type is not ProbeValue point")
		}

		if value.RTT != val.RTT {
			b.Fatal("BenchmarkCacheRedis_Get, get value expect rtt", val.RTT, ", but rtt", value.RTT)
		}
	}
}

func TestCacheRedis_MultiGet(t *testing.T) {
	var cache lru.Cache = newCacheRedis()

	var keys []lru.Key = []lru.Key{"key1", "key2"}
	val1, err := newProbeValue(111)
	if err != nil {
		t.Fatal("TestCacheRedis_MultiGet, newProbeValue failed:", err)
	}
	val2, err := newProbeValue(222)
	if err != nil {
		t.Fatal("TestCacheRedis_MultiGet, newProbeValue failed:", err)
	}

	var vals []interface{} = []interface{}{val1, val2}
	err = cache.MultiAdd(keys, vals)
	if err != nil {
		t.Fatal("TestCacheRedis_MultiGet, CacheMem.MultiAdd failed:", err)
	}

	_, err = cache.MultiGet(keys)
	if err != nil {
		t.Fatal("TestCacheRedis_MultiGet, CacheMem.MultiGet failed:", err)
	}
}

func BenchmarkCacheRedis_MultiGet(b *testing.B) {
	var cache lru.Cache = newCacheRedis()

	var keys []lru.Key = []lru.Key{"key1", "key2"}
	val1, err := newProbeValue(111)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_MultiGet, newProbeValue failed:", err)
	}
	val2, err := newProbeValue(222)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_MultiGet, newProbeValue failed:", err)
	}

	var vals []interface{} = []interface{}{val1, val2}
	err = cache.MultiAdd(keys, vals)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_MultiGet, CacheMem.MultiAdd failed:", err)
	}

	for i := 0; i < b.N; i++ {
		_, err = cache.MultiGet(keys)
		if err != nil {
			b.Fatal("BenchmarkCacheRedis_MultiGet, CacheMem.MultiGet failed:", err)
		}
	}
}

func TestCacheRedis_Peek(t *testing.T) {
	var cache lru.Cache = newCacheRedis()

	var key string = "key"
	val, err := newProbeValue(111)
	if err != nil {
		t.Fatal("TestCacheRedis_Peek, newProbeValue failed:", err)
	}
	err = cache.Add(key, val)
	if err != nil {
		t.Fatal("TestCacheRedis_Peek, CacheMem.Add failed:", err)
	}

	v, exist, err := cache.Peek(key)
	if err != nil {
		t.Fatal("TestCacheRedis_Peek, CacheMem.Peek failed:", err)
	}

	if !exist {
		t.Fatal("TestCacheRedis_Peek, can not get", key, "value")
	}

	value, ok := v.(*ProbeValue)
	if !ok {
		t.Fatal("TestCacheRedis_Peek, value type is not ProbeValue point")
	}

	if value.RTT != val.RTT {
		t.Fatal("TestCacheRedis_Peek, get value expect rtt", val.RTT, ", but rtt", value.RTT)
	}
}

func BenchmarkCacheRedis_Peek(b *testing.B) {
	var cache lru.Cache = newCacheRedis()

	var key string = "key"
	val, err := newProbeValue(111)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_Peek, newProbeValue failed:", err)
	}

	err = cache.Add(key, val)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_Peek, CacheMem.Add failed:", err)
	}

	for i := 0; i < b.N; i++ {

		v, exist, err := cache.Peek(key)
		if err != nil {
			b.Fatal("BenchmarkCacheRedis_Peek, CacheMem.Get failed:", err)
		}

		if !exist {
			b.Fatal("BenchmarkCacheRedis_Peek, can not get", key, "value")
		}

		value, ok := v.(*ProbeValue)
		if !ok {
			b.Fatal("BenchmarkCacheRedis_Peek, value type is not ProbeValue point")
		}

		if value.RTT != val.RTT {
			b.Fatal("BenchmarkCacheRedis_Peek, get value expect rtt", val.RTT, ", but rtt", value.RTT)
		}
	}
}

func TestCacheRedis_Purge(t *testing.T) {
	var cache lru.Cache = newCacheRedis()

	var key1, key2 string = "key1", "key2"
	val1, err := newProbeValue(111)
	if err != nil {
		t.Fatal("TestCacheRedis_Purge, newProbeValue failed:", err)
	}
	val2, err := newProbeValue(222)
	if err != nil {
		t.Fatal("TestCacheRedis_Purge, newProbeValue failed:", err)
	}

	err = cache.Add(key1, val1)
	if err != nil {
		t.Fatal("TestCacheRedis_Purge, CacheMem.Add failed:", err)
	}

	time.Sleep(2 * time.Second)

	err = cache.Add(key2, val2)
	if err != nil {
		t.Fatal("TestCacheRedis_Purge, CacheMem.Add failed:", err)
	}

	var getAliveFunc func(lru.Key, interface{}) = func(k lru.Key, v interface{}) {
		fmt.Println("Alive:", k, v.(*ProbeValue).RTT)
	}
	var getExpiredFunc func(lru.Key, interface{}) = func(k lru.Key, v interface{}) {
		fmt.Println("Expired:", k, v.(*ProbeValue).RTT)
	}

	err = cache.Purge(getAliveFunc, getExpiredFunc)
	if err != nil {
		t.Fatal("TestCacheRedis_Purge, CacheMem.Purge failed:", err)
	}
}

func BenchmarkCacheRedis_Purge(b *testing.B) {
	var cache lru.Cache = newCacheRedis()

	var key1, key2 string = "key1", "key2"
	val1, err := newProbeValue(111)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_Purge, newProbeValue failed:", err)
	}
	val2, err := newProbeValue(222)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_Purge, newProbeValue failed:", err)
	}
	err = cache.Add(key1, val1)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_Purge, CacheMem.Add failed:", err)
	}

	err = cache.Add(key2, val2)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_Purge, CacheMem.Add failed:", err)
	}

	for i := 0; i < b.N; i++ {
		err = cache.Purge(nil, nil)
		if err != nil {
			b.Fatal("BenchmarkCacheRedis_Purge, CacheMem.Purge failed:", err)
		}
	}
}

func TestCacheRedis_GetAll(t *testing.T) {
	var cache lru.Cache = newCacheRedis()

	var key1, key2 string = "key1", "key2"
	val1, err := newProbeValue(111)
	if err != nil {
		t.Fatal("TestCacheRedis_GetAll, newProbeValue failed:", err)
	}
	val2, err := newProbeValue(222)
	if err != nil {
		t.Fatal("TestCacheRedis_GetAll, newProbeValue failed:", err)
	}

	err = cache.Add(key1, val1)
	if err != nil {
		t.Fatal("TestCacheRedis_GetAll, CacheMem.Add failed:", err)
	}

	err = cache.Add(key2, val2)
	if err != nil {
		t.Fatal("TestCacheRedis_GetAll, CacheMem.Add failed:", err)
	}

	var getAliveFunc func(lru.Key, interface{}) = func(k lru.Key, v interface{}) {
		fmt.Println("Alive:", k, v.(*ProbeValue).RTT)
	}
	err = cache.GetAll(getAliveFunc)
	if err != nil {
		t.Fatal("TestCacheRedis_GetAll, CacheMem.GetAll failed:", err)
	}
}

func BenchmarkCacheRedis_GetAll(b *testing.B) {
	var cache lru.Cache = newCacheRedis()

	var key1, key2 string = "key1", "key2"
	val1, err := newProbeValue(111)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_GetAll, newProbeValue failed:", err)
	}
	val2, err := newProbeValue(222)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_GetAll, newProbeValue failed:", err)
	}
	err = cache.Add(key1, val1)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_GetAll, CacheMem.Add failed:", err)
	}

	err = cache.Add(key2, val2)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_GetAll, CacheMem.Add failed:", err)
	}
	for i := 0; i < b.N; i++ {
		err = cache.GetAll(nil)
		if err != nil {
			b.Fatal("BenchmarkCacheRedis_GetAll, CacheMem.GetAll failed:", err)
		}
	}
}

func TestCacheRedis_Remove(t *testing.T) {
	var cache lru.Cache = newCacheRedis()

	var key1, key2 string = "key1", "key2"
	val1, err := newProbeValue(111)
	if err != nil {
		t.Fatal("TestCacheRedis_Remove, newProbeValue failed:", err)
	}
	val2, err := newProbeValue(222)
	if err != nil {
		t.Fatal("TestCacheRedis_Remove, newProbeValue failed:", err)
	}
	err = cache.Add(key1, val1)
	if err != nil {
		t.Fatal("TestCacheRedis_Remove, CacheMem.Add failed:", err)
	}

	err = cache.Add(key2, val2)
	if err != nil {
		t.Fatal("TestCacheRedis_Remove, CacheMem.Add failed:", err)
	}

	err = cache.Remove(key1, key2)
	if err != nil {
		t.Fatal("TestCacheRedis_Remove, CacheMem.Remove failed:", err)
	}

	_, exist, _ := cache.Get(key1)
	if exist {
		t.Fatal("TestCacheRedis_Remove, CacheMem.Remove \"", key1, "\"failed")
	}

	_, exist, _ = cache.Get(key2)
	if exist {
		t.Fatal("TestCacheRedis_Remove, CacheMem.Remove \"", key2, "\"failed")
	}
}

func BenchmarkCacheRedis_Remove(b *testing.B) {
	var cache lru.Cache = newCacheRedis()

	var key1, key2 string = "key1", "key2"
	val1, err := newProbeValue(111)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_Remove, newProbeValue failed:", err)
	}
	val2, err := newProbeValue(222)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_Remove, newProbeValue failed:", err)
	}
	err = cache.Add(key1, val1)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_Remove, CacheMem.Add failed:", err)
	}

	err = cache.Add(key2, val2)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_Remove, CacheMem.Add failed:", err)
	}
	for i := 0; i < b.N; i++ {
		err = cache.Remove(key1, key2)
		if err != nil {
			b.Fatal("BenchmarkCacheRedis_Remove, CacheMem.Remove failed:", err)
		}
	}
}

func TestCacheRedis_RemoveAll(t *testing.T) {
	var cache lru.Cache = newCacheRedis()

	var key1, key2 string = "key1", "key2"
	val1, err := newProbeValue(111)
	if err != nil {
		t.Fatal("TestCacheRedis_RemoveAll, newProbeValue failed:", err)
	}
	val2, err := newProbeValue(222)
	if err != nil {
		t.Fatal("TestCacheRedis_RemoveAll, newProbeValue failed:", err)
	}
	err = cache.Add(key1, val1)
	if err != nil {
		t.Fatal("TestCacheRedis_RemoveAll, CacheMem.Add failed:", err)
	}

	err = cache.Add(key2, val2)
	if err != nil {
		t.Fatal("TestCacheRedis_RemoveAll, CacheMem.Add failed:", err)
	}

	err = cache.RemoveAll()
	if err != nil {
		t.Fatal("TestCacheRedis_RemoveAll, CacheMem.RemoveAll failed:", err)
	}

	_, exist, _ := cache.Get(key1)
	if exist {
		t.Fatal("TestCacheRedis_RemoveAll, CacheMem.RemoveAll \"", key1, "\"failed")
	}

	_, exist, _ = cache.Get(key2)
	if exist {
		t.Fatal("TestCacheRedis_RemoveAll, CacheMem.RemoveAll \"", key2, "\"failed")
	}
}

func BenchmarkCacheRedis_RemoveAll(b *testing.B) {
	var cache lru.Cache = newCacheRedis()

	var key1, key2 string = "key1", "key2"
	val1, err := newProbeValue(111)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_RemoveAll, newProbeValue failed:", err)
	}
	val2, err := newProbeValue(222)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_RemoveAll, newProbeValue failed:", err)
	}
	err = cache.Add(key1, val1)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_RemoveAll, CacheMem.Add failed:", err)
	}

	err = cache.Add(key2, val2)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_RemoveAll, CacheMem.Add failed:", err)
	}
	for i := 0; i < b.N; i++ {
		err = cache.RemoveAll()
		if err != nil {
			b.Fatal("BenchmarkCacheRedis_RemoveAll, CacheMem.RemoveAll failed:", err)
		}
	}
}

func TestCacheRedis_RemoveOldest(t *testing.T) {
	var cache lru.Cache = newCacheRedis()

	var key1, key2 string = "key1", "key2"
	val1, err := newProbeValue(111)
	if err != nil {
		t.Fatal("TestCacheRedis_RemoveOldest, newProbeValue failed:", err)
	}
	val2, err := newProbeValue(222)
	if err != nil {
		t.Fatal("TestCacheRedis_RemoveOldest, newProbeValue failed:", err)
	}
	err = cache.Add(key1, val1)
	if err != nil {
		t.Fatal("TestCacheRedis_RemoveOldest, CacheMem.Add failed:", err)
	}

	err = cache.Add(key2, val2)
	if err != nil {
		t.Fatal("TestCacheRedis_RemoveOldest, CacheMem.Add failed:", err)
	}

	err = cache.RemoveOldest()
	if err != nil {
		t.Fatal("TestCacheRedis_RemoveOldest, CacheMem.RemoveOldest failed:", err)
	}

	_, exist, _ := cache.Get(key1)
	if exist {
		t.Fatal("TestCacheRedis_RemoveOldest, CacheMem.RemoveOldest \"", key1, "\"failed")
	}

	_, exist, _ = cache.Get(key2)
	if !exist {
		t.Fatal("TestCacheRedis_RemoveOldest, CacheMem.RemoveOldest \"", key2, "\"failed")
	}
}

func BenchmarkCacheRedis_RemoveOldest(b *testing.B) {
	var cache lru.Cache = newCacheRedis()

	var key1, key2 string = "key1", "key2"
	val1, err := newProbeValue(111)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_RemoveOldest, newProbeValue failed:", err)
	}
	val2, err := newProbeValue(222)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_RemoveOldest, newProbeValue failed:", err)
	}
	err = cache.Add(key1, val1)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_RemoveOldest, CacheMem.Add failed:", err)
	}

	err = cache.Add(key2, val2)
	if err != nil {
		b.Fatal("BenchmarkCacheRedis_RemoveOldest, CacheMem.Add failed:", err)
	}
	for i := 0; i < b.N; i++ {
		err = cache.RemoveOldest()
		if err != nil {
			b.Fatal("BenchmarkCacheRedis_RemoveAll, CacheMem.RemoveOldest failed:", err)
		}
	}
}
