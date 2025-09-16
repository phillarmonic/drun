package http

import (
	"net/http"
	"testing"
	"time"
)

func TestMemoryCache(t *testing.T) {
	cache := NewMemoryCache()

	// Create a mock response
	resp := &Response{
		Response: &http.Response{StatusCode: 200},
		body:     []byte("test response"),
	}

	// Test Set and Get
	cache.Set("key1", resp, time.Minute)

	retrieved, found := cache.Get("key1")
	if !found {
		t.Error("Expected to find cached item")
	}

	if string(retrieved.body) != "test response" {
		t.Errorf("Expected 'test response', got '%s'", string(retrieved.body))
	}

	// Test non-existent key
	_, found = cache.Get("nonexistent")
	if found {
		t.Error("Expected not to find non-existent key")
	}

	// Test expiration
	cache.Set("expiring", resp, time.Millisecond)
	time.Sleep(2 * time.Millisecond)

	_, found = cache.Get("expiring")
	if found {
		t.Error("Expected expired item to not be found")
	}

	// Test Delete
	cache.Set("deleteme", resp, time.Minute)
	cache.Delete("deleteme")

	_, found = cache.Get("deleteme")
	if found {
		t.Error("Expected deleted item to not be found")
	}

	// Test Size
	cache.Clear()
	if cache.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", cache.Size())
	}

	cache.Set("item1", resp, time.Minute)
	cache.Set("item2", resp, time.Minute)

	if cache.Size() != 2 {
		t.Errorf("Expected size 2, got %d", cache.Size())
	}
}

func TestNoCache(t *testing.T) {
	cache := &NoCache{}

	resp := &Response{
		Response: &http.Response{StatusCode: 200},
		body:     []byte("test response"),
	}

	// Set should do nothing
	cache.Set("key1", resp, time.Minute)

	// Get should always return false
	_, found := cache.Get("key1")
	if found {
		t.Error("NoCache should never return cached items")
	}

	// Delete should do nothing (no panic)
	cache.Delete("key1")
}

func TestLRUCache(t *testing.T) {
	cache := NewLRUCache(2) // Capacity of 2

	resp1 := &Response{
		Response: &http.Response{StatusCode: 200},
		body:     []byte("response 1"),
	}

	resp2 := &Response{
		Response: &http.Response{StatusCode: 200},
		body:     []byte("response 2"),
	}

	resp3 := &Response{
		Response: &http.Response{StatusCode: 200},
		body:     []byte("response 3"),
	}

	// Add two items
	cache.Set("key1", resp1, time.Minute)
	cache.Set("key2", resp2, time.Minute)

	// Both should be retrievable
	retrieved, found := cache.Get("key1")
	if !found || string(retrieved.body) != "response 1" {
		t.Error("Expected to find key1")
	}

	retrieved, found = cache.Get("key2")
	if !found || string(retrieved.body) != "response 2" {
		t.Error("Expected to find key2")
	}

	// Add third item, should evict least recently used (key1)
	cache.Set("key3", resp3, time.Minute)

	// key1 should be evicted
	_, found = cache.Get("key1")
	if found {
		t.Error("Expected key1 to be evicted")
	}

	// key2 and key3 should still be there
	_, found = cache.Get("key2")
	if !found {
		t.Error("Expected key2 to still be cached")
	}

	_, found = cache.Get("key3")
	if !found {
		t.Error("Expected key3 to be cached")
	}

	// Access key2 to make it most recently used
	cache.Get("key2")

	// Add another item, should evict key3 (least recently used)
	cache.Set("key4", resp1, time.Minute)

	_, found = cache.Get("key3")
	if found {
		t.Error("Expected key3 to be evicted")
	}

	_, found = cache.Get("key2")
	if !found {
		t.Error("Expected key2 to still be cached")
	}

	// Test expiration
	cache.Set("expiring", resp1, time.Millisecond)
	time.Sleep(2 * time.Millisecond)

	_, found = cache.Get("expiring")
	if found {
		t.Error("Expected expired item to not be found")
	}

	// Test Delete
	cache.Set("deleteme", resp1, time.Minute)
	cache.Delete("deleteme")

	_, found = cache.Get("deleteme")
	if found {
		t.Error("Expected deleted item to not be found")
	}
}

func TestLRUCacheUpdateExisting(t *testing.T) {
	cache := NewLRUCache(2)

	resp1 := &Response{
		Response: &http.Response{StatusCode: 200},
		body:     []byte("response 1"),
	}

	resp1Updated := &Response{
		Response: &http.Response{StatusCode: 200},
		body:     []byte("response 1 updated"),
	}

	// Set initial value
	cache.Set("key1", resp1, time.Minute)

	// Update existing key
	cache.Set("key1", resp1Updated, time.Minute)

	// Should get updated value
	retrieved, found := cache.Get("key1")
	if !found {
		t.Error("Expected to find key1")
	}

	if string(retrieved.body) != "response 1 updated" {
		t.Errorf("Expected 'response 1 updated', got '%s'", string(retrieved.body))
	}
}
