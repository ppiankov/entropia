package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// Cache defines the interface for caching
type Cache interface {
	Get(key string) ([]byte, bool)
	Set(key string, value []byte, ttl time.Duration) error
	Delete(key string) error
	Clear() error
}

// CacheKey generates a cache key from a URL
func CacheKey(url string) string {
	hash := sha256.Sum256([]byte(url))
	return "entropia:v1:" + hex.EncodeToString(hash[:])
}
