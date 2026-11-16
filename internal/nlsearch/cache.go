package nlsearch

import (
	"strings"
	"sync"
	"time"
	"unicode"
)

// The cache holds interpretations, not results. The rows are read from the
// database on every request; only the reading of the question is kept.

// planCacheTTL is how long a reading is kept. It is short because the
// vocabulary behind it can change.
const planCacheTTL = 5 * time.Minute

// planCacheMax is how many readings are kept at once.
const planCacheMax = 512

// planCache remembers how a question was read.
type planCache struct {
	mu      sync.Mutex
	entries map[string]cacheEntry
}

type cacheEntry struct {
	plans   *Plans
	expires time.Time
}

// newPlanCache builds a cache.
func newPlanCache() *planCache {
	return &planCache{entries: make(map[string]cacheEntry)}
}

// Get returns the reading of a question, if one is still fresh.
func (c *planCache) Get(key string) (*Plans, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	if !time.Now().Before(entry.expires) {
		delete(c.entries, key)
		return nil, false
	}
	return entry.plans, true
}

// Put remembers how a question was read.
func (c *planCache) Put(key string, plans *Plans) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.entries) >= planCacheMax {
		c.evict()
	}
	c.entries[key] = cacheEntry{plans: plans, expires: time.Now().Add(planCacheTTL)}
}

// evict makes room. Readings that have already expired go first; if that is
// not enough, the one closest to expiring goes.
func (c *planCache) evict() {
	now := time.Now()
	for key, entry := range c.entries {
		if !now.Before(entry.expires) {
			delete(c.entries, key)
		}
	}
	if len(c.entries) < planCacheMax {
		return
	}

	oldestKey := ""
	var oldest time.Time
	for key, entry := range c.entries {
		if oldestKey == "" || entry.expires.Before(oldest) {
			oldestKey, oldest = key, entry.expires
		}
	}
	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

// normaliseQuery reduces a question to a form two questions share when they
func normaliseQuery(query string) string {
	var b strings.Builder
	b.Grow(len(query))

	space := false
	for _, r := range strings.ToLower(query) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			space = false
			continue
		}
		if !space {
			b.WriteByte(' ')
			space = true
		}
	}
	return strings.TrimSpace(b.String())
}
