# Simple KV 

This is a barebones KV implementation. It lets a user run an HTTP server and perform the following operations:

1. SET/PUT a Key Value pair. 
2. GET the value of a Key. 
3. UPDATE an existing key.
4. DELETE an existing key.
5. UPDATE a list of keys.


## Assumptions On the UPDATE API: 

One of the biggest impacts on my design decisions were the Update and BulkUpdate fields.

Ideally I would have asked the question : "Does an update also mean a PUT" but the fact that 
a separate API existed meant it was different. So I made the following assumption for Update:

- Only change a value that already exists. 
- If Update is called for a Key that does not exist, return 404. 
- If Bulk Update is called for a list of Keys, update keys that exist, ignore the ones that do not exist, and return 
a list of keys that exist. 

TODO: We should return a different status message if only a partial update is done.
- Allow Updates to be cancelled or timed out if they take too long. 
- Cancels should rollback bulk updates (Not available in LRU mode). 

## The KV currently starts two variants. 

1. The more performant mapcache mode (where extra keys get rejected if we reach size limits)
2. The less performant LRU cache mode where the oldest used items simply get evicted. 

## Mutexes vs Channels

For simplicity's sake I went with Mutexes as a way of assuring threadsafety. The cost of context switching
between multiple channels talking to goroutines is something that could might not offer much more benefits
than mutex contention. I could be wrong here of course and the best way to tell is probably by benchmarking.

I have a syncmap implementation TODO where I want to perfrom

