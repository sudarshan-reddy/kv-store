package kv

import (
	"container/list"
	"context"
	"errors"
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

func (l *lru) BatchUpdate(ctx context.Context, pairs []Pair) ([]Pair, error) {
	updatedPairs := make([]Pair, 0)

	for _, pair := range pairs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			if err := l.Put(pair.Key, pair.Value); err != nil {
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

func (l *lru) BatchUpdateAsync(pairs []Pair) error {
	return errors.New("not implemented: See interface docs for why")
}
