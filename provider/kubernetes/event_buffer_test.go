package kubernetes

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/keel-hq/keel/extension/notification"
	"github.com/keel-hq/keel/internal/k8s"
	"github.com/keel-hq/keel/pkg/config"
	"github.com/keel-hq/keel/types"

	apps_v1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/require"
)

// threadSafeSender is a notification sender that records every event and is
// safe to call from multiple goroutines (applyPlan runs per-deployment
// workers concurrently).
type threadSafeSender struct {
	mu   sync.Mutex
	sent []types.EventNotification
}

func (s *threadSafeSender) Configure(cfg *notification.Config) (bool, error) {
	return true, nil
}

func (s *threadSafeSender) Send(event types.EventNotification) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sent = append(s.sent, event)
	return nil
}

func (s *threadSafeSender) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.sent)
}

// slowImplementer records every update and simulates API latency.
type slowImplementer struct {
	fakeImplementer
	delay   time.Duration
	mu      sync.Mutex
	updated []string
}

func (i *slowImplementer) Update(obj *k8s.GenericResource) error {
	time.Sleep(i.delay)
	i.mu.Lock()
	defer i.mu.Unlock()
	i.updated = append(i.updated, obj.GetIdentifier())
	return nil
}

func (i *slowImplementer) updates() []string {
	i.mu.Lock()
	defer i.mu.Unlock()
	return append([]string(nil), i.updated...)
}

func TestEventBufferDefaultSize(t *testing.T) {
	p, err := NewProvider(&fakeImplementer{}, &fakeSender{}, nil, &k8s.GenericResourceCache{})
	require.NoError(t, err)
	require.Equal(t, config.DefaultEventBufferSize, cap(p.events))
}

// TestSubmitBackpressureBlocksInsteadOfDropping pins the fix for
// keel-hq/keel#443: when the event buffer is saturated Submit applies
// backpressure (it blocks until a slot frees) instead of silently dropping
// the event.
func TestSubmitBackpressureBlocksInsteadOfDropping(t *testing.T) {
	p, err := NewProvider(&fakeImplementer{}, &fakeSender{}, nil, &k8s.GenericResourceCache{})
	require.NoError(t, err)
	p.events = make(chan *types.Event, 2)

	event := func(tag string) types.Event {
		return types.Event{Repository: types.Repository{Name: "gcr.io/ns/img", Tag: tag}}
	}

	require.NoError(t, p.Submit(event("1.0.1")))
	require.NoError(t, p.Submit(event("1.0.2")))

	blocked := make(chan error, 1)
	go func() {
		blocked <- p.Submit(event("1.0.3"))
	}()

	select {
	case err := <-blocked:
		t.Fatalf("Submit completed while the buffer was full; expected backpressure, got err=%v", err)
	case <-time.After(200 * time.Millisecond):
		// still blocked: the event is being held, not dropped
	}

	// Draining one slot unblocks the pending submit.
	require.Equal(t, "1.0.1", (<-p.events).Repository.Tag)
	require.NoError(t, <-blocked)

	require.Equal(t, "1.0.2", (<-p.events).Repository.Tag)
	require.Equal(t, "1.0.3", (<-p.events).Repository.Tag)
	require.Empty(t, p.events)
}

func TestSubmitAfterStopFailsFast(t *testing.T) {
	p, err := NewProvider(&fakeImplementer{}, &fakeSender{}, nil, &k8s.GenericResourceCache{})
	require.NoError(t, err)
	p.Stop()

	err = p.Submit(types.Event{Repository: types.Repository{Name: "gcr.io/ns/img", Tag: "1.0.1"}})
	require.ErrorIs(t, err, ErrProviderStopped)
}

// TestSubmitUnderConcurrentLoadDrainsAllEvents simulates a burst of
// concurrent trigger submissions (many deployments producing events at
// once) against a small buffer and a live consumer: every event must be
// accepted (no drops) and the buffer must fully drain.
func TestSubmitUnderConcurrentLoadDrainsAllEvents(t *testing.T) {
	grc := &k8s.GenericResourceCache{}
	p, err := NewProvider(&fakeImplementer{}, &threadSafeSender{}, nil, grc)
	require.NoError(t, err)
	p.events = make(chan *types.Event, 2)

	go p.Start()
	defer p.Stop()

	const total = 50
	submits := make(chan error, total)
	for i := 0; i < total; i++ {
		go func(i int) {
			submits <- p.Submit(types.Event{Repository: types.Repository{
				Name: fmt.Sprintf("gcr.io/ns/img-%d", i),
				Tag:  "1.0.0",
			}})
		}(i)
	}
	for i := 0; i < total; i++ {
		require.NoError(t, <-submits)
	}

	require.Eventually(t, func() bool {
		return len(p.events) == 0
	}, 10*time.Second, 10*time.Millisecond, "event buffer did not drain after all submissions")
}

// TestUpdateDeploymentsParallelUnderLoad verifies the head-of-line blocking
// fix: an event that fans out to many deployments is applied by a bounded
// worker pool, so the total time grows with (deployments / workers) instead
// of with (deployments), and no update is lost.
func TestUpdateDeploymentsParallelUnderLoad(t *testing.T) {
	const (
		n     = 30
		delay = 30 * time.Millisecond
	)

	deps := make([]*apps_v1.Deployment, n)
	for i := 0; i < n; i++ {
		deps[i] = &apps_v1.Deployment{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:        fmt.Sprintf("deployment-%d", i),
				Namespace:   fmt.Sprintf("ns-%d", i),
				Labels:      map[string]string{types.KeelPolicyLabel: "all"},
				Annotations: map[string]string{},
			},
			apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					ObjectMeta: meta_v1.ObjectMeta{
						Annotations: map[string]string{"this": "that"},
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{Image: "gcr.io/v2-namespace/hello-world:1.1.1"},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		}
	}

	grc := &k8s.GenericResourceCache{}
	grc.Add(MustParseGRS(deps)...)
	approver, teardown := approver()
	defer teardown()

	impl := &slowImplementer{delay: delay}
	fs := &threadSafeSender{}
	p, err := NewProvider(impl, fs, approver, grc)
	require.NoError(t, err)

	event := &types.Event{Repository: types.Repository{Name: "gcr.io/v2-namespace/hello-world", Tag: "1.4.5"}}
	plans, err := p.createUpdatePlansForEvent(event)
	require.NoError(t, err)
	require.Len(t, plans, n)

	start := time.Now()
	updated, err := p.updateDeployments(plans)
	elapsed := time.Since(start)
	require.NoError(t, err)

	require.Len(t, updated, n)
	require.Len(t, impl.updates(), n, "every deployment of the event must be updated exactly once")
	require.Equal(t, 2*n, fs.count(), "pre-update and post-update notifications expected for every deployment")

	serial := time.Duration(n) * delay
	require.Less(t, elapsed, serial/2,
		"parallel apply should beat serial (%s) by at least 2x, took %s", serial, elapsed)
}
