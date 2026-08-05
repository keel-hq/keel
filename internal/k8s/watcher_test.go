package k8s

import (
	"bytes"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/keel-hq/keel/internal/workgroup"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
)

const testTimeout = 5 * time.Second

type observedEvent struct {
	kind       string
	name       string
	oldVersion string
	version    string
}

type recordingHandler struct {
	mu          sync.Mutex
	events      []observedEvent
	latest      map[string]string
	changed     chan struct{}
	inFlight    int32
	maxInFlight int32
	before      func()
}

func newRecordingHandler() *recordingHandler {
	return &recordingHandler{latest: make(map[string]string), changed: make(chan struct{}, 1)}
}

func (h *recordingHandler) enter() {
	current := atomic.AddInt32(&h.inFlight, 1)
	for {
		maximum := atomic.LoadInt32(&h.maxInFlight)
		if current <= maximum || atomic.CompareAndSwapInt32(&h.maxInFlight, maximum, current) {
			break
		}
	}
	if h.before != nil {
		h.before()
	}
}

func (h *recordingHandler) leave(event observedEvent) {
	h.mu.Lock()
	h.events = append(h.events, event)
	h.latest[event.name] = event.version
	h.mu.Unlock()
	atomic.AddInt32(&h.inFlight, -1)
	notify(h.changed)
}

func (h *recordingHandler) OnAdd(obj interface{}, _ bool) {
	h.enter()
	deployment := obj.(*appsv1.Deployment)
	h.leave(observedEvent{kind: "add", name: deployment.Name, version: deployment.ResourceVersion})
}

func (h *recordingHandler) OnUpdate(oldObj, newObj interface{}) {
	h.enter()
	oldDeployment := oldObj.(*appsv1.Deployment)
	newDeployment := newObj.(*appsv1.Deployment)
	h.leave(observedEvent{kind: "update", name: newDeployment.Name, oldVersion: oldDeployment.ResourceVersion, version: newDeployment.ResourceVersion})
}

func (h *recordingHandler) OnDelete(obj interface{}) {
	h.enter()
	if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		obj = tombstone.Obj
	}
	deployment := obj.(*appsv1.Deployment)
	h.leave(observedEvent{kind: "delete", name: deployment.Name, version: deployment.ResourceVersion})
}

func (h *recordingHandler) waitLatest(t testing.TB, expected map[string]string) {
	t.Helper()
	deadline := time.NewTimer(testTimeout)
	defer deadline.Stop()
	for {
		h.mu.Lock()
		matches := len(h.latest) >= len(expected)
		for name, version := range expected {
			matches = matches && h.latest[name] == version
		}
		h.mu.Unlock()
		if matches {
			return
		}
		select {
		case <-h.changed:
		case <-deadline.C:
			h.mu.Lock()
			latest := make(map[string]string, len(h.latest))
			for name, version := range h.latest {
				latest[name] = version
			}
			h.mu.Unlock()
			t.Fatalf("timed out waiting for latest states: got %v, want %v", latest, expected)
		}
	}
}

func deployment(name, uid, version string) *appsv1.Deployment {
	return &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
		Namespace:       "default",
		Name:            name,
		UID:             types.UID(uid),
		ResourceVersion: version,
	}}
}

func startTestBuffer(t *testing.T, handler cache.ResourceEventHandler, capacity int, logger logrus.FieldLogger) (*buffer, chan struct{}) {
	t.Helper()
	var group workgroup.Group
	b := NewBuffer(&group, handler, logger, capacity).(*buffer)
	stop := make(chan struct{})
	exited := make(chan struct{})
	go func() {
		b.loop(stop)
		close(exited)
	}()
	t.Cleanup(func() {
		select {
		case <-stop:
		default:
			close(stop)
		}
		select {
		case <-exited:
		case <-time.After(testTimeout):
			t.Error("buffer loop did not stop")
		}
	})
	return b, stop
}

func TestBufferCapacityBurstCoalescesWithoutLosingLatestState(t *testing.T) {
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	var calls int32
	handler := newRecordingHandler()
	handler.before = func() {
		if atomic.AddInt32(&calls, 1) == 1 {
			close(firstStarted)
			<-releaseFirst
		}
	}
	var logs bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&logs)
	b, _ := startTestBuffer(t, handler, 2, logger)

	b.OnAdd(deployment("zero", "zero", "1"), false)
	select {
	case <-firstStarted:
	case <-time.After(testTimeout):
		t.Fatal("slow consumer did not start")
	}
	b.OnAdd(deployment("one", "one", "1"), false)
	b.OnAdd(deployment("two", "two", "1"), false)

	thirdQueued := make(chan struct{})
	go func() {
		b.OnAdd(deployment("three", "three", "1"), false)
		close(thirdQueued)
	}()
	select {
	case <-thirdQueued:
		t.Fatal("a new resource bypassed backpressure at capacity")
	case <-time.After(25 * time.Millisecond):
	}

	duplicateReturned := make(chan struct{})
	go func() {
		b.OnUpdate(deployment("two", "two", "1"), deployment("two", "two", "latest"))
		close(duplicateReturned)
	}()
	select {
	case <-duplicateReturned:
	case <-time.After(testTimeout):
		t.Fatal("duplicate resource did not coalesce while the queue was full")
	}
	close(releaseFirst)
	select {
	case <-thirdQueued:
	case <-time.After(testTimeout):
		t.Fatal("backpressured producer did not resume")
	}

	handler.waitLatest(t, map[string]string{"zero": "1", "one": "1", "two": "latest", "three": "1"})
	if got := atomic.LoadInt32(&handler.maxInFlight); got != 1 {
		t.Fatalf("handler ran concurrently: max in flight = %d", got)
	}
	if !bytes.Contains(logs.Bytes(), []byte("applying backpressure")) {
		t.Fatalf("saturation warning missing from logs: %s", logs.String())
	}
}

func TestBufferCoalescingPreservesStateTransitionsAndResourceIdentity(t *testing.T) {
	handler := newRecordingHandler()
	var group workgroup.Group
	b := NewBuffer(&group, handler, logrus.New(), 8).(*buffer)

	b.OnUpdate(deployment("app", "uid-1", "0"), deployment("app", "uid-1", "1"))
	b.OnUpdate(deployment("app", "uid-1", "1"), deployment("app", "uid-1", "2"))
	b.OnDelete(deployment("gone", "uid-2", "3"))
	b.OnAdd(deployment("gone", "uid-2", "4"), false)
	b.OnAdd(deployment("same-name", "old-uid", "1"), false)
	b.OnAdd(deployment("same-name", "new-uid", "2"), false)

	first := b.queue.Front().Value.(*queuedEvent).ev.(*updateEvent)
	if oldVersion := first.oldObj.(*appsv1.Deployment).ResourceVersion; oldVersion != "0" {
		t.Fatalf("coalesced update lost its original state: got %q", oldVersion)
	}
	if newVersion := first.newObj.(*appsv1.Deployment).ResourceVersion; newVersion != "2" {
		t.Fatalf("coalesced update lost its latest state: got %q", newVersion)
	}
	second := b.queue.Front().Next().Value.(*queuedEvent).ev.(*updateEvent)
	if second.oldObj.(*appsv1.Deployment).ResourceVersion != "3" || second.newObj.(*appsv1.Deployment).ResourceVersion != "4" {
		t.Fatalf("delete/add transition was not preserved: %#v", second)
	}
	if got := b.queue.Len(); got != 4 {
		t.Fatalf("different UIDs with the same name were coalesced: queue length = %d, want 4", got)
	}
	b.shutdown()
}

func TestBufferSustainedLoadEventuallyProcessesEveryResourceLatestState(t *testing.T) {
	handler := newRecordingHandler()
	handler.before = func() { time.Sleep(10 * time.Microsecond) }
	b, _ := startTestBuffer(t, handler, 8, logrus.New())

	const resources = 32
	const versions = 200
	var producers sync.WaitGroup
	expected := make(map[string]string, resources)
	for resource := 0; resource < resources; resource++ {
		name := fmt.Sprintf("resource-%02d", resource)
		expected[name] = fmt.Sprint(versions)
		producers.Add(1)
		go func(name string) {
			defer producers.Done()
			uid := "uid-" + name
			b.OnAdd(deployment(name, uid, "0"), false)
			for version := 1; version <= versions; version++ {
				b.OnUpdate(deployment(name, uid, fmt.Sprint(version-1)), deployment(name, uid, fmt.Sprint(version)))
			}
		}(name)
	}
	producers.Wait()
	handler.waitLatest(t, expected)
	if got := atomic.LoadInt32(&handler.maxInFlight); got != 1 {
		t.Fatalf("handler ran concurrently: max in flight = %d", got)
	}
	b.mu.Lock()
	queued := b.queue.Len()
	b.mu.Unlock()
	if queued > b.capacity {
		t.Fatalf("queue exceeded its bound: got %d, capacity %d", queued, b.capacity)
	}
}

func TestBufferShutdownUnblocksBackpressuredProducer(t *testing.T) {
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	var once sync.Once
	handler := newRecordingHandler()
	handler.before = func() {
		once.Do(func() {
			close(firstStarted)
			<-releaseFirst
		})
	}
	b, stop := startTestBuffer(t, handler, 1, logrus.New())
	b.OnAdd(deployment("active", "active", "1"), false)
	select {
	case <-firstStarted:
	case <-time.After(testTimeout):
		t.Fatal("consumer did not start")
	}
	b.OnAdd(deployment("queued", "queued", "1"), false)

	producerReturned := make(chan struct{})
	go func() {
		b.OnAdd(deployment("blocked", "blocked", "1"), false)
		close(producerReturned)
	}()
	select {
	case <-producerReturned:
		t.Fatal("producer was not backpressured")
	case <-time.After(25 * time.Millisecond):
	}
	close(stop)
	select {
	case <-producerReturned:
	case <-time.After(testTimeout):
		t.Fatal("shutdown did not unblock producer")
	}
	close(releaseFirst)
}

func BenchmarkBufferSustainedLoad(b *testing.B) {
	handler := newRecordingHandler()
	var group workgroup.Group
	queue := NewBuffer(&group, handler, logrus.New(), 128).(*buffer)
	stop := make(chan struct{})
	go queue.loop(stop)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resource := i % 256
		name := fmt.Sprintf("resource-%03d", resource)
		version := fmt.Sprint(i)
		queue.OnUpdate(deployment(name, "uid-"+name, version), deployment(name, "uid-"+name, version))
	}
	b.StopTimer()
	expected := make(map[string]string, min(b.N, 256))
	for resource := 0; resource < min(b.N, 256); resource++ {
		last := b.N - 1 - ((b.N - 1 - resource) % 256)
		expected[fmt.Sprintf("resource-%03d", resource)] = fmt.Sprint(last)
	}
	handler.waitLatest(b, expected)
	close(stop)
	<-queue.done
}
