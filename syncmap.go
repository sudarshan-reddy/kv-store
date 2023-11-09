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

type WriteOptimizedMap struct {
	m  sync.RWMutex
	db map[string]interface{}

	// batched update settings
	// batchedWritesCheckInterval controls how often we check to see if the context is cancelled for
	// batch writes.
	batchedWritesCheckInterval int
	rollback                   bool
}

func NewWriteOptimizedMapStore(batchedWritesCheckInterval int, rollback bool) *WriteOptimizedMap {
	return &WriteOptimizedMap{
		db:                         make(map[string]interface{}),
		rollback:                   rollback,
		batchedWritesCheckInterval: batchedWritesCheckInterval,
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

// TODO
func (s *WriteOptimizedMap) BatchUpdateAsync(pairs []Pair) error {
	return nil
}
