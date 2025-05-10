package cache

import (
	"sync"
	"time"
)

// Item represents a cache item with value and metadata
type Item struct {
	Value      interface{}
	Expiration int64
}

// RedisLikeCache provides a simple in-memory cache with Redis-like functionality
type RedisLikeCache struct {
	items map[string]Item
	mu    sync.RWMutex
}

// NewRedisLikeCache creates a new Redis-like cache
func NewRedisLikeCache() *RedisLikeCache {
	cache := &RedisLikeCache{
		items: make(map[string]Item),
	}
	
	// Start a goroutine to clean expired items periodically
	go cache.cleanupRoutine()
	
	return cache
}

// Set adds a key-value pair to the cache
func (c *RedisLikeCache) Set(key string, value interface{}, expiration time.Duration) {
	var exp int64
	
	if expiration > 0 {
		exp = time.Now().Add(expiration).UnixNano()
	}
	
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.items[key] = Item{
		Value:      value,
		Expiration: exp,
	}
}

// Get retrieves a value from the cache
func (c *RedisLikeCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	item, found := c.items[key]
	if !found {
		return nil, false
	}
	
	// Check if the item has expired
	if item.Expiration > 0 && time.Now().UnixNano() > item.Expiration {
		return nil, false
	}
	
	return item.Value, true
}

// Delete removes a key-value pair from the cache
func (c *RedisLikeCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	delete(c.items, key)
}

// Dump returns all key-value pairs in the cache (for debugging)
func (c *RedisLikeCache) Dump() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	dump := make(map[string]interface{})
	
	for k, item := range c.items {
		// Skip expired items
		if item.Expiration > 0 && time.Now().UnixNano() > item.Expiration {
			continue
		}
		dump[k] = item.Value
	}
	
	return dump
}

// cleanupRoutine periodically removes expired items from the cache
func (c *RedisLikeCache) cleanupRoutine() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	
	for {
		<-ticker.C
		c.mu.Lock()
		now := time.Now().UnixNano()
		
		for k, item := range c.items {
			if item.Expiration > 0 && now > item.Expiration {
				delete(c.items, k)
			}
		}
		
		c.mu.Unlock()
	}
}
