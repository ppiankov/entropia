package cache

import (
	"fmt"
	"os"
	"time"
)

// LayeredCache implements a multi-layer cache (memory + disk)
type LayeredCache struct {
	memory Cache
	disk   Cache
}

// NewLayeredCache creates a new layered cache
func NewLayeredCache(memoryTTL time.Duration, diskDir string, diskTTL time.Duration) *LayeredCache {
	return &LayeredCache{
		memory: NewMemoryCache(memoryTTL, 10*time.Minute),
		disk:   NewDiskCache(diskDir, diskTTL),
	}
}

// Get retrieves a value from the cache (checks memory first, then disk)
func (c *LayeredCache) Get(key string) ([]byte, bool) {
	// Check memory cache first
	if val, found := c.memory.Get(key); found {
		return val, true
	}

	// Check disk cache
	if val, found := c.disk.Get(key); found {
		// Promote to memory cache (ignore error - memory cache Set never fails)
		_ = c.memory.Set(key, val, 0) // Use default TTL
		return val, true
	}

	return nil, false
}

// Set stores a value in both caches
func (c *LayeredCache) Set(key string, value []byte, ttl time.Duration) error {
	// Store in memory
	if err := c.memory.Set(key, value, ttl); err != nil {
		return err
	}

	// Store in disk
	if err := c.disk.Set(key, value, ttl); err != nil {
		return err
	}

	return nil
}

// Delete removes a value from both caches
func (c *LayeredCache) Delete(key string) error {
	// Memory delete never fails, but disk delete might
	_ = c.memory.Delete(key)
	if err := c.disk.Delete(key); err != nil && !os.IsNotExist(err) {
		// Ignore "not exists" errors - key might only be in memory
		return fmt.Errorf("delete from disk cache: %w", err)
	}
	return nil
}

// Clear removes all values from both caches
func (c *LayeredCache) Clear() error {
	// Memory clear never fails, but disk clear might
	_ = c.memory.Clear()
	if err := c.disk.Clear(); err != nil && !os.IsNotExist(err) {
		// Ignore "not exists" errors - cache dir might not exist
		return fmt.Errorf("clear disk cache: %w", err)
	}
	return nil
}
