package main

import "kv"

func main() {
	// Could have two different in memory stores.
	// ReadOptimized Store: Use an internal sync.Map implementation
	// WriteOptimized Store: Use an internal map implementation
	store := kv.NewWriteOptimizedMapStore(1, true)
	frontend := kv.NewHTTPServer(store, "localhost:11200")
	frontend.Start()
}
