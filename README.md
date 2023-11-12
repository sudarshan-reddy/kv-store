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


## Benchmarks

I have three different implementations  of a KV. This is mostly because I was curious to check if my theory
holds.

```
goos: darwin
goarch: amd64
pkg: kv
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkLRUCachePut-12                             	 9792549	       116.3 ns/op
BenchmarkLRUCacheGet-12                             	11791741	       102.4 ns/op
BenchmarkLRUCacheUpdate-12                          	 9982662	       116.3 ns/op
BenchmarkLRUCacheBatchUpdate-12                     	   16750	     70477 ns/op
BenchmarkMapCachePut-12                             	12798013	        96.57 ns/op
BenchmarkMapCacheGet-12                             	13420050	        88.41 ns/op
BenchmarkMapCacheUpdate-12                          	13724646	        83.62 ns/op
BenchmarkMapCacheBatchUpdateWithRollbackFalse-12    	   96966	     12173 ns/op
BenchmarkMapCacheBatchUpdateWithRollbackTrue-12     	   75212	     14902 ns/op
BenchmarkShardedSyncMapStore_Put-12                 	 1273052	       944.9 ns/op
BenchmarkShardedSyncMapStore_Get-12                 	11869588	       103.4 ns/op
BenchmarkShardedSyncMapStore_BatchUpdate-12         	   47833	     24083 ns/op
```

What I noticed seems to ascertain a lot of my assumptions. 

- The Map with Mutex solution seems to be the fastest. The downside here is obviously that we cant use an LRU based system.
   This might be okay for usecases where data is not ephermeral.

- The LRU implementation has an upside for more cache based use cases where the least recently used values get dropped.

- SyncMap seems to perform poorly on updates (even with my lazy sharding approach, but this could be due to multiple reasons
  like poor distribution amongst other things.


## Running this 

```
docker build -t kv-app . 
docker run -it -p 11201:11201 -p 11200:11200 --rm kv-app
```

## Testing

### Basic API functionality testing: 

Run the docker commands above first 
Also run
```
pip install -r requirements.txt
```

```
pytest test_kvstore.py
```

### Unit tests

```
go test -v -race
```

### Benchmarks

```
go test -bench=.
```

## Endpoints

### Set a Key-Value Pair

- **URL:** `/set`
- **Method:** `POST`
- **Body:**
  ```json
  {
    "key": "exampleKey",
    "value": "exampleValue"
  }
  ```
- **Success Response:**
  - **Code:** `201 Created`
- **Error Response:**
  - **Code:** `500 Internal Server Error`
  - **Description:** Server-side error

---

### Get a Value by Key

- **URL:** `/get?key=<key>`
- **Method:** `GET`
- **URL Parameters:**
  - `key` (required): The key to retrieve the value for.
- **Success Response:**
  - **Code:** `200 OK`
  - **Content:** `{ "value": "exampleValue" }`
- **Error Response:**
  - **Code:** `404 Not Found`
  - **Description:** Key not found
  - **Code:** `500 Internal Server Error`
  - **Description:** Server-side error

---

### Update a Key-Value Pair

- **URL:** `/updateBulk`
- **Method:** `PATCH`
- **Body:**
  ```json
  [
    {"key": "exampleKey1", "value": "newValue1"},
    {"key": "exampleKey2", "value": "newValue2"}
  ]
  ```
- **Success Response:**
  - **Code:** `200 OK`
  - **Content:** List of updated keys
- **Partial Update Response:**
  - **Code:** `206 Partial Content`
  - **Description:** Partial update, some keys not updated
- **Error Response:**
  - **Code:** `500 Internal Server Error`
  - **Description:** Server-side error

---

### Delete a Key

- **URL:** `/delete?key=<key>`
- **Method:** `DELETE`
- **URL Parameters:**
  - `key` (required): The key to delete.
- **Success Response:**
  - **Code:** `200 OK`
- **Error Response:**
  - **Code:** `500 Internal Server Error`
  - **Description:** Server-side error



