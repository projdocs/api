package versions

import (
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/projdocs/projdocs/packages/go/database"
)

type fileVersionEntry struct {
	FileVersion     *database.PublicFilesVersionsSelect
	StorageUpload   *database.PublicStorageUploadsSelect
	StorageProvider *database.PublicStorageProvidersSelect
	expiresAt       time.Time
}

type Cache struct {
	mu      sync.RWMutex
	entries map[string]*fileVersionEntry
	ttl     time.Duration
}

var cache = newFileVersionCache(time.Hour)

func newFileVersionCache(ttl time.Duration) *Cache {
	c := &Cache{
		entries: make(map[string]*fileVersionEntry),
		ttl:     ttl,
	}
	go c.cleanupLoop()
	return c
}

func (c *Cache) Set(fv *database.PublicFilesVersionsSelect, su *database.PublicStorageUploadsSelect, sp *database.PublicStorageProvidersSelect) string {

	c.mu.Lock()
	defer c.mu.Unlock()

	cacheKey := strings.ReplaceAll(uuid.New().String(), "-", "")

	c.entries[cacheKey] = &fileVersionEntry{
		FileVersion:     fv,
		StorageUpload:   su,
		StorageProvider: sp,
		expiresAt:       time.Now().Add(c.ttl),
	}

	return cacheKey
}

func (c *Cache) Get(key string) (*database.PublicFilesVersionsSelect, *database.PublicStorageUploadsSelect, *database.PublicStorageProvidersSelect, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, nil, nil, false
	}
	return entry.FileVersion, entry.StorageUpload, entry.StorageProvider, true
}

func (c *Cache) cleanupLoop() {
	ticker := time.NewTicker(c.ttl)
	defer ticker.Stop()
	for range ticker.C {
		c.evict()
	}
}

func (c *Cache) evict() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for k, v := range c.entries {
		if now.After(v.expiresAt) {
			delete(c.entries, k)
		}
	}
}
