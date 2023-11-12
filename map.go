package kv

import (
	"context"
	"sync"
)

type notFoundError struct {
	key string
}

func (e *notFoundError) Error() string {
	return "key not found: " + e.key
}

func newNotFoundError(key string) *notFoundError {
	return &notFoundError{
		key: key,
	}
}

type kvFullError struct{}

func (e *kvFullError) Error() string {
	return "kv store is full"
}

var ErrKVFull = &kvFullError{}

// This map prefers a map locked with mutexes over a sync.Map because the sync.Map is
// optimized for when the entry for a given key is only ever written once but read many times,
// as in caches that only grow, or (2) when multiple goroutines read, write, and overwrite entries
// for disjoint sets of keys.
// TODO: I could perhaps optimize this by having a Sync.Map and use multiple goroutines to
// write for a disjoint set of keys avoiding both mutex contention and taking full advantage of a sync.Map
type WriteOptimizedMap struct {
	// Mutexes galore: I chose mutexes over channels because the complexity of channels
	// that would add to this demonstration is not worth the performance benefits at this moment.
	// In my opinion, the overhead of channel communication and and context switching between
	// goroutines might offset the benefits of removing mutexes, especially in a scenario where
	// the cache operations are not particularly contention-heavy.
	// Maybe there is value in re-writing this with a worker/actor based model in a system where
	// we write very very often (like this "WriteOptimizedMap" and  see a lot of contention on mutexes.
	// But I would prefer to benchmark and compare to make that optimization if needed.
	m  sync.RWMutex
	db map[string]interface{}

	size int
	// batched update settings
	// batchedWritesCheckInterval controls how often we check to see if the context is cancelled for
	// batch writes.
	batchedWritesCheckInterval int
	rollback                   bool
}

// NewWriteOptimizedMapStore returns a new WriteOptimizedMapStore
func NewWriteOptimizedMapStore(batchedWritesCheckInterval int, rollback bool, cacheSize int) *WriteOptimizedMap {
	return &WriteOptimizedMap{
		db:                         make(map[string]interface{}),
		rollback:                   rollback,
		batchedWritesCheckInterval: batchedWritesCheckInterval,
		size:                       cacheSize,
	}
}

func (s *WriteOptimizedMap) Get(key string) (interface{}, error) {
	s.m.RLock()
	defer s.m.RUnlock()
	if value, ok := s.db[key]; ok {
		return value, nil
	}
	return nil, newNotFoundError(key)
}

func (s *WriteOptimizedMap) Put(key string, value interface{}) error {
	s.m.Lock()
	defer s.m.Unlock()
	if len(s.db) >= s.size {
		return ErrKVFull
	}
	s.db[key] = value
	return nil
}

func (s *WriteOptimizedMap) Update(key string, value interface{}) error {
	s.m.Lock()
	defer s.m.Unlock()
	if _, exists := s.db[key]; !exists {
		return newNotFoundError(key)
	}
	s.db[key] = value
	return nil
}

func (s *WriteOptimizedMap) Delete(key string) error {
	s.m.Lock()
	defer s.m.Unlock()
	delete(s.db, key)
	return nil
}

func (s *WriteOptimizedMap) BatchUpdate(ctx context.Context, pairs []Pair) ([]Pair, error) {
	// Check if the context is already cancelled before proceeding
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.m.Lock()
	defer s.m.Unlock()

	shouldRollback := false
	if s.rollback {
		snapshot := make(map[string]interface{})
		for _, pair := range pairs {
			if originalValue, exists := s.db[pair.Key]; exists {
				snapshot[pair.Key] = originalValue
			}
		}

		// Defers are hard to reason with but this one exists because we have all the
		// rollback logic in one spot and so we can rely on the call stack to rollback
		// in case something panics.
		defer func() {
			if shouldRollback {
				for k, v := range snapshot {
					s.db[k] = v
				}
			}
		}()
	}

	updatedPairs := make([]Pair, 0)
	for i, pair := range pairs {
		// We want to ensure that we can cancel this operation if it takes too long.
		// But checking for it every iteration may not be too optimal so we check it every
		// few iterations.
		if i%s.batchedWritesCheckInterval == 0 {
			if err := ctx.Err(); err != nil {
				shouldRollback = true
				return nil, err
			}
		}

		// The exists check is because I am making an assumption. I'm assuming that
		// an UPDATE should only happen if a key already exists (unlike a PUT/SET).
		if _, exists := s.db[pair.Key]; exists {
			s.db[pair.Key] = pair.Value
			updatedPairs = append(updatedPairs, pair)
		}
	}

	return updatedPairs, nil
}
