package kubernetes

import (
	"strings"
	"sync"
	"time"

	"github.com/keel-hq/keel/internal/k8s"

	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DefaultPodCacheTTL is how long a pod listing is reused before it is fetched
// again. Workload definitions come from an informer backed cache, but pods are
// listed straight from the API server: every TrackedImages() enumeration lists
// the pods of every tracked workload twice (platform resolution and running
// digest resolution), and TrackedImages() runs once per poll scan plus once per
// watcher tick. Without a cache the request rate grows with
// workloads x watchers and saturates the client side rate limiter.
const DefaultPodCacheTTL = 30 * time.Second

type podCacheEntry struct {
	mu        sync.Mutex
	fetchedAt time.Time
	list      *v1.PodList
}

// CachingImplementer decorates an Implementer with a short lived cache for pod
// lookups. Everything else is forwarded untouched.
//
// The cached list is shared between callers, so callers must treat the returned
// PodList as read only - which is what every caller in this repository does.
// Entries for a namespace are dropped as soon as Keel changes anything in it,
// so an update is never planned against a pod listing from before the rollout.
type CachingImplementer struct {
	Implementer

	ttl time.Duration
	now func() time.Time

	mu      sync.Mutex
	entries map[string]*podCacheEntry
}

// NewCachingImplementer wraps impl with a pod listing cache. A ttl of zero or
// less selects DefaultPodCacheTTL.
func NewCachingImplementer(impl Implementer, ttl time.Duration) *CachingImplementer {
	if ttl <= 0 {
		ttl = DefaultPodCacheTTL
	}
	return &CachingImplementer{
		Implementer: impl,
		ttl:         ttl,
		now:         time.Now,
		entries:     make(map[string]*podCacheEntry),
	}
}

// Pods returns the pods matching labelSelector, reusing a recent listing when
// there is one. Concurrent callers asking for the same namespace and selector
// share a single API request instead of queueing one each behind the rate
// limiter.
func (c *CachingImplementer) Pods(namespace, labelSelector string) (*v1.PodList, error) {
	entry := c.entryFor(cacheKey(namespace, labelSelector))

	entry.mu.Lock()
	defer entry.mu.Unlock()

	if entry.list != nil && c.now().Sub(entry.fetchedAt) < c.ttl {
		return entry.list, nil
	}

	list, err := c.Implementer.Pods(namespace, labelSelector)
	if err != nil {
		// failures are not cached: a transient API error should not blind the
		// next caller for a full TTL
		return list, err
	}

	entry.list = list
	entry.fetchedAt = c.now()

	return list, nil
}

// Update applies the resource change and drops the cached pod listings of its
// namespace, because the pods that are about to be replaced no longer describe
// what the workload runs.
func (c *CachingImplementer) Update(obj *k8s.GenericResource) error {
	err := c.Implementer.Update(obj)
	if obj != nil {
		c.invalidate(obj.GetNamespace())
	}
	return err
}

// DeletePod removes the pod and drops the cached pod listings of its namespace.
func (c *CachingImplementer) DeletePod(namespace, name string, opts *meta_v1.DeleteOptions) error {
	err := c.Implementer.DeletePod(namespace, name, opts)
	c.invalidate(namespace)
	return err
}

func (c *CachingImplementer) entryFor(key string) *podCacheEntry {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if !ok {
		entry = &podCacheEntry{}
		c.entries[key] = entry
	}
	return entry
}

func (c *CachingImplementer) invalidate(namespace string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	prefix := namespace + "/"
	for key := range c.entries {
		if strings.HasPrefix(key, prefix) {
			delete(c.entries, key)
		}
	}
}

// cacheKey is unambiguous because a namespace name cannot contain a slash,
// while a label selector can.
func cacheKey(namespace, labelSelector string) string {
	return namespace + "/" + labelSelector
}
