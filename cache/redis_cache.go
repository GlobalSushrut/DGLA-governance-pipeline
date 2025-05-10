package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/umesh/dgla/config"
)

// RedisCache provides a Redis-backed implementation of the cache
type RedisCache struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisCache creates a new Redis cache
func NewRedisCache(cfg config.CacheConfig) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	ctx := context.Background()

	// Ping Redis to verify connection
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCache{
		client: client,
		ctx:    ctx,
	}, nil
}

// Set adds a key-value pair to the cache
func (c *RedisCache) Set(key string, value interface{}, expiration time.Duration) error {
	// Serialize the value to JSON
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to serialize value: %w", err)
	}

	// Store in Redis
	return c.client.Set(c.ctx, key, data, expiration).Err()
}

// Get retrieves a value from the cache
func (c *RedisCache) Get(key string) (interface{}, bool) {
	// Get from Redis
	data, err := c.client.Get(c.ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			// Key does not exist
			return nil, false
		}
		// Some other error occurred
		return nil, false
	}

	// For now, return the raw bytes
	// In a real implementation, you'd deserialize based on the type
	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, false
	}

	return result, true
}

// Delete removes a key-value pair from the cache
func (c *RedisCache) Delete(key string) error {
	return c.client.Del(c.ctx, key).Err()
}

// Dump returns all key-value pairs in the cache (for debugging)
func (c *RedisCache) Dump() map[string]interface{} {
	dump := make(map[string]interface{})

	// Get all keys
	keys, err := c.client.Keys(c.ctx, "*").Result()
	if err != nil {
		return dump
	}

	// Get values for each key
	for _, key := range keys {
		if value, found := c.Get(key); found {
			dump[key] = value
		}
	}

	return dump
}

// Close closes the Redis connection
func (c *RedisCache) Close() error {
	return c.client.Close()
}
