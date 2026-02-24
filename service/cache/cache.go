package cache

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Entry struct {
	Body    []byte
	Headers http.Header
	Expiry  time.Time
}

type Cache struct {
	mu      sync.RWMutex
	entries map[string]*Entry
}

func New() *Cache {
	return &Cache{entries: make(map[string]*Entry)}
}

func (c *Cache) Get(url string) (*Entry, bool) {
	c.mu.RLock()
	entry, ok := c.entries[url]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.Expiry) {
		c.mu.Lock()
		delete(c.entries, url)
		c.mu.Unlock()
		return nil, false
	}
	return entry, true
}

func (c *Cache) Set(url string, entry *Entry) {
	c.mu.Lock()
	c.entries[url] = entry
	c.mu.Unlock()
}

func (c *Cache) Delete(url string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.entries[url]; !ok {
		return false
	}
	delete(c.entries, url)
	return true
}

// ParseCacheControl returns (max-age seconds, shouldStore).
func ParseCacheControl(header string) (int, bool) {
	if header == "" {
		return 0, false
	}
	directives := strings.Split(header, ",")
	for _, d := range directives {
		d = strings.TrimSpace(strings.ToLower(d))
		if d == "no-store" {
			return 0, false
		}
	}
	for _, d := range directives {
		d = strings.TrimSpace(strings.ToLower(d))
		if strings.HasPrefix(d, "max-age=") {
			val := strings.TrimPrefix(d, "max-age=")
			seconds, err := strconv.Atoi(val)
			if err == nil && seconds > 0 {
				return seconds, true
			}
		}
	}
	return 0, false
}
