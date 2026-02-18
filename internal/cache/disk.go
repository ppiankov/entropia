package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DiskCache implements persistent disk-based caching
type DiskCache struct {
	dir string
	ttl time.Duration
}

// NewDiskCache creates a new disk cache
func NewDiskCache(dir string, ttl time.Duration) *DiskCache {
	return &DiskCache{
		dir: dir,
		ttl: ttl,
	}
}

type cacheEntry struct {
	Data      []byte    `json:"data"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Get retrieves a value from the disk cache
func (c *DiskCache) Get(key string) ([]byte, bool) {
	path := c.path(key)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, false
	}

	// Check expiration
	if time.Now().After(entry.ExpiresAt) {
		_ = os.Remove(path)
		return nil, false
	}

	return entry.Data, true
}

// Set stores a value in the disk cache
func (c *DiskCache) Set(key string, value []byte, ttl time.Duration) error {
	if ttl == 0 {
		ttl = c.ttl
	}

	entry := cacheEntry{
		Data:      value,
		ExpiresAt: time.Now().Add(ttl),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal entry: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(c.dir, 0755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}

	path := c.path(key)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write cache file: %w", err)
	}

	return nil
}

// Delete removes a value from the disk cache
func (c *DiskCache) Delete(key string) error {
	path := c.path(key)
	return os.Remove(path)
}

// Clear removes all cached files
func (c *DiskCache) Clear() error {
	return os.RemoveAll(c.dir)
}

// path generates the file path for a cache key
func (c *DiskCache) path(key string) string {
	return filepath.Join(c.dir, key+".cache")
}
