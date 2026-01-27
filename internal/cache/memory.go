package cache

import (
	"time"

	gocache "github.com/patrickmn/go-cache"
)

// MemoryCache implements in-memory LRU caching
type MemoryCache struct {
	cache *gocache.Cache
}

// NewMemoryCache creates a new memory cache
func NewMemoryCache(defaultTTL time.Duration, cleanupInterval time.Duration) *MemoryCache {
	return &MemoryCache{
		cache: gocache.New(defaultTTL, cleanupInterval),
	}
}

// Get retrieves a value from the cache
func (c *MemoryCache) Get(key string) ([]byte, bool) {
	if val, found := c.cache.Get(key); found {
		return val.([]byte), true
	}
	return nil, false
}

// Set stores a value in the cache with the given TTL
func (c *MemoryCache) Set(key string, value []byte, ttl time.Duration) error {
	c.cache.Set(key, value, ttl)
	return nil
}

// Delete removes a value from the cache
func (c *MemoryCache) Delete(key string) error {
	c.cache.Delete(key)
	return nil
}

// Clear removes all values from the cache
func (c *MemoryCache) Clear() error {
	c.cache.Flush()
	return nil
}
