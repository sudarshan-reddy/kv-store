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
	// DO NOT DO: The idea I had here was to run a worker pool that would process the batch updates
	// asynchronously. Calls to this API will simply queue a job to the worker pool and return.
	// I didn't have the time to do this because that opens up a whole layer of complexity (e.g
	// manage the workers, compute time over duplicate jobs.
	// Further more, this also introduces a lot of issues with Set and Get values being raced by scheduled jobs.
	// Doing this would make this system hard to reason with. This function simply exists to document this
	// thought process.
	BatchUpdateAsync(pairs []Pair) error
}
