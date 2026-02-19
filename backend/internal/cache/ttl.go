package cache

import (
	"sync"
	"time"
)

// TTL is a simple in-memory cache with TTL. Keys are strings, values are []byte (e.g. JSON).
type TTL struct {
	mu    sync.RWMutex
	items map[string]item
	ttl   time.Duration
}

type item struct {
	data []byte
	exp  time.Time
}

// New returns a new TTL cache with the given duration. After duration, entries expire.
func New(ttl time.Duration) *TTL {
	c := &TTL{items: make(map[string]item), ttl: ttl}
	go c.cleanup()
	return c
}

func (c *TTL) cleanup() {
	tick := time.NewTicker(c.ttl / 2)
	defer tick.Stop()
	for range tick.C {
		c.mu.Lock()
		now := time.Now()
		for k, v := range c.items {
			if v.exp.Before(now) {
				delete(c.items, k)
			}
		}
		c.mu.Unlock()
	}
}

// Get returns the value for key if present and not expired. Otherwise nil.
func (c *TTL) Get(key string) []byte {
	c.mu.RLock()
	it, ok := c.items[key]
	c.mu.RUnlock()
	if !ok || it.exp.Before(time.Now()) {
		return nil
	}
	return it.data
}

// Set stores the value for key with the cache TTL.
func (c *TTL) Set(key string, value []byte) {
	exp := time.Now().Add(c.ttl)
	c.mu.Lock()
	c.items[key] = item{data: value, exp: exp}
	c.mu.Unlock()
}

// Delete removes the key.
func (c *TTL) Delete(key string) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}

// DeletePrefix removes all keys that start with prefix (e.g. "me:" to clear all me cache).
func (c *TTL) DeletePrefix(prefix string) {
	c.mu.Lock()
	for k := range c.items {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			delete(c.items, k)
		}
	}
	c.mu.Unlock()
}
