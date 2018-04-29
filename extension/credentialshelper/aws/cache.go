package aws

import (
	"fmt"
	"sync"
	"time"

	"github.com/keel-hq/keel/types"
)

type item struct {
	credentials *types.Credentials
	created     time.Time
}

// Cache - internal cache for aws
type Cache struct {
	creds map[string]*item
	tick  time.Duration
	ttl   time.Duration
	mu    *sync.RWMutex
}

// NewCache - new credentials cache
func NewCache(ttl time.Duration) *Cache {
	c := &Cache{
		creds: make(map[string]*item),
		ttl:   ttl,
		tick:  30 * time.Second,
		mu:    &sync.RWMutex{},
	}
	go c.expiryService()
	return c
}

func (c *Cache) expiryService() {
	ticker := time.NewTicker(c.tick)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.expire()
		}
	}
}

func (c *Cache) expire() {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := time.Now()
	for k, v := range c.creds {
		if t.Sub(v.created) > c.ttl {
			delete(c.creds, k)
		}
	}
}

// Put - saves new creds
func (c *Cache) Put(registry string, creds *types.Credentials) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.creds[registry] = &item{credentials: creds, created: time.Now()}
}

// Get - retrieves creds
func (c *Cache) Get(registry string) (*types.Credentials, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.creds[registry]
	if !ok {
		return nil, fmt.Errorf("not found")
	}

	cr := new(types.Credentials)
	*cr = *item.credentials

	return cr, nil
}
