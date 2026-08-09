package k8s

import (
	"container/list"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/keel-hq/keel/internal/workgroup"
	appconfig "github.com/keel-hq/keel/pkg/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	apps_v1 "k8s.io/api/apps/v1"
	batch_v1 "k8s.io/api/batch/v1"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	bufferEvents = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "kubernetes_event_buffer_events_total",
		Help: "Kubernetes resource events received by the event buffer.",
	}, []string{"event"})
	bufferCoalesced = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "kubernetes_event_buffer_coalesced_total",
		Help: "Kubernetes resource events merged into an already pending event for the same resource.",
	}, []string{"event"})
	bufferBackpressure = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "kubernetes_event_buffer_backpressure_total",
		Help: "Kubernetes resource events that waited because the distinct-resource queue was full.",
	})
	bufferDropped = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "kubernetes_event_buffer_dropped_total",
		Help: "Kubernetes resource states discarded by the event buffer.",
	}, []string{"reason"})
	bufferQueueDepth = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "kubernetes_event_buffer_queue_depth",
		Help: "Current number of distinct Kubernetes resources waiting in the event buffer.",
	})
	bufferQueueCapacity = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "kubernetes_event_buffer_queue_capacity",
		Help: "Maximum number of distinct Kubernetes resources in the event buffer.",
	})
)

func init() {
	prometheus.MustRegister(bufferEvents, bufferCoalesced, bufferBackpressure, bufferDropped, bufferQueueDepth, bufferQueueCapacity)
}

// WatchDeployments creates a SharedInformer for apps/v1.Deployments and registers it with g.
func WatchDeployments(g *workgroup.Group, client *kubernetes.Clientset, log logrus.FieldLogger, config appconfig.KubernetesConfig, rs ...cache.ResourceEventHandler) {
	watch(g, client.AppsV1().RESTClient(), log, config, "deployments", new(apps_v1.Deployment), rs...)
}

// WatchStatefulSets creates a SharedInformer for apps/v1.StatefulSet and registers it with g.
func WatchStatefulSets(g *workgroup.Group, client *kubernetes.Clientset, log logrus.FieldLogger, config appconfig.KubernetesConfig, rs ...cache.ResourceEventHandler) {
	watch(g, client.AppsV1().RESTClient(), log, config, "statefulsets", new(apps_v1.StatefulSet), rs...)
}

// WatchDaemonSets creates a SharedInformer for apps/v1.DaemonSet and registers it with g.
func WatchDaemonSets(g *workgroup.Group, client *kubernetes.Clientset, log logrus.FieldLogger, config appconfig.KubernetesConfig, rs ...cache.ResourceEventHandler) {
	watch(g, client.AppsV1().RESTClient(), log, config, "daemonsets", new(apps_v1.DaemonSet), rs...)
}

// WatchCronJobs creates a SharedInformer for batch_v1.CronJob and registers it with g.
func WatchCronJobs(g *workgroup.Group, client *kubernetes.Clientset, log logrus.FieldLogger, config appconfig.KubernetesConfig, rs ...cache.ResourceEventHandler) {
	watch(g, client.BatchV1().RESTClient(), log, config, "cronjobs", new(batch_v1.CronJob), rs...)
}

func watch(g *workgroup.Group, c cache.Getter, log logrus.FieldLogger, config appconfig.KubernetesConfig, resource string, objType runtime.Object, rs ...cache.ResourceEventHandler) {
	lw := cache.NewListWatchFromClient(c, resource, namespaceFor(config), fields.Everything())
	sw := cache.NewSharedInformer(lw, objType, 30*time.Minute)
	for _, r := range rs {
		sw.AddEventHandler(r)
	}
	g.Add(func(stop <-chan struct{}) {
		log := log.WithField("resource", resource)
		log.Println("started")
		defer log.Println("stopped")
		sw.Run(stop)
	})
}

func namespaceFor(config appconfig.KubernetesConfig) string {
	if config.RestrictedNamespace == "" || config.RestrictedNamespace == "keel" {
		return v1.NamespaceAll
	}
	return config.RestrictedNamespace
}

type buffer struct {
	mu                sync.Mutex
	queue             *list.List
	pending           map[string]*list.Element
	capacity          int
	wake              chan struct{}
	space             *sync.Cond
	done              chan struct{}
	stopOnce          sync.Once
	sequence          uint64
	lastSaturationLog time.Time

	log logrus.FieldLogger
	rh  cache.ResourceEventHandler
}

type queuedEvent struct {
	key string
	ev  interface{}
}

type addEvent struct {
	obj             interface{}
	isInInitialList bool
}

type updateEvent struct {
	oldObj, newObj interface{}
}

type deleteEvent struct {
	obj interface{}
}

// NewBuffer returns a ResourceEventHandler which buffers and serialises events.
// Pending events for the same Kubernetes resource are coalesced to its latest
// state. Once size distinct resources are pending, producers apply backpressure.
func NewBuffer(g *workgroup.Group, rh cache.ResourceEventHandler, log logrus.FieldLogger, size int) cache.ResourceEventHandler {
	if size < 1 {
		size = 1
	}
	buf := &buffer{
		queue:    list.New(),
		pending:  make(map[string]*list.Element, size),
		capacity: size,
		wake:     make(chan struct{}, 1),
		done:     make(chan struct{}),
		log:      log.WithField("context", "buffer"),
		rh:       rh,
	}
	buf.space = sync.NewCond(&buf.mu)
	bufferQueueCapacity.Set(float64(size))
	g.Add(buf.loop)
	return buf
}

func (b *buffer) loop(stop <-chan struct{}) {
	b.log.Info("started")
	defer b.log.Info("stopped")
	go func() {
		<-stop
		b.shutdown()
	}()

	for {
		select {
		case <-b.wake:
			for {
				ev, ok := b.pop()
				if !ok {
					break
				}
				b.handle(ev)
			}
		case <-b.done:
			return
		}
	}
}

func (b *buffer) shutdown() {
	b.stopOnce.Do(func() {
		b.mu.Lock()
		pending := b.queue.Len()
		b.queue.Init()
		clear(b.pending)
		close(b.done)
		b.space.Broadcast()
		b.mu.Unlock()
		if pending > 0 {
			b.log.WithField("pending", pending).Warn("discarding pending events during shutdown")
			bufferDropped.WithLabelValues("shutdown").Add(float64(pending))
		}
		bufferQueueDepth.Set(0)
	})
}

func (b *buffer) pop() (interface{}, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	elem := b.queue.Front()
	if elem == nil {
		return nil, false
	}
	queued := elem.Value.(*queuedEvent)
	b.queue.Remove(elem)
	delete(b.pending, queued.key)
	bufferQueueDepth.Set(float64(b.queue.Len()))
	b.space.Signal()
	return queued.ev, true
}

func (b *buffer) handle(ev interface{}) {
	switch ev := ev.(type) {
	case *addEvent:
		b.rh.OnAdd(ev.obj, ev.isInInitialList)
	case *updateEvent:
		b.rh.OnUpdate(ev.oldObj, ev.newObj)
	case *deleteEvent:
		b.rh.OnDelete(ev.obj)
	default:
		b.log.Errorf("unhandled event type: %T: %v", ev, ev)
	}
}

func (b *buffer) OnAdd(obj interface{}, isInInitialList bool) {
	b.send(&addEvent{obj, isInInitialList})
}

func (b *buffer) OnUpdate(oldObj, newObj interface{}) {
	b.send(&updateEvent{oldObj, newObj})
}

func (b *buffer) OnDelete(obj interface{}) {
	b.send(&deleteEvent{obj})
}

func (b *buffer) send(ev interface{}) {
	eventType := eventName(ev)
	bufferEvents.WithLabelValues(eventType).Inc()
	key, _ := b.eventKey(ev) // Unkeyed events remain bounded but cannot safely be coalesced.
	waited := false
	for {
		b.mu.Lock()
		select {
		case <-b.done:
			b.mu.Unlock()
			bufferDropped.WithLabelValues("shutdown").Inc()
			return
		default:
		}
		if elem, ok := b.pending[key]; ok {
			queued := elem.Value.(*queuedEvent)
			queued.ev = coalesce(queued.ev, ev)
			b.mu.Unlock()
			bufferCoalesced.WithLabelValues(eventType).Inc()
			return
		}
		if b.queue.Len() < b.capacity {
			elem := b.queue.PushBack(&queuedEvent{key: key, ev: ev})
			b.pending[key] = elem
			bufferQueueDepth.Set(float64(b.queue.Len()))
			b.mu.Unlock()
			notify(b.wake)
			return
		}
		if !waited {
			waited = true
			bufferBackpressure.Inc()
			now := time.Now()
			if b.lastSaturationLog.IsZero() || now.Sub(b.lastSaturationLog) >= time.Minute {
				b.lastSaturationLog = now
				b.log.WithFields(logrus.Fields{"capacity": b.capacity, "event": eventType}).Warn("event buffer saturated; applying backpressure")
			}
		}
		b.space.Wait()
		b.mu.Unlock()
	}
}

func (b *buffer) eventKey(ev interface{}) (string, bool) {
	obj := eventObject(ev)
	if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		obj = tombstone.Obj
	}
	accessor, err := meta.Accessor(obj)
	if err == nil {
		typ := reflect.TypeOf(obj)
		if uid := accessor.GetUID(); uid != "" {
			return fmt.Sprintf("%v/uid/%s", typ, uid), true
		}
		if name := accessor.GetName(); name != "" {
			return fmt.Sprintf("%v/name/%s/%s", typ, accessor.GetNamespace(), name), true
		}
	}
	sequence := atomic.AddUint64(&b.sequence, 1)
	return fmt.Sprintf("unkeyed/%d", sequence), false
}

func eventObject(ev interface{}) interface{} {
	switch ev := ev.(type) {
	case *addEvent:
		return ev.obj
	case *updateEvent:
		return ev.newObj
	case *deleteEvent:
		return ev.obj
	default:
		return ev
	}
}

func eventName(ev interface{}) string {
	switch ev.(type) {
	case *addEvent:
		return "add"
	case *updateEvent:
		return "update"
	case *deleteEvent:
		return "delete"
	default:
		return "unknown"
	}
}

func coalesce(previous, next interface{}) interface{} {
	switch previous := previous.(type) {
	case *addEvent:
		switch next := next.(type) {
		case *addEvent:
			return &addEvent{obj: next.obj, isInInitialList: previous.isInInitialList}
		case *updateEvent:
			return &addEvent{obj: next.newObj, isInInitialList: previous.isInInitialList}
		case *deleteEvent:
			return next
		}
	case *updateEvent:
		switch next := next.(type) {
		case *addEvent:
			return &updateEvent{oldObj: previous.oldObj, newObj: next.obj}
		case *updateEvent:
			return &updateEvent{oldObj: previous.oldObj, newObj: next.newObj}
		case *deleteEvent:
			return next
		}
	case *deleteEvent:
		switch next := next.(type) {
		case *addEvent:
			return &updateEvent{oldObj: previous.obj, newObj: next.obj}
		case *updateEvent:
			return &updateEvent{oldObj: previous.obj, newObj: next.newObj}
		case *deleteEvent:
			return next
		}
	}
	return next
}

func notify(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}
