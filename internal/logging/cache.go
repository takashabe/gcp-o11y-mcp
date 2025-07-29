package logging

import (
	"crypto/md5"
	"fmt"
	"sync"
	"time"
)

type CacheEntry struct {
	Data      []LogEntry
	Timestamp time.Time
	TTL       time.Duration
}

type LogCache struct {
	mu    sync.RWMutex
	cache map[string]*CacheEntry
}

func NewLogCache() *LogCache {
	cache := &LogCache{
		cache: make(map[string]*CacheEntry),
	}

	// Start cleanup routine
	go cache.cleanup()

	return cache
}

func (c *LogCache) Get(key string) ([]LogEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.cache[key]
	if !exists {
		return nil, false
	}

	if time.Since(entry.Timestamp) > entry.TTL {
		return nil, false
	}

	return entry.Data, true
}

func (c *LogCache) Set(key string, data []LogEntry, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[key] = &CacheEntry{
		Data:      data,
		Timestamp: time.Now(),
		TTL:       ttl,
	}
}

func (c *LogCache) GenerateKey(params interface{}) string {
	hash := md5.Sum([]byte(fmt.Sprintf("%+v", params)))
	return fmt.Sprintf("%x", hash)
}

func (c *LogCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, entry := range c.cache {
			if now.Sub(entry.Timestamp) > entry.TTL {
				delete(c.cache, key)
			}
		}
		c.mu.Unlock()
	}
}
