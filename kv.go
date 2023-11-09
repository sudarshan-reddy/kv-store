package kv

import "context"

// Pair is just a quick representation of KV for batch puts
type Pair struct {
	Key   string
	Value interface{}
}

// Store represents the operations that any KV store has to implement
type Store interface {
	Get(key string) (interface{}, error)
	// Notes: An Update is also a Put
	Put(key string, value interface{}) error
	Delete(key string) error
	// BatchUpdate updates the keys that exist and ignores the ones that dont.
	BatchUpdate(ctx context.Context, pairs []Pair) (updatedPairs []Pair, err error)
	BatchUpdateAsync(pairs []Pair) error
}
