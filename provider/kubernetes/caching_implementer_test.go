package kubernetes

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/keel-hq/keel/internal/k8s"

	apps_v1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// countingPodImplementer records how many pod listings actually reach the API.
// Only the methods exercised by CachingImplementer are implemented, the
// embedded interface is never called.
type countingPodImplementer struct {
	Implementer

	mu    sync.Mutex
	calls int
	err   error
}

func (c *countingPodImplementer) Pods(namespace, labelSelector string) (*v1.PodList, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	if c.err != nil {
		return nil, c.err
	}
	return &v1.PodList{Items: []v1.Pod{{ObjectMeta: meta_v1.ObjectMeta{Name: "pod", Namespace: namespace}}}}, nil
}

func (c *countingPodImplementer) Update(obj *k8s.GenericResource) error { return nil }

func (c *countingPodImplementer) DeletePod(namespace, name string, opts *meta_v1.DeleteOptions) error {
	return nil
}

func (c *countingPodImplementer) callCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

func TestCachingImplementerReusesRecentPodListing(t *testing.T) {
	fake := &countingPodImplementer{}
	caching := NewCachingImplementer(fake, time.Minute)

	for i := 0; i < 5; i++ {
		if _, err := caching.Pods("default", "app=foo"); err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
	}

	if fake.callCount() != 1 {
		t.Errorf("expected 1 API call, got %d", fake.callCount())
	}
}

func TestCachingImplementerSeparatesNamespacesAndSelectors(t *testing.T) {
	fake := &countingPodImplementer{}
	caching := NewCachingImplementer(fake, time.Minute)

	caching.Pods("default", "app=foo")
	caching.Pods("default", "app=bar")
	caching.Pods("other", "app=foo")
	caching.Pods("default", "app=foo")

	if fake.callCount() != 3 {
		t.Errorf("expected 3 API calls, got %d", fake.callCount())
	}
}

func TestCachingImplementerRefetchesAfterTTL(t *testing.T) {
	fake := &countingPodImplementer{}
	caching := NewCachingImplementer(fake, 30*time.Second)

	now := time.Now()
	caching.now = func() time.Time { return now }

	caching.Pods("default", "app=foo")
	now = now.Add(29 * time.Second)
	caching.Pods("default", "app=foo")

	if fake.callCount() != 1 {
		t.Fatalf("expected the listing to be reused within the TTL, got %d calls", fake.callCount())
	}

	now = now.Add(2 * time.Second)
	caching.Pods("default", "app=foo")

	if fake.callCount() != 2 {
		t.Errorf("expected the listing to be refreshed after the TTL, got %d calls", fake.callCount())
	}
}

func TestCachingImplementerDoesNotCacheErrors(t *testing.T) {
	fake := &countingPodImplementer{err: errors.New("api is down")}
	caching := NewCachingImplementer(fake, time.Minute)

	if _, err := caching.Pods("default", "app=foo"); err == nil {
		t.Fatal("expected an error")
	}
	if _, err := caching.Pods("default", "app=foo"); err == nil {
		t.Fatal("expected an error")
	}

	if fake.callCount() != 2 {
		t.Errorf("expected the failed listing to be retried, got %d calls", fake.callCount())
	}
}

func TestCachingImplementerInvalidatesNamespaceOnUpdate(t *testing.T) {
	fake := &countingPodImplementer{}
	caching := NewCachingImplementer(fake, time.Minute)

	caching.Pods("default", "app=foo")
	caching.Pods("other", "app=foo")

	resource, err := k8s.NewGenericResource(&apps_v1.Deployment{
		ObjectMeta: meta_v1.ObjectMeta{Name: "foo", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("failed to create generic resource: %s", err)
	}

	if err := caching.Update(resource); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	caching.Pods("default", "app=foo")
	caching.Pods("other", "app=foo")

	if fake.callCount() != 3 {
		t.Errorf("expected the updated namespace to be refetched and the other one reused, got %d calls", fake.callCount())
	}
}

func TestCachingImplementerCollapsesConcurrentListings(t *testing.T) {
	fake := &countingPodImplementer{}
	caching := NewCachingImplementer(fake, time.Minute)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			caching.Pods("default", "app=foo")
		}()
	}
	wg.Wait()

	if fake.callCount() != 1 {
		t.Errorf("expected concurrent callers to share one API call, got %d", fake.callCount())
	}
}
