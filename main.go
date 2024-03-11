package main

import (
	"container/list"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// GOALS
// 1. Develop a LRU cache with Get/Set API
// 2. Build a react app that consumes the LRU cache api

// REQUIREMENTS

// Develop a LRU Cache
// ● The cache will store Key/Value with expiration.
// ● If the expiration for key is set to 5 seconds,then that key should be evicted from the cache after 5 seconds.
// ● The cache can store maximum of 1024 keys.

// Must Haves
// ● Backend should be built on Golang
// ● The Get/set method in cache should be exposed as api endpoints

// Good to have
// ● Implementing concurrency in cache

// Develop a React Application
// ● Develop a react application that will consume Get api to get the key from cache and set key/value in the cache

// Considerations
// LRU
// Size
// Concurrency
// Exipration
// API

type Cache struct {
	mu    sync.Mutex         // protects lru and cache
	lru   *list.List         // doubly linked list
	cache map[string]Element // map of string to Element
	size  int                // max size of cache
}

type Element struct {
	value *list.Element // pointer to list.Element
	timer *time.Timer   // pointer to time.Timer
}

func NewCache(size int) *Cache {
	return &Cache{
		lru:   list.New(),
		cache: make(map[string]Element),
		size:  size,
	}
}

func (c *Cache) Get(key string) (*list.Element, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if e, ok := c.cache[key]; ok {
		c.lru.MoveToFront(e.value)
		return e.value, true
	}

	// c.mu.Unlock()
	return nil, false
}

func (c *Cache) Set(key string, value *list.Element, expiration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if e, ok := c.cache[key]; ok {
		c.lru.MoveToFront(e.value) // move to front
		e.timer.Stop()             // stop timer
		e.timer = time.AfterFunc(expiration, func() {
			c.Delete(key)
		}) // reset timer
		return
	}

	if c.lru.Len() >= c.size { // if cache is full
		c.evict() // evict
	}

	c.lru.PushFront(key) // push to front
	c.cache[key] = Element{value: value, timer: time.AfterFunc(expiration, func() {
		c.Delete(key) // delete after expiration
	})} // add to cache
}

func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if e, ok := c.cache[key]; ok {
		c.lru.Remove(e.value) // remove from list
		e.timer.Stop()        // stop timer
		delete(c.cache, key)  // delete from cache
	}
}

func (c *Cache) evict() {
	e := c.lru.Back() // get last element
	if e != nil {
		c.lru.Remove(e)                   // remove from list
		delete(c.cache, e.Value.(string)) // delete from cache
	}
}

func ValueToElement(value string) *list.Element {
	return &list.Element{Value: value}
}

type Response struct {
	Message string `json:"message"`
}

func (r *Response) String() string {
	return fmt.Sprintf("{ \"message\": \"%s\" }", r.Message)
}

func main() {
	cache := NewCache(1024)

	server := http.NewServeMux()
	handler := corsMiddleware(server)
	server.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		key := r.URL.Query().Get("key")

		if key == "" {
			res := Response{
				Message: "key not found",
			}

			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(res.String()))
			return
		}

		if e, ok := cache.Get(key); ok {
			res := Response{
				Message: e.Value.(string),
			}

			w.Write([]byte(res.String()))
			return
		}
		w.WriteHeader(http.StatusNotFound)
		res := Response{
			Message: "key not found",
		}
		w.Write([]byte(res.String()))

	})

	server.HandleFunc("/set", func(w http.ResponseWriter, r *http.Request) {
		log.Println("set")
		w.Header().Set("Content-Type", "application/json")
		key := r.URL.Query().Get("key")
		value := r.URL.Query().Get("value")
		expirationInMilliSeconds, err := strconv.Atoi(r.URL.Query().Get("exp"))
		if err != nil {
			res := Response{
				Message: "exp must be a number",
			}
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(res.String()))
			return
		}

		expiration := time.Duration(expirationInMilliSeconds) * time.Millisecond

		if key == "" || value == "" {
			res := Response{
				Message: "key and value are required",
			}
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(res.String()))
			return
		}

		cache.Set(key, ValueToElement(value), expiration)
		res := Response{
			Message: "key set",
		}
		w.Write([]byte(res.String()))

	})

	// Enable CORS

	http.ListenAndServe(":8080", handler)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")                                                                            // Allow requests from any origin
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")                                             // Define the allowed methods
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Access-Control-Allow-Headers, Authorization, X-Requested-With") // Define the allowed headers

		// Handle preflight OPTIONS requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Call the next handler in the chain
		next.ServeHTTP(w, r)
	})
}
