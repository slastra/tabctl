package client

import (
	"sync"
	"time"
)

// cacheEntry represents a cached item with expiry
type cacheEntry struct {
	value     interface{}
	expiresAt time.Time
}

// ResponseCache provides simple in-memory caching
type ResponseCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
}

// NewResponseCache creates a new response cache
func NewResponseCache(cleanupInterval time.Duration) *ResponseCache {
	cache := &ResponseCache{
		entries: make(map[string]*cacheEntry),
	}

	// Start cleanup goroutine
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()

		for range ticker.C {
			cache.cleanup()
		}
	}()

	return cache
}

// Get retrieves a value from cache
func (c *ResponseCache) Get(key string) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return nil
	}

	if time.Now().After(entry.expiresAt) {
		return nil
	}

	return entry.value
}

// Set adds a value to cache with TTL
func (c *ResponseCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = &cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
}

// cleanup removes expired entries
func (c *ResponseCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.expiresAt) {
			delete(c.entries, key)
		}
	}
}