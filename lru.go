package kv

import (
	"container/list"
	"context"
	"sync"
)

type entry struct {
	key   string
	value interface{}
}

type lru struct {
	mu         sync.RWMutex
	ll         *list.List
	elementMap map[string]*list.Element
	size       int
}

// This is an LRU cache implementation of the KVStore interface.
// In this implementation, we evict older entries when the cache is full.
// A good time to use this implementation is as a cache where it is okay
// to lose data.
func NewLRUCacheStore(size int) *lru {
	return &lru{
		ll:         list.New(),
		size:       size,
		elementMap: make(map[string]*list.Element),
	}
}

func (l *lru) Get(key string) (interface{}, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	elem, ok := l.elementMap[key]
	if !ok {
		return nil, newNotFoundError(key)
	}
	l.ll.MoveToFront(elem)
	return elem.Value.(*entry).value, nil
}

func (l *lru) Put(key string, value interface{}) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if elem, ok := l.elementMap[key]; ok {
		l.ll.MoveToFront(elem)
		elem.Value.(*entry).value = value
		return nil
	}

	if len(l.elementMap) >= l.size {
		l.evictLRU()
	}

	elem := l.ll.PushFront(&entry{key: key, value: value})
	l.elementMap[key] = elem
	return nil
}

func (l *lru) Delete(key string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	elem, ok := l.elementMap[key]
	if !ok {
		return newNotFoundError(key)
	}
	l.ll.Remove(elem)
	delete(l.elementMap, key)
	return nil
}

func (l *lru) Update(key string, value interface{}) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	elem, ok := l.elementMap[key]
	if !ok {
		return newNotFoundError(key)
	}
	elem.Value.(*entry).value = value
	l.ll.MoveToFront(elem)
	return nil
}

func (l *lru) BatchUpdate(ctx context.Context, pairs []Pair) ([]Pair, error) {
	updatedPairs := make([]Pair, 0)

	for _, pair := range pairs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			if err := l.Update(pair.Key, pair.Value); err != nil {
				if err, ok := err.(*notFoundError); ok && err != nil {
					continue
				}
				return nil, err
			}
			updatedPairs = append(updatedPairs, pair)
		}
	}

	return updatedPairs, nil
}

func (l *lru) evictLRU() {
	elem := l.ll.Back()
	if elem == nil {
		return
	}
	l.ll.Remove(elem)
	delete(l.elementMap, elem.Value.(*entry).key)
}
