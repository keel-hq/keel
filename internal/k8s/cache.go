package k8s

import (
	"sort"
	"sync"
)

type genericResourceCache struct {
	sync.Mutex
	values []*GenericResource
}

// GenericResourceCache - storage for generic resources with a rendezvous point for goroutines
// waiting for or announcing the occurence of a cache events.
type GenericResourceCache struct {
	genericResourceCache
	Cond
}

// Values returns a copy of the contents of the cache.
func (cc *genericResourceCache) Values() []*GenericResource {
	cc.Lock()
	r := []*GenericResource{}
	for _, v := range cc.values {
		r = append(r, v.DeepCopy())
	}
	cc.Unlock()
	return r
}

// Add adds an entry to the cache. If a GenericResource with the same
// name exists, it is replaced.
func (cc *genericResourceCache) Add(grs ...*GenericResource) {
	if len(grs) == 0 {
		return
	}
	cc.Lock()
	sort.Sort(genericResource(cc.values))
	for _, gr := range grs {
		cc.add(gr)
	}
	cc.Unlock()
}

// add adds c to the cache. If c is already present, the cached value of c is overwritten.
// invariant: cc.values should be sorted on entry.
func (cc *genericResourceCache) add(c *GenericResource) {
	i := sort.Search(len(cc.values), func(i int) bool { return cc.values[i].Identifier >= c.Identifier })
	if i < len(cc.values) && cc.values[i].Identifier == c.Identifier {
		// c is already present, replace
		cc.values[i] = c
	} else {
		// c is not present, append
		cc.values = append(cc.values, c)
		// restort to convert append into insert
		sort.Sort(genericResource(cc.values))
	}
}

// Remove removes the named entry from the cache. If the entry
// is not present in the cache, the operation is a no-op.
func (cc *genericResourceCache) Remove(identifiers ...string) {
	if len(identifiers) == 0 {
		return
	}
	cc.Lock()
	sort.Sort(genericResource(cc.values))
	for _, n := range identifiers {
		cc.remove(n)
	}
	cc.Unlock()
}

// remove removes the named entry from the cache.
// invariant: cc.values should be sorted on entry.
func (cc *genericResourceCache) remove(identifier string) {
	i := sort.Search(len(cc.values), func(i int) bool { return cc.values[i].Identifier >= identifier })
	if i < len(cc.values) && cc.values[i].Identifier == identifier {
		// c is present, remove
		cc.values = append(cc.values[:i], cc.values[i+1:]...)
	}
}

// Cond implements a condition variable, a rendezvous point for goroutines
// waiting for or announcing the occurence of an event.
type Cond struct {
	mu      sync.Mutex
	waiters []chan int
	last    int
}

// Register registers ch to receive a value when Notify is called.
func (c *Cond) Register(ch chan int, last int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if last < c.last {
		// notify this channel immediately
		ch <- c.last
		return
	}
	c.waiters = append(c.waiters, ch)
}

// Notify notifies all registered waiters that an event has occured.
func (c *Cond) Notify() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.last++

	for _, ch := range c.waiters {
		ch <- c.last
	}
	c.waiters = c.waiters[:0]
}
