package kv

import (
	"context"
	"strconv"
	"testing"
)

func BenchmarkShardedSyncMapStore_Put(b *testing.B) {
	store := NewShardedSyncMapStore()
	for i := 0; i < b.N; i++ {
		key := "key" + strconv.Itoa(i)
		store.Put(key, i)
	}
}

func BenchmarkShardedSyncMapStore_Get(b *testing.B) {
	store := NewShardedSyncMapStore()
	// Prepopulate the store
	for i := 0; i < 1000; i++ {
		key := "key" + strconv.Itoa(i)
		store.Put(key, i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "key" + strconv.Itoa(i%1000)
		store.Get(key)
	}
}

func BenchmarkShardedSyncMapStore_BatchUpdate(b *testing.B) {
	store := NewShardedSyncMapStore()
	pairs := make([]Pair, 1000)
	for i := 0; i < 1000; i++ {
		pairs[i] = Pair{Key: "key" + strconv.Itoa(i), Value: i}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.BatchUpdate(context.Background(), pairs)
	}
}
