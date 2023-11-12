package kv

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"
)

func TestLRUCache_GetPut(t *testing.T) {
	cache := NewLRUCacheStore(2)

	// Test Put
	err := cache.Put("key1", "value1")
	if err != nil {
		t.Fatalf("Put returned an error: %v", err)
	}

	// Test Get
	value, err := cache.Get("key1")
	if err != nil || value != "value1" {
		t.Fatalf("Get returned an unexpected result: %v, %v", value, err)
	}

	// Test eviction
	cache.Put("key2", "value2")
	cache.Put("key3", "value3") // This should evict "key1"

	_, err = cache.Get("key1")
	if err == nil {
		t.Fatal("Expected an error for key1 as it should have been evicted")
	}
}

func TestLRUCache_ConcurrentAccess(t *testing.T) {
	cache := NewLRUCacheStore(1000)
	var wg sync.WaitGroup

	// Test concurrent access
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", i)
			cache.Put(key, i)
			if _, err := cache.Get(key); err != nil {
				t.Errorf("Error getting key: %v", err)
			}
		}(i)
	}

	wg.Wait()
}

func TestLRUCache_BatchUpdate(t *testing.T) {
	cache := NewLRUCacheStore(10)

	pairs := []Pair{
		{"key1", "value1"},
		{"key2", "value2"},
	}

	updatedPairs, err := cache.BatchUpdate(context.Background(), pairs)
	if err != nil {
		t.Fatalf("BatchUpdate returned an error: %v", err)
	}
	if len(updatedPairs) != 2 {
		t.Fatalf("BatchUpdate did not return correct number of updated pairs")
	}

	for _, pair := range pairs {
		value, _ := cache.Get(pair.Key)
		if value != pair.Value {
			t.Fatalf("BatchUpdate did not update the pair correctly")
		}
	}
}

func BenchmarkLRUCachePut(b *testing.B) {
	cacheSize := 1000
	cache := NewLRUCacheStore(cacheSize)

	for i := 0; i < b.N; i++ {
		key := "key" + strconv.Itoa(i%cacheSize)
		cache.Put(key, i)
	}
}

func BenchmarkLRUCacheGet(b *testing.B) {
	cacheSize := 1000
	cache := NewLRUCacheStore(1000)

	// Prepopulate the cache
	for i := 0; i < cacheSize; i++ {
		key := "key" + strconv.Itoa(i)
		cache.Put(key, i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "key" + strconv.Itoa(i%cacheSize)
		cache.Get(key)
	}
}

func BenchmarkLRUCacheUpdate(b *testing.B) {
	cacheSize := 1000
	cache := NewLRUCacheStore(cacheSize)

	// Prepopulate the cache
	for i := 0; i < cacheSize; i++ {
		key := "key" + strconv.Itoa(i)
		cache.Put(key, i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "key" + strconv.Itoa(i%cacheSize)
		cache.Put(key, i) // Using Put for Update as LRU cache does not differentiate
	}
}

func BenchmarkLRUCacheBatchUpdate(b *testing.B) {
	cacheSize := 1000
	cache := NewLRUCacheStore(cacheSize)

	pairs := make([]Pair, cacheSize)
	for i := 0; i < cacheSize; i++ {
		pairs[i] = Pair{Key: "key" + strconv.Itoa(i), Value: i}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.BatchUpdate(context.Background(), pairs)
	}
}
