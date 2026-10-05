package middleware

import (
	"encoding/json"
	"sync"
	"time"
)

// CacheEntry holds a cached value with expiration.
type CacheEntry struct {
	Value     []byte    `json:"value"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Cache provides a simple in-memory TTL cache for read-only API responses.
// Keys are typically "account:resource:query_fingerprint".
type Cache struct {
	mu       sync.RWMutex
	entries  map[string]CacheEntry
	stopCh   chan struct{}
	doneCh   chan struct{}
	interval time.Duration
	stopOnce sync.Once
}

// NewCache creates a cache with a cleanup interval.
func NewCache(cleanupInterval time.Duration) *Cache {
	c := &Cache{
		entries:  make(map[string]CacheEntry),
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
		interval: cleanupInterval,
	}
	go c.cleanup()
	return c
}

// Get retrieves a cached value. Returns nil, false if not found or expired.
func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.ExpiresAt) {
		return nil, false
	}
	// Return a copy to prevent mutation.
	out := make([]byte, len(entry.Value))
	copy(out, entry.Value)
	return out, true
}

// Set stores a value with TTL.
func (c *Cache) Set(key string, value []byte, ttl time.Duration) {
	if ttl <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	// Copy value to avoid external mutation.
	v := make([]byte, len(value))
	copy(v, value)
	c.entries[key] = CacheEntry{
		Value:     v,
		ExpiresAt: time.Now().Add(ttl),
	}
}

// SetJSON marshals v to JSON and caches it.
func (c *Cache) SetJSON(key string, v interface{}, ttl time.Duration) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	c.Set(key, data, ttl)
	return nil
}

// Delete removes a key.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

// DeletePrefix removes all keys starting with prefix.
func (c *Cache) DeletePrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.entries {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			delete(c.entries, k)
		}
	}
}

// InvalidateAll clears the entire cache.
func (c *Cache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]CacheEntry)
}

// Size returns the number of cached entries.
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// Stop stops the cleanup goroutine. Safe to call multiple times.
func (c *Cache) Stop() {
	c.stopOnce.Do(func() {
		close(c.stopCh)
		<-c.doneCh
	})
}

func (c *Cache) cleanup() {
	defer close(c.doneCh)
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.purgeExpired()
		}
	}
}

func (c *Cache) purgeExpired() {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, v := range c.entries {
		if now.After(v.ExpiresAt) {
			delete(c.entries, k)
		}
	}
}
