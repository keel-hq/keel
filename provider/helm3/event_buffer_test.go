package helm3

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/keel-hq/keel/extension/notification"
	"github.com/keel-hq/keel/pkg/config"
	"github.com/keel-hq/keel/types"

	hapi_chart "helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"

	"github.com/stretchr/testify/require"
)

// threadSafeSender is a notification sender that is safe to call from
// multiple goroutines (applyPlan runs per-release workers concurrently).
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

// slowHelmImplementer records every release update and simulates API
// latency.
type slowHelmImplementer struct {
	fakeImplementer
	delay   time.Duration
	mu      sync.Mutex
	updated []string
}

func (i *slowHelmImplementer) UpdateReleaseFromChart(rlsName string, chart *hapi_chart.Chart, vals map[string]string, namespace string, opts ...bool) (*release.Release, error) {
	time.Sleep(i.delay)
	i.mu.Lock()
	defer i.mu.Unlock()
	i.updated = append(i.updated, namespace+"/"+rlsName)
	return &release.Release{Name: rlsName, Chart: chart, Version: 2}, nil
}

func (i *slowHelmImplementer) updates() []string {
	i.mu.Lock()
	defer i.mu.Unlock()
	return append([]string(nil), i.updated...)
}

func TestEventBufferDefaultSize(t *testing.T) {
	p := NewProvider(&fakeImplementer{}, &fakeSender{}, nil)
	require.Equal(t, config.DefaultEventBufferSize, cap(p.events))
}

// TestSubmitBackpressureBlocksInsteadOfDropping pins the fix for
// keel-hq/keel#443: when the event buffer is saturated Submit applies
// backpressure (it blocks until a slot frees) instead of silently dropping
// the event.
func TestSubmitBackpressureBlocksInsteadOfDropping(t *testing.T) {
	p := NewProvider(&fakeImplementer{}, &fakeSender{}, nil)
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
	p := NewProvider(&fakeImplementer{}, &fakeSender{}, nil)
	p.Stop()

	err := p.Submit(types.Event{Repository: types.Repository{Name: "gcr.io/ns/img", Tag: "1.0.1"}})
	require.ErrorIs(t, err, ErrProviderStopped)
}

// TestSubmitUnderConcurrentLoadDrainsAllEvents simulates a burst of
// concurrent trigger submissions against a small buffer and a live consumer:
// every event must be accepted (no drops) and the buffer must fully drain.
func TestSubmitUnderConcurrentLoadDrainsAllEvents(t *testing.T) {
	p := NewProvider(&fakeImplementer{}, &threadSafeSender{}, nil)
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

// TestApplyPlansParallelUnderLoad verifies the head-of-line blocking fix:
// an event that fans out to many releases is applied by a bounded worker
// pool, so the total time grows with (releases / workers) instead of with
// (releases), and no release update is lost.
func TestApplyPlansParallelUnderLoad(t *testing.T) {
	const (
		n     = 30
		delay = 30 * time.Millisecond
	)

	impl := &slowHelmImplementer{delay: delay}
	fs := &threadSafeSender{}
	approver, teardown := approver()
	defer teardown()
	p := NewProvider(impl, fs, approver)

	plans := make([]*UpdatePlan, n)
	for i := 0; i < n; i++ {
		plans[i] = &UpdatePlan{
			Namespace:      "ns",
			Name:           fmt.Sprintf("release-%d", i),
			Config:         &KeelChartConfig{},
			Chart:          &hapi_chart.Chart{Metadata: &hapi_chart.Metadata{Name: "test-chart", Version: "1.0.0"}},
			Values:         map[string]string{"image.tag": "1.4.5"},
			CurrentVersion: "1.1.1",
			NewVersion:     "1.4.5",
		}
	}

	start := time.Now()
	require.NoError(t, p.applyPlans(plans))
	elapsed := time.Since(start)

	require.Len(t, impl.updates(), n, "every release of the event must be updated exactly once")
	require.Equal(t, 2*n, fs.count(), "pre-update and post-update notifications expected for every release")

	serial := time.Duration(n) * delay
	require.Less(t, elapsed, serial/2,
		"parallel apply should beat serial (%s) by at least 2x, took %s", serial, elapsed)
}
