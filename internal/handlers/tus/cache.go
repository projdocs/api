package tus

import (
	"sync"
	"time"

	"github.com/projdocs/projdocs/packages/go/database"
)

type configCacheEntry struct {
	cfg       *database.PublicStorageProvidersSelect
	expiresAt time.Time
}

type configCache struct {
	mu      sync.RWMutex
	entries map[string]configCacheEntry
	ttl     time.Duration
}

func newConfigCache(ttl time.Duration) *configCache {
	c := &configCache{
		entries: make(map[string]configCacheEntry),
		ttl:     ttl,
	}
	go c.reap()
	return c
}

func (c *configCache) get(uploadID string) *database.PublicStorageProvidersSelect {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[uploadID]
	if !ok || time.Now().After(e.expiresAt) {
		return nil
	}
	return e.cfg
}

func (c *configCache) set(uploadID string, cfg *database.PublicStorageProvidersSelect) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[uploadID] = configCacheEntry{
		cfg:       cfg,
		expiresAt: time.Now().Add(c.ttl),
	}
}

func (c *configCache) delete(uploadID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, uploadID)
}

// reap removes expired entries every ttl/2 to prevent unbounded growth.
func (c *configCache) reap() {
	for range time.Tick(c.ttl / 2) {
		c.mu.Lock()
		for id, e := range c.entries {
			if time.Now().After(e.expiresAt) {
				delete(c.entries, id)
			}
		}
		c.mu.Unlock()
	}
}
