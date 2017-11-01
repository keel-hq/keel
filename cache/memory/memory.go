package memory

import (
	"fmt"
	"strings"
	"time"

	"github.com/keel-hq/keel/cache"
)

type requestType int

// Request types
const (
	GET requestType = iota
	SET
	DELETE
	EXPIRE
	COPY
)

type (
	// Value - value is stored together with access and creation time
	Value struct {
		ctime time.Time
		atime time.Time
		value []byte
	}

	// Cache - cache container with a map of values and defaults
	Cache struct {
		cache          map[string]*Value
		ctimeExpiry    time.Duration // creation time
		atimeExpiry    time.Duration // access time
		expiryTick     time.Duration
		requestChannel chan *request
	}

	request struct {
		requestType
		key             string
		value           []byte
		responseChannel chan *response
	}
	response struct {
		error
		existingValue []byte
		mapCopy       map[string][]byte
		value         []byte
	}
)

func (c *Cache) isOld(v *Value) bool {
	if (c.ctimeExpiry != time.Duration(0)) && (time.Now().Sub(v.ctime) > c.ctimeExpiry) {
		return true
	}

	if (c.atimeExpiry != time.Duration(0)) && (time.Now().Sub(v.atime) > c.atimeExpiry) {
		return true
	}

	return false
}

// NewMemoryCache - creates new cache
func NewMemoryCache(ctimeExpiry, atimeExpiry, expiryTick time.Duration) *Cache {
	c := &Cache{
		cache:          make(map[string]*Value),
		ctimeExpiry:    ctimeExpiry,
		atimeExpiry:    atimeExpiry,
		expiryTick:     expiryTick,
		requestChannel: make(chan *request),
	}
	go c.service()
	if ctimeExpiry != time.Duration(0) || atimeExpiry != time.Duration(0) {
		go c.expiryService()
	}
	return c
}

func (c *Cache) service() {
	for {
		req := <-c.requestChannel
		resp := &response{}
		switch req.requestType {
		case GET:
			val, ok := c.cache[req.key]
			if !ok {
				resp.error = cache.ErrNotFound
			} else if c.isOld(val) {
				resp.error = cache.ErrExpired
				delete(c.cache, req.key)
			} else {
				// update atime
				val.atime = time.Now()
				c.cache[req.key] = val
				resp.value = val.value
			}
			req.responseChannel <- resp
		case SET:
			c.cache[req.key] = &Value{
				value: req.value,
				ctime: time.Now(),
				atime: time.Now(),
			}
			req.responseChannel <- resp
		case DELETE:
			delete(c.cache, req.key)
			req.responseChannel <- resp
		case EXPIRE:
			for k, v := range c.cache {
				if c.isOld(v) {
					delete(c.cache, k)
				}
			}
			// no response
		case COPY:
			resp.mapCopy = make(map[string][]byte)
			for k, v := range c.cache {
				resp.mapCopy[k] = v.value
			}
			req.responseChannel <- resp
		default:
			resp.error = fmt.Errorf("invalid request type: %v", req.requestType)
			req.responseChannel <- resp
		}
	}
}

// Get - looks up value and returns it
func (c *Cache) Get(key string) ([]byte, error) {
	respChannel := make(chan *response)
	c.requestChannel <- &request{
		requestType:     GET,
		key:             key,
		responseChannel: respChannel,
	}
	resp := <-respChannel
	return resp.value, resp.error
}

// Put - sets key/string. Overwrites existing key
func (c *Cache) Put(key string, value []byte) error {
	respChannel := make(chan *response)
	c.requestChannel <- &request{
		requestType:     SET,
		key:             key,
		value:           value,
		responseChannel: respChannel,
	}
	resp := <-respChannel
	return resp.error
}

// Delete - deletes key
func (c *Cache) Delete(key string) error {
	respChannel := make(chan *response)
	c.requestChannel <- &request{
		requestType:     DELETE,
		key:             key,
		responseChannel: respChannel,
	}
	resp := <-respChannel
	return resp.error
}

// List all values for specified prefix
func (c *Cache) List(prefix string) (map[string][]byte, error) {
	respChannel := make(chan *response)
	c.requestChannel <- &request{
		requestType:     COPY,
		responseChannel: respChannel,
	}
	resp := <-respChannel

	list := make(map[string][]byte)

	for k, v := range resp.mapCopy {
		if strings.HasPrefix(k, prefix) {
			list[k] = v
		}
	}
	return list, nil
}

// Copy - makes a copy of inmemory map
func (c *Cache) Copy() map[string][]byte {
	respChannel := make(chan *response)
	c.requestChannel <- &request{
		requestType:     COPY,
		responseChannel: respChannel,
	}
	resp := <-respChannel
	return resp.mapCopy
}

func (c *Cache) expiryService() {
	ticker := time.NewTicker(c.expiryTick)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.requestChannel <- &request{
				requestType: EXPIRE,
			}
		}
	}
}
