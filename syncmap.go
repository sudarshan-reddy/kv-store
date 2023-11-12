// NOTE: Just a quick and dirty sharded syncmap I spun up
// to run some benchmarks. This should not be used.
// Maybe if I have time, I'd be able to do some research and
// optimise here. This is mostly to demonstrate me trying to get around
// sync.Map being inefficient for a lot of writes.
package kv

import (
	"context"
	"sync"
)

const shardCount = 32

// A function that hashes a key and returns a shard index.
// This is a simple hash function for a quick demonstration.
func getShardIndex(key string) uint32 {
	var hash uint32
	for _, char := range key {
		hash = 31*hash + uint32(char)
	}
	return hash % shardCount
}

type ShardedSyncMapStore struct {
	shards [shardCount]sync.Map
}

func NewShardedSyncMapStore() *ShardedSyncMapStore {
	return &ShardedSyncMapStore{}
}

func (s *ShardedSyncMapStore) Get(key string) (interface{}, error) {
	shard := &s.shards[getShardIndex(key)]
	value, ok := shard.Load(key)
	if !ok {
		return nil, newNotFoundError(key)
	}
	return value, nil
}

func (s *ShardedSyncMapStore) Put(key string, value interface{}) error {
	shard := &s.shards[getShardIndex(key)]
	shard.Store(key, value)
	return nil
}

func (s *ShardedSyncMapStore) Delete(key string) error {
	shard := &s.shards[getShardIndex(key)]
	shard.Delete(key)
	return nil
}

func (s *ShardedSyncMapStore) BatchUpdate(ctx context.Context, pairs []Pair) ([]Pair, error) {
	updatedPairs := make([]Pair, 0, len(pairs))
	for _, pair := range pairs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			shard := &s.shards[getShardIndex(pair.Key)]
			if _, ok := shard.Load(pair.Key); ok {
				shard.Store(pair.Key, pair.Value)
				updatedPairs = append(updatedPairs, pair)
			}
		}
	}
	return updatedPairs, nil
}
