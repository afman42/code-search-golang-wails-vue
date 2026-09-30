package main

import (
	"container/list"
	"regexp"
	"strings"
	"sync"
)

// NewLRUPatternCache creates a new LRU cache with the specified max size
func NewLRUPatternCache(maxSize int64) *LRUPatternCache {
	// A maxSize <= 0 would silently disable eviction (Set only evicts while
	// maxSize > 0), making the cache unbounded. Clamp to a sane floor.
	if maxSize <= 0 {
		maxSize = 100
	}
	return &LRUPatternCache{
		cache:   make(map[string]*list.Element),
		list:    list.New(),
		maxSize: maxSize,
	}
}

// Get retrieves a value from the cache, moving it to the front if found
func (c *LRUPatternCache) Get(key string) (*regexp.Regexp, bool) {
	// Fast path under RLock: hits only need the read lock until the LRU
	// promotion. The old code took the full write Lock on every Get,
	// serializing concurrent searches on the cache mutex.
	c.mu.RLock()
	_, ok := c.cache[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	// Re-check under the write lock: an eviction may have dropped the key
	// between the RLock release and the Lock acquisition.
	if elem, ok := c.cache[key]; ok {
		c.list.MoveToFront(elem)
		return elem.Value.(*lruEntry).value, true
	}
	return nil, false
}

// Set adds or updates a value in the cache, evicting old entries if necessary.
// Eviction is O(1) per entry: the key is stored inside the list element, so no
// map scan is needed to find which key the back element belongs to.
func (c *LRUPatternCache) Set(key string, value *regexp.Regexp) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if key exists and move to front
	if elem, ok := c.cache[key]; ok {
		c.list.MoveToFront(elem)
		elem.Value.(*lruEntry).value = value
		return
	}

	// Add new entry with the key stored alongside the value
	elem := c.list.PushFront(&lruEntry{key: key, value: value})
	c.cache[key] = elem

	// Evict the back (least-recently-used) element while over capacity
	for c.maxSize > 0 && int64(len(c.cache)) > c.maxSize {
		backElem := c.list.Back()
		if backElem == nil {
			break
		}
		c.list.Remove(backElem)
		delete(c.cache, backElem.Value.(*lruEntry).key)
	}
}

func getPatternCacheKey(useRegex bool, caseSensitive bool, query string) string {
	// No Sprintf: one small string build with exact capacity instead of
	// fmt's reflection/verb parsing per search.
	var sb strings.Builder
	sb.Grow(len(query) + 8)
	if useRegex {
		sb.WriteByte('1')
	} else {
		sb.WriteByte('0')
	}
	sb.WriteByte(':')
	if caseSensitive {
		sb.WriteString("true:")
	} else {
		sb.WriteString("false:")
	}
	sb.WriteString(query)
	return sb.String()
}

// lruEntry pairs a cache key with its compiled regex so eviction can remove
// the backing map entry in O(1) instead of scanning the whole map for the key
// that maps to the back element of the LRU list.
type lruEntry struct {
	key   string
	value *regexp.Regexp
}

// LRUPatternCache is a thread-safe LRU cache for compiled regex patterns.
type LRUPatternCache struct {
	mu      sync.RWMutex
	cache   map[string]*list.Element
	list    *list.List
	maxSize int64
}
