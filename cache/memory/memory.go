package memory

import (
	"strings"
	"sync"

	"github.com/keel-hq/keel/cache"
)

type Cache struct {
	entries map[string][]byte
	mu      *sync.RWMutex
}

func NewMemoryCache() *Cache {
	return &Cache{
		entries: make(map[string][]byte),
		mu:      &sync.RWMutex{},
	}

}

func (c *Cache) Put(key string, value []byte) error {
	c.mu.Lock()
	c.entries[key] = value
	c.mu.Unlock()

	return nil
}
func (c *Cache) Get(key string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// result := make([]byte, len(k))

	res, ok := c.entries[key]
	if !ok {
		return nil, cache.ErrNotFound
	}

	dst := make([]byte, len(res))
	copy(dst, res)

	return dst, nil
}
func (c *Cache) Delete(key string) error {
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()
	return nil
}
func (c *Cache) List(prefix string) (map[string][]byte, error) {
	c.mu.RLock()
	values := make(map[string][]byte)

	for k, v := range c.entries {
		if strings.HasPrefix(k, prefix) {
			dst := make([]byte, len(v))
			copy(dst, v)
			values[k] = dst
		}
	}

	c.mu.RUnlock()
	return values, nil
}
