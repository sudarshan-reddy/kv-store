package main

import "kv"

func main() {
	// Could have two different in memory stores.
	// ReadOptimized Store: Use an internal sync.Map implementation
	// WriteOptimized Store: Use an internal map implementation
	mapstore := kv.NewWriteOptimizedMapStore(1, true, 100)
	frontend := kv.NewHTTPServer(mapstore, "0.0.0.0:11200")
	go frontend.Start()
	lrustore := kv.NewLRUCacheStore(100)
	lruFrontend := kv.NewHTTPServer(lrustore, "0.0.0.0:11201")
	lruFrontend.Start()
}
