package k8s

import (
	"os"
	"time"

	"github.com/keel-hq/keel/constants"
	"github.com/keel-hq/keel/internal/workgroup"
	"github.com/sirupsen/logrus"

	apps_v1 "k8s.io/api/apps/v1"
	v1beta1 "k8s.io/api/batch/v1beta1"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// WatchDeployments creates a SharedInformer for apps/v1.Deployments and registers it with g.
func WatchDeployments(g *workgroup.Group, client *kubernetes.Clientset, log logrus.FieldLogger, rs ...cache.ResourceEventHandler) {
	watch(g, client.AppsV1().RESTClient(), log, "deployments", new(apps_v1.Deployment), rs...)
}

// WatchStatefulSets creates a SharedInformer for apps/v1.StatefulSet and registers it with g.
func WatchStatefulSets(g *workgroup.Group, client *kubernetes.Clientset, log logrus.FieldLogger, rs ...cache.ResourceEventHandler) {
	watch(g, client.AppsV1().RESTClient(), log, "statefulsets", new(apps_v1.StatefulSet), rs...)
}

// WatchDaemonSets creates a SharedInformer for apps/v1.DaemonSet and registers it with g.
func WatchDaemonSets(g *workgroup.Group, client *kubernetes.Clientset, log logrus.FieldLogger, rs ...cache.ResourceEventHandler) {
	watch(g, client.AppsV1().RESTClient(), log, "daemonsets", new(apps_v1.DaemonSet), rs...)
}

// WatchCronJobs creates a SharedInformer for v1beta1.CronJob and registers it with g.
func WatchCronJobs(g *workgroup.Group, client *kubernetes.Clientset, log logrus.FieldLogger, rs ...cache.ResourceEventHandler) {
	watch(g, client.BatchV1beta1().RESTClient(), log, "cronjobs", new(v1beta1.CronJob), rs...)
}

func watch(g *workgroup.Group, c cache.Getter, log logrus.FieldLogger, resource string, objType runtime.Object, rs ...cache.ResourceEventHandler) {
	//Check if the env var RESTRICTED_NAMESPACE is empty or equal to keel
	// If equal to keel or empty, the scan will be over all the cluster
	// If RESTRICTED_NAMESPACE is different than keel or empty, keel will scan in the defined namespace
	namespaceScan := "keel"
	if os.Getenv(constants.EnvRestrictedNamespace) == "keel" {
		namespaceScan = v1.NamespaceAll
	} else if os.Getenv(constants.EnvRestrictedNamespace) == "" {
		namespaceScan = v1.NamespaceAll
	} else {
		namespaceScan = os.Getenv(constants.EnvRestrictedNamespace)
	}

	lw := cache.NewListWatchFromClient(c, resource, namespaceScan, fields.Everything())
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

type buffer struct {
	ev chan interface{}
	logrus.StdLogger
	rh cache.ResourceEventHandler
}

type addEvent struct {
	obj interface{}
}

type updateEvent struct {
	oldObj, newObj interface{}
}

type deleteEvent struct {
	obj interface{}
}

// NewBuffer returns a ResourceEventHandler which buffers and serialises ResourceEventHandler events.
func NewBuffer(g *workgroup.Group, rh cache.ResourceEventHandler, log logrus.FieldLogger, size int) cache.ResourceEventHandler {
	buf := &buffer{
		ev:        make(chan interface{}, size),
		StdLogger: log.WithField("context", "buffer"),
		rh:        rh,
	}
	g.Add(buf.loop)
	return buf
}

func (b *buffer) loop(stop <-chan struct{}) {
	b.Println("started")
	defer b.Println("stopped")

	for {
		select {
		case ev := <-b.ev:
			switch ev := ev.(type) {
			case *addEvent:
				b.rh.OnAdd(ev.obj)
			case *updateEvent:
				b.rh.OnUpdate(ev.oldObj, ev.newObj)
			case *deleteEvent:
				b.rh.OnDelete(ev.obj)
			default:
				b.Printf("unhandled event type: %T: %v", ev, ev)
			}
		case <-stop:
			return
		}
	}
}

func (b *buffer) OnAdd(obj interface{}) {
	b.send(&addEvent{obj})
}

func (b *buffer) OnUpdate(oldObj, newObj interface{}) {
	b.send(&updateEvent{oldObj, newObj})
}

func (b *buffer) OnDelete(obj interface{}) {
	b.send(&deleteEvent{obj})
}

func (b *buffer) send(ev interface{}) {
	select {
	case b.ev <- ev:
		// all good
	default:
		b.Printf("event channel is full, len: %v, cap: %v", len(b.ev), cap(b.ev))
		b.ev <- ev
	}
}
