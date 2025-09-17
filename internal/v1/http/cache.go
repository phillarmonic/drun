package http

import (
	"sync"
	"time"
)

// MemoryCache implements an in-memory cache for HTTP responses
type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]*cacheItem
}

type cacheItem struct {
	response  *Response
	expiresAt time.Time
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache() *MemoryCache {
	cache := &MemoryCache{
		items: make(map[string]*cacheItem),
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves a cached response
func (c *MemoryCache) Get(key string) (*Response, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(item.expiresAt) {
		// Item has expired, remove it
		delete(c.items, key)
		return nil, false
	}

	return item.response, true
}

// Set stores a response in the cache
func (c *MemoryCache) Set(key string, response *Response, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &cacheItem{
		response:  response,
		expiresAt: time.Now().Add(ttl),
	}
}

// Delete removes a cached response
func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// Clear removes all cached responses
func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*cacheItem)
}

// Size returns the number of cached items
func (c *MemoryCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}

// cleanup removes expired items periodically
func (c *MemoryCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.expiresAt) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}

// NoCache is a cache implementation that doesn't cache anything
type NoCache struct{}

// Get always returns false (no cache hit)
func (n *NoCache) Get(key string) (*Response, bool) {
	return nil, false
}

// Set does nothing
func (n *NoCache) Set(key string, response *Response, ttl time.Duration) {
	// Do nothing
}

// Delete does nothing
func (n *NoCache) Delete(key string) {
	// Do nothing
}

// LRUCache implements a Least Recently Used cache
type LRUCache struct {
	mu       sync.RWMutex
	capacity int
	items    map[string]*lruItem
	head     *lruItem
	tail     *lruItem
}

type lruItem struct {
	key       string
	response  *Response
	expiresAt time.Time
	prev      *lruItem
	next      *lruItem
}

// NewLRUCache creates a new LRU cache with the specified capacity
func NewLRUCache(capacity int) *LRUCache {
	cache := &LRUCache{
		capacity: capacity,
		items:    make(map[string]*lruItem),
	}

	// Create dummy head and tail nodes
	cache.head = &lruItem{}
	cache.tail = &lruItem{}
	cache.head.next = cache.tail
	cache.tail.prev = cache.head

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves a cached response and moves it to the front
func (c *LRUCache) Get(key string) (*Response, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(item.expiresAt) {
		// Item has expired, remove it
		c.removeItem(item)
		return nil, false
	}

	// Move to front (most recently used)
	c.moveToFront(item)

	return item.response, true
}

// Set stores a response in the cache
func (c *LRUCache) Set(key string, response *Response, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, exists := c.items[key]; exists {
		// Update existing item
		item.response = response
		item.expiresAt = time.Now().Add(ttl)
		c.moveToFront(item)
		return
	}

	// Create new item
	item := &lruItem{
		key:       key,
		response:  response,
		expiresAt: time.Now().Add(ttl),
	}

	c.items[key] = item
	c.addToFront(item)

	// Remove least recently used item if capacity exceeded
	if len(c.items) > c.capacity {
		c.removeLRU()
	}
}

// Delete removes a cached response
func (c *LRUCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, exists := c.items[key]; exists {
		c.removeItem(item)
	}
}

// addToFront adds an item to the front of the list
func (c *LRUCache) addToFront(item *lruItem) {
	item.prev = c.head
	item.next = c.head.next
	c.head.next.prev = item
	c.head.next = item
}

// removeItem removes an item from the list and map
func (c *LRUCache) removeItem(item *lruItem) {
	item.prev.next = item.next
	item.next.prev = item.prev
	delete(c.items, item.key)
}

// moveToFront moves an item to the front of the list
func (c *LRUCache) moveToFront(item *lruItem) {
	c.removeFromList(item)
	c.addToFront(item)
}

// removeFromList removes an item from the list but not from the map
func (c *LRUCache) removeFromList(item *lruItem) {
	item.prev.next = item.next
	item.next.prev = item.prev
}

// removeLRU removes the least recently used item
func (c *LRUCache) removeLRU() {
	lru := c.tail.prev
	if lru != c.head {
		c.removeItem(lru)
	}
}

// cleanup removes expired items periodically
func (c *LRUCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for _, item := range c.items {
			if now.After(item.expiresAt) {
				c.removeItem(item)
			}
		}
		c.mu.Unlock()
	}
}
