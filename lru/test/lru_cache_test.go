package lru

import (
	"fmt"
	"goal/log"
	"testing"
	"time"
	"compass/lru"
)

func TestCacheMem_Add(t *testing.T) {
	var cache lru.Cache = lru.NewCacheMem("test", 100, 10)

	var key string = "key"
	var val int = 111
	err := cache.Add(key, val)
	if err != nil {
		t.Fatal("TestCacheMem_Add, CacheMem.Add failed:", err)
	}

	v, exist, err := cache.Get(key)
	if err != nil {
		t.Fatal("TestCacheMem_Add, CacheMem.Get failed:", err)
	}

	if !exist {
		t.Fatal("TestCacheMem_Add, can not get", key, "value")
	}

	value, ok := v.(int)
	if !ok {
		t.Fatal("TestCacheMem_Add, value type is not integer")
	}

	if value != val {
		t.Fatal("TestCacheMem_Add, get value expect", val, ", but", value)
	}
}

func BenchmarkCacheMem_Add(b *testing.B) {
	var cache lru.Cache = lru.NewCacheMem("test", -1, 10)
	var keyhead string = "key"
	log.MinSeverity = log.INFO
	for i := 0; i < b.N; i++ {
		err := cache.Add(fmt.Sprintf("%s%d", keyhead, i), i)
		if err != nil {
			b.Fatal("BenchmarkCacheMem_Add, CacheMem.Add failed:", err)
		}
	}
}

func TestCacheMem_MultiAdd(t *testing.T) {
	var cache lru.Cache = lru.NewCacheMem("test", 100, 10)

	var keys []lru.Key = []lru.Key{"key1", "key2"}
	var vals []interface{} = []interface{}{111, 222}
	err := cache.MultiAdd(keys, vals)
	if err != nil {
		t.Fatal("TestCacheMem_MultiAdd, CacheMem.MultiAdd failed:", err)
	}

	var getAliveFunc func(lru.Key, interface{}) = func(k lru.Key, v interface{}) {
		fmt.Println("Alive:", k, v)
	}

	err = cache.GetAll(getAliveFunc)
	if err != nil {
		t.Fatal("TestCacheMem_MultiAdd, CacheMem.GetAll failed:", err)
	}
}

func BenchmarkCacheMem_MultiAdd(b *testing.B) {
	var cache lru.Cache = lru.NewCacheMem("test", 100, 10)

	var keys []lru.Key = []lru.Key{"key1", "key2"}
	var vals []interface{} = []interface{}{111, 222}

	for i := 0; i < b.N; i++ {
		err := cache.MultiAdd(keys, vals)
		if err != nil {
			b.Fatal("TestCacheMem_MultiAdd, CacheMem.MultiAdd failed:", err)
		}
	}
}

func TestCacheMem_Modify(t *testing.T) {

	var cache lru.Cache = lru.NewCacheMem("testm", 100, 10000)
	var key string = "modify"
	var val int = 111
	var m_val int = 222

	err := cache.Add(key, val)
	if err != nil {
		t.Fatal("TestCacheMem_Modify, CacheMem.Add failed:", err)
	}

	v, exist, err := cache.Get(key)
	if err != nil {
		t.Fatal("TestCacheMem_Modify, CacheMem.Get failed:", err)
	}

	if !exist {
		t.Fatal("TestCacheMem_Modify, can not get \"", key, "\" value")
	}

	value, ok := v.(int)
	if !ok {
		t.Fatal("TestCacheMem_Modify, value type is not integer")
	}

	if value != val {
		t.Fatal("TestCacheMem_Modify, get value expect", val, ", but", value)
	}

	exist, err = cache.Modify(key, m_val)
	if err != nil {
		t.Fatal("TestCacheMem_Modify, CacheMem.Modify failed:", err)
	}

	if !exist {

		t.Fatal("TestCacheMem_Modify, CacheMem.Modify failed: can not find", key)
	}

	v, exist, err = cache.Get(key)
	if err != nil {
		t.Fatal("TestCacheMem_Modify, CacheMem.Get failed:", err)
	}

	if !exist {
		t.Fatal("TestCacheMem_Modify, can not get", key, "value")
	}

	value, ok = v.(int)
	if !ok {
		t.Fatal("TestCacheMem_Modify, value type is not integer")
	}

	if value != m_val {
		t.Fatal("TestCacheMem_Modify, get value expect", m_val, ", but", value)
	}
}

func BenchmarkCacheMem_Modify(t *testing.B) {
	var cache lru.Cache = lru.NewCacheMem("test", 100, 10)
	log.MinSeverity = log.INFO
	var key string = "key"
	var val int = 111
	var v interface{}
	var exist bool
	err := cache.Add(key, val)
	if err != nil {
		t.Fatal("BenchmarkCacheMem_Modify, CacheMem.Add failed:", err)
	}

	v, exist, err = cache.Get(key)
	if err != nil {
		t.Fatal("BenchmarkCacheMem_Modify, CacheMem.Get failed:", err)
	}

	if !exist {
		t.Fatal("BenchmarkCacheMem_Modify, can not get", key, "value", v.(int))
	}

	for i := 0; i < t.N; i++ {
		exist, err = cache.Modify(key, i)
		if err != nil {
			t.Fatal("BenchmarkCacheMem_Modify, CacheMem.Modify failed:", err)
		}

		if !exist {
			t.Fatal("BenchmarkCacheMem_Modify, can not get", key, "value", i)
		}
	}
}

func TestCacheMem_MultiModify(t *testing.T) {
	var cache lru.Cache = lru.NewCacheMem("test", 100, 10)

	var keys []lru.Key = []lru.Key{"key1", "key2"}
	var vals1 []interface{} = []interface{}{111, 222}
	var vals2 []interface{} = []interface{}{1111, 2222}
	err := cache.MultiAdd(keys, vals1)
	if err != nil {
		t.Fatal("TestCacheMem_MultiModify, CacheMem.MultiAdd failed:", err)
	}

	err = cache.MultiModify(keys, vals2)
	if err != nil {
		t.Fatal("TestCacheMem_MultiModify, CacheMem.MultiModify failed:", err)
	}
}

func BenchmarkCacheMem_MultiModify(b *testing.B) {
	var cache lru.Cache = lru.NewCacheMem("test", 100, 10)

	var keys []lru.Key = []lru.Key{"key1", "key2"}
	var vals1 []interface{} = []interface{}{111, 222}
	var vals2 []interface{} = []interface{}{1111, 2222}
	err := cache.MultiAdd(keys, vals1)
	if err != nil {
		b.Fatal("BenchmarkCacheMem_MultiModify, CacheMem.MultiAdd failed:", err)
	}

	for i := 0; i < b.N; i++ {
		err = cache.MultiModify(keys, vals2)
		if err != nil {
			b.Fatal("BenchmarkCacheMem_MultiModify, CacheMem.MultiModify failed:", err)
		}
	}
}

func TestCacheMem_Get(t *testing.T) {
	var cache lru.Cache = lru.NewCacheMem("test", 100, 10)

	var key string = "key"
	var val int = 111
	err := cache.Add(key, val)
	if err != nil {
		t.Fatal("TestCacheMem_Get, CacheMem.Add failed:", err)
	}

	v, exist, err := cache.Get(key)
	if err != nil {
		t.Fatal("TestCacheMem_Get, CacheMem.Get failed:", err)
	}

	if !exist {
		t.Fatal("TestCacheMem_Get, can not get", key, "value")
	}

	value, ok := v.(int)
	if !ok {
		t.Fatal("TestCacheMem_Get, value type is not integer")
	}

	if value != val {
		t.Fatal("TestCacheMem_Get, get value expect", val, ", but", value)
	}
}

func BenchmarkCacheMem_Get(b *testing.B) {
	var cache lru.Cache = lru.NewCacheMem("test", 100, 10)

	var key string = "key"
	var val int = 111
	err := cache.Add(key, val)
	if err != nil {
		b.Fatal("BenchmarkCacheMem_Get, CacheMem.Add failed:", err)
	}

	for i := 0; i < b.N; i++ {

		v, exist, err := cache.Get(key)
		if err != nil {
			b.Fatal("BenchmarkCacheMem_Get, CacheMem.Get failed:", err)
		}

		if !exist {
			b.Fatal("BenchmarkCacheMem_Get, can not get", key, "value")
		}

		value, ok := v.(int)
		if !ok {
			b.Fatal("BenchmarkCacheMem_Get, value type is not integer")
		}

		if value != val {
			b.Fatal("BenchmarkCacheMem_Get, get value expect", val, ", but", value)
		}
	}
}

func TestCacheMem_MultiGet(t *testing.T) {
	var cache lru.Cache = lru.NewCacheMem("test", 100, 10)

	var keys []lru.Key = []lru.Key{"key1", "key2"}
	var vals []interface{} = []interface{}{111, 222}
	err := cache.MultiAdd(keys, vals)
	if err != nil {
		t.Fatal("TestCacheMem_MultiGet, CacheMem.MultiAdd failed:", err)
	}

	_, err = cache.MultiGet(keys)
	if err != nil {
		t.Fatal("TestCacheMem_MultiGet, CacheMem.MultiGet failed:", err)
	}
}

func BenchmarkCacheMem_MultiGet(b *testing.B) {
	var cache lru.Cache = lru.NewCacheMem("test", 100, 10)

	var keys []lru.Key = []lru.Key{"key1", "key2"}
	var vals []interface{} = []interface{}{111, 222}
	err := cache.MultiAdd(keys, vals)
	if err != nil {
		b.Fatal("BenchmarkCacheMem_MultiGet, CacheMem.MultiAdd failed:", err)
	}

	for i := 0; i < b.N; i++ {
		_, err = cache.MultiGet(keys)
		if err != nil {
			b.Fatal("BenchmarkCacheMem_MultiGet, CacheMem.MultiGet failed:", err)
		}
	}
}

func TestCacheMem_Peek(t *testing.T) {
	var cache lru.Cache = lru.NewCacheMem("test", 100, 10)

	var key string = "key"
	var val int = 111
	err := cache.Add(key, val)
	if err != nil {
		t.Fatal("TestCacheMem_Peek, CacheMem.Add failed:", err)
	}

	v, exist, err := cache.Peek(key)
	if err != nil {
		t.Fatal("TestCacheMem_Peek, CacheMem.Peek failed:", err)
	}

	if !exist {
		t.Fatal("TestCacheMem_Peek, can not get", key, "value")
	}

	value, ok := v.(int)
	if !ok {
		t.Fatal("TestCacheMem_Peek, value type is not integer")
	}

	if value != val {
		t.Fatal("TestCacheMem_Peek, get value expect", val, ", but", value)
	}
}

func BenchmarkCacheMem_Peek(b *testing.B) {
	var cache lru.Cache = lru.NewCacheMem("test", 100, 10)

	var key string = "key"
	var val int = 111
	err := cache.Add(key, val)
	if err != nil {
		b.Fatal("BenchmarkCacheMem_Peek, CacheMem.Add failed:", err)
	}

	for i := 0; i < b.N; i++ {

		v, exist, err := cache.Peek(key)
		if err != nil {
			b.Fatal("BenchmarkCacheMem_Peek, CacheMem.Get failed:", err)
		}

		if !exist {
			b.Fatal("BenchmarkCacheMem_Peek, can not get", key, "value")
		}

		value, ok := v.(int)
		if !ok {
			b.Fatal("BenchmarkCacheMem_Peek, value type is not integer")
		}

		if value != val {
			b.Fatal("BenchmarkCacheMem_Peek, get value expect", val, ", but", value)
		}
	}
}

func TestCacheMem_Purge(t *testing.T) {
	var cache lru.Cache = lru.NewCacheMem("test", 100, 1)

	var key1, key2 string = "key1", "key2"
	var val1, val2 int = 111, 222
	err := cache.Add(key1, val1)
	if err != nil {
		t.Fatal("TestCacheMem_Purge, CacheMem.Add failed:", err)
	}

	time.Sleep(2 * time.Second)

	err = cache.Add(key2, val2)
	if err != nil {
		t.Fatal("TestCacheMem_Purge, CacheMem.Add failed:", err)
	}

	var getAliveFunc func(lru.Key, interface{}) = func(k lru.Key, v interface{}) {
		fmt.Println("Alive:", k, v)
	}
	var getExpiredFunc func(lru.Key, interface{}) = func(k lru.Key, v interface{}) {
		fmt.Println("Expired:", k, v)
	}

	err = cache.Purge(getAliveFunc, getExpiredFunc)
	if err != nil {
		t.Fatal("TestCacheMem_Purge, CacheMem.Purge failed:", err)
	}
}

func BenchmarkCacheMem_Purge(b *testing.B) {
	var cache lru.Cache = lru.NewCacheMem("test", 100, 1)

	var key1, key2 string = "key1", "key2"
	var val1, val2 int = 111, 222
	err := cache.Add(key1, val1)
	if err != nil {
		b.Fatal("BenchmarkCacheMem_Purge, CacheMem.Add failed:", err)
	}

	err = cache.Add(key2, val2)
	if err != nil {
		b.Fatal("BenchmarkCacheMem_Purge, CacheMem.Add failed:", err)
	}

	for i := 0; i < b.N; i++ {
		err = cache.Purge(nil, nil)
		if err != nil {
			b.Fatal("BenchmarkCacheMem_Purge, CacheMem.Purge failed:", err)
		}
	}
}

func TestCacheMem_GetAll(t *testing.T) {
	var cache lru.Cache = lru.NewCacheMem("test", 100, 1)

	var key1, key2 string = "key1", "key2"
	var val1, val2 int = 111, 222
	err := cache.Add(key1, val1)
	if err != nil {
		t.Fatal("TestCacheMem_GetAll, CacheMem.Add failed:", err)
	}

	err = cache.Add(key2, val2)
	if err != nil {
		t.Fatal("TestCacheMem_GetAll, CacheMem.Add failed:", err)
	}

	var getAliveFunc func(lru.Key, interface{}) = func(k lru.Key, v interface{}) {
		fmt.Println("Alive:", k, v)
	}
	err = cache.GetAll(getAliveFunc)
	if err != nil {
		t.Fatal("TestCacheMem_GetAll, CacheMem.GetAll failed:", err)
	}
}

func BenchmarkCacheMem_GetAll(b *testing.B) {
	var cache lru.Cache = lru.NewCacheMem("test", 100, 1)

	var key1, key2 string = "key1", "key2"
	var val1, val2 int = 111, 222
	err := cache.Add(key1, val1)
	if err != nil {
		b.Fatal("BenchmarkCacheMem_GetAll, CacheMem.Add failed:", err)
	}

	err = cache.Add(key2, val2)
	if err != nil {
		b.Fatal("BenchmarkCacheMem_GetAll, CacheMem.Add failed:", err)
	}
	for i := 0; i < b.N; i++ {
		err = cache.GetAll(nil)
		if err != nil {
			b.Fatal("BenchmarkCacheMem_GetAll, CacheMem.GetAll failed:", err)
		}
	}
}

func TestCacheMem_Remove(t *testing.T) {
	var cache lru.Cache = lru.NewCacheMem("test", 100, 1)

	var key1, key2 string = "key1", "key2"
	var val1, val2 int = 111, 222
	err := cache.Add(key1, val1)
	if err != nil {
		t.Fatal("TestCacheMem_Remove, CacheMem.Add failed:", err)
	}

	err = cache.Add(key2, val2)
	if err != nil {
		t.Fatal("TestCacheMem_Remove, CacheMem.Add failed:", err)
	}

	err = cache.Remove(key1, key2)
	if err != nil {
		t.Fatal("TestCacheMem_Remove, CacheMem.Remove failed:", err)
	}

	_, exist, _ := cache.Get(key1)
	if exist {
		t.Fatal("TestCacheMem_Remove, CacheMem.Remove \"", key1, "\"failed")
	}

	_, exist, _ = cache.Get(key2)
	if exist {
		t.Fatal("TestCacheMem_Remove, CacheMem.Remove \"", key2, "\"failed")
	}
}

func BenchmarkCacheMem_Remove(b *testing.B) {
	var cache lru.Cache = lru.NewCacheMem("test", 100, 1)

	var key1, key2 string = "key1", "key2"
	var val1, val2 int = 111, 222
	err := cache.Add(key1, val1)
	if err != nil {
		b.Fatal("BenchmarkCacheMem_Remove, CacheMem.Add failed:", err)
	}

	err = cache.Add(key2, val2)
	if err != nil {
		b.Fatal("BenchmarkCacheMem_Remove, CacheMem.Add failed:", err)
	}
	for i := 0; i < b.N; i++ {
		err = cache.Remove(key1, key2)
		if err != nil {
			b.Fatal("BenchmarkCacheMem_Remove, CacheMem.Remove failed:", err)
		}
	}
}

func TestCacheMem_RemoveAll(t *testing.T) {
	var cache lru.Cache = lru.NewCacheMem("test", 100, 1)

	var key1, key2 string = "key1", "key2"
	var val1, val2 int = 111, 222
	err := cache.Add(key1, val1)
	if err != nil {
		t.Fatal("TestCacheMem_RemoveAll, CacheMem.Add failed:", err)
	}

	err = cache.Add(key2, val2)
	if err != nil {
		t.Fatal("TestCacheMem_RemoveAll, CacheMem.Add failed:", err)
	}

	err = cache.RemoveAll()
	if err != nil {
		t.Fatal("TestCacheMem_RemoveAll, CacheMem.RemoveAll failed:", err)
	}

	_, exist, _ := cache.Get(key1)
	if exist {
		t.Fatal("TestCacheMem_RemoveAll, CacheMem.RemoveAll \"", key1, "\"failed")
	}

	_, exist, _ = cache.Get(key2)
	if exist {
		t.Fatal("TestCacheMem_RemoveAll, CacheMem.RemoveAll \"", key2, "\"failed")
	}
}

func BenchmarkCacheMem_RemoveAll(b *testing.B) {
	var cache lru.Cache = lru.NewCacheMem("test", 100, 1)

	var key1, key2 string = "key1", "key2"
	var val1, val2 int = 111, 222
	err := cache.Add(key1, val1)
	if err != nil {
		b.Fatal("BenchmarkCacheMem_RemoveAll, CacheMem.Add failed:", err)
	}

	err = cache.Add(key2, val2)
	if err != nil {
		b.Fatal("BenchmarkCacheMem_RemoveAll, CacheMem.Add failed:", err)
	}
	for i := 0; i < b.N; i++ {
		err = cache.RemoveAll()
		if err != nil {
			b.Fatal("BenchmarkCacheMem_RemoveAll, CacheMem.RemoveAll failed:", err)
		}
	}
}

func TestCacheMem_RemoveOldest(t *testing.T) {
	var cache lru.Cache = lru.NewCacheMem("test", 100, 1)

	var key1, key2 string = "key1", "key2"
	var val1, val2 int = 111, 222
	err := cache.Add(key1, val1)
	if err != nil {
		t.Fatal("TestCacheMem_RemoveOldest, CacheMem.Add failed:", err)
	}

	err = cache.Add(key2, val2)
	if err != nil {
		t.Fatal("TestCacheMem_RemoveOldest, CacheMem.Add failed:", err)
	}

	err = cache.RemoveOldest()
	if err != nil {
		t.Fatal("TestCacheMem_RemoveOldest, CacheMem.RemoveOldest failed:", err)
	}

	_, exist, _ := cache.Get(key1)
	if exist {
		t.Fatal("TestCacheMem_RemoveOldest, CacheMem.RemoveOldest \"", key1, "\"failed")
	}

	_, exist, _ = cache.Get(key2)
	if !exist {
		t.Fatal("TestCacheMem_RemoveOldest, CacheMem.RemoveOldest \"", key2, "\"failed")
	}
}

func BenchmarkCacheMem_RemoveOldest(b *testing.B) {
	var cache lru.Cache = lru.NewCacheMem("test", 100, 1)

	var key1, key2 string = "key1", "key2"
	var val1, val2 int = 111, 222
	err := cache.Add(key1, val1)
	if err != nil {
		b.Fatal("BenchmarkCacheMem_RemoveOldest, CacheMem.Add failed:", err)
	}

	err = cache.Add(key2, val2)
	if err != nil {
		b.Fatal("BenchmarkCacheMem_RemoveOldest, CacheMem.Add failed:", err)
	}
	for i := 0; i < b.N; i++ {
		err = cache.RemoveOldest()
		if err != nil {
			b.Fatal("BenchmarkCacheMem_RemoveAll, CacheMem.RemoveOldest failed:", err)
		}
	}
}
