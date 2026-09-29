package k8s

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/keel-hq/keel/internal/workgroup"
	appconfig "github.com/keel-hq/keel/pkg/config"
	"github.com/sirupsen/logrus"

	core_v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	apiwatch "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	listers_v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

const (
	podListPageSize    = 500
	podSyncTimeout     = 30 * time.Second
	podRBACRemediation = "grant Keel's service account list and watch access to the core/v1 pods resource, then restart Keel"
)

// PodCache serves pod lookups from a shared informer so that resolving the
// pods of every tracked workload does not cost an API request per workload.
// Only the fields read by pod lookups are retained.
type PodCache struct {
	namespace string
	informer  cache.SharedIndexInformer
	lister    listers_v1.PodLister
	log       logrus.FieldLogger

	started  atomic.Bool
	ready    chan struct{}
	disabled atomic.Bool
	stop     chan struct{}
	stopOnce sync.Once
}

// WatchPods creates a pod cache and registers its informer with g.
func WatchPods(g *workgroup.Group, client kubernetes.Interface, log logrus.FieldLogger, config appconfig.KubernetesConfig) *PodCache {
	log = log.WithField("resource", "pods")
	pc := NewPodCache(client, config, log)
	g.Add(func(stop <-chan struct{}) {
		log.Println("started")
		defer log.Println("stopped")
		pc.Run(stop)
	})
	return pc
}

// NewPodCache creates a pod cache. It serves lookups once Run has synced it.
func NewPodCache(client kubernetes.Interface, config appconfig.KubernetesConfig, log logrus.FieldLogger) *PodCache {
	namespace := namespaceFor(config)
	lw := &cache.ListWatch{
		ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
			return listTrimmedPods(client, namespace, options)
		},
		WatchFunc: func(options meta_v1.ListOptions) (apiwatch.Interface, error) {
			w, err := client.CoreV1().Pods(namespace).Watch(context.TODO(), options)
			if err != nil {
				return nil, err
			}
			return apiwatch.Filter(w, trimPodEvent), nil
		},
	}
	// No event handlers are registered, so periodic resyncs would do nothing.
	informer := cache.NewSharedIndexInformer(lw, new(core_v1.Pod), 0, cache.Indexers{
		cache.NamespaceIndex: cache.MetaNamespaceIndexFunc,
	})
	pc := &PodCache{
		namespace: namespace,
		informer:  informer,
		lister:    listers_v1.NewPodLister(informer.GetIndexer()),
		log:       log,
		ready:     make(chan struct{}),
		stop:      make(chan struct{}),
	}
	if err := informer.SetWatchErrorHandler(pc.watchError); err != nil {
		panic(err)
	}
	return pc
}

// Run starts the informer and blocks until stop is closed. Disabling the
// cache stops the informer but not Run, as returning early would stop the
// workgroup.
func (c *PodCache) Run(stop <-chan struct{}) {
	c.started.Store(true)
	go func() {
		<-stop
		c.stopInformer()
	}()
	go c.awaitSync()
	c.informer.Run(c.stop)
	<-stop
}

// awaitSync marks the cache ready once it has synced, has been disabled, or
// podSyncTimeout has passed, whichever comes first.
func (c *PodCache) awaitSync() {
	defer close(c.ready)
	timeout := time.NewTimer(podSyncTimeout)
	defer timeout.Stop()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for !c.informer.HasSynced() && !c.disabled.Load() {
		select {
		case <-ticker.C:
		case <-timeout.C:
			c.log.Warn("pod cache has not synced yet; pod lookups will query the Kubernetes API until it does")
			return
		case <-c.stop:
			return
		}
	}
}

// HasSynced reports whether the initial pod list has been loaded.
func (c *PodCache) HasSynced() bool {
	return c.informer.HasSynced()
}

// Serves reports whether lookups in namespace can be answered by the cache.
// While a running cache performs its initial sync, Serves waits for it for up
// to podSyncTimeout, so startup scans do not query the API for every workload.
func (c *PodCache) Serves(namespace string) bool {
	if c == nil || namespace == "" {
		return false
	}
	if c.started.Load() {
		<-c.ready
	}
	if c.disabled.Load() || !c.informer.HasSynced() {
		return false
	}
	return c.namespace == core_v1.NamespaceAll || c.namespace == namespace
}

// Pods returns the cached pods in namespace that match labelSelector. The
// returned pods are shared with the cache and must not be modified.
func (c *PodCache) Pods(namespace, labelSelector string) (*core_v1.PodList, error) {
	selector, err := labels.Parse(labelSelector)
	if err != nil {
		return nil, fmt.Errorf("invalid pod selector %q: %w", labelSelector, err)
	}
	pods, err := c.lister.Pods(namespace).List(selector)
	if err != nil {
		return nil, err
	}
	list := &core_v1.PodList{Items: make([]core_v1.Pod, 0, len(pods))}
	for _, pod := range pods {
		list.Items = append(list.Items, *pod)
	}
	return list, nil
}

// watchError disables the cache when Keel may not list or watch pods, so that
// lookups fall back to per-selector API requests instead of the informer
// relisting every pod in a loop.
func (c *PodCache) watchError(r *cache.Reflector, err error) {
	if !apierrors.IsForbidden(err) {
		cache.DefaultWatchErrorHandler(r, err)
		return
	}
	if c.disabled.Swap(true) {
		return
	}
	c.log.WithFields(logrus.Fields{
		"error":       err,
		"remediation": podRBACRemediation,
	}).Warn("pod cache disabled; pod lookups will query the Kubernetes API for each workload")
	c.stopInformer()
}

func (c *PodCache) stopInformer() {
	c.stopOnce.Do(func() { close(c.stop) })
}

// listTrimmedPods lists pods page by page and trims each page before
// requesting the next one, so a sync never holds every full pod object in
// memory at once.
func listTrimmedPods(client kubernetes.Interface, namespace string, options meta_v1.ListOptions) (*core_v1.PodList, error) {
	// Pagination requires a consistent read: the API server ignores limit
	// when serving resourceVersion "0" from its watch cache.
	options.ResourceVersion = ""
	options.ResourceVersionMatch = ""
	options.Limit = podListPageSize
	options.Continue = ""

	result := &core_v1.PodList{}
	for {
		page, err := client.CoreV1().Pods(namespace).List(context.TODO(), options)
		if err != nil {
			return nil, err
		}
		if result.ResourceVersion == "" {
			result.ResourceVersion = page.ResourceVersion
		}
		for i := range page.Items {
			result.Items = append(result.Items, *trimmedPod(&page.Items[i]))
		}
		if page.Continue == "" {
			return result, nil
		}
		options.Continue = page.Continue
	}
}

func trimPodEvent(event apiwatch.Event) (apiwatch.Event, bool) {
	if pod, ok := event.Object.(*core_v1.Pod); ok {
		event.Object = trimmedPod(pod)
	}
	return event, true
}

func trimmedPod(pod *core_v1.Pod) *core_v1.Pod {
	return &core_v1.Pod{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:              pod.Name,
			Namespace:         pod.Namespace,
			UID:               pod.UID,
			ResourceVersion:   pod.ResourceVersion,
			Labels:            pod.Labels,
			DeletionTimestamp: pod.DeletionTimestamp,
		},
		Spec: core_v1.PodSpec{
			NodeName:         pod.Spec.NodeName,
			ImagePullSecrets: pod.Spec.ImagePullSecrets,
			InitContainers:   trimContainers(pod.Spec.InitContainers),
			Containers:       trimContainers(pod.Spec.Containers),
		},
		Status: core_v1.PodStatus{
			InitContainerStatuses: trimContainerStatuses(pod.Status.InitContainerStatuses),
			ContainerStatuses:     trimContainerStatuses(pod.Status.ContainerStatuses),
		},
	}
}

func trimContainers(containers []core_v1.Container) []core_v1.Container {
	if len(containers) == 0 {
		return nil
	}
	result := make([]core_v1.Container, len(containers))
	for i, container := range containers {
		result[i] = core_v1.Container{Name: container.Name, Image: container.Image}
	}
	return result
}

func trimContainerStatuses(statuses []core_v1.ContainerStatus) []core_v1.ContainerStatus {
	if len(statuses) == 0 {
		return nil
	}
	result := make([]core_v1.ContainerStatus, len(statuses))
	for i, status := range statuses {
		result[i] = core_v1.ContainerStatus{Name: status.Name, ImageID: status.ImageID}
		if status.State.Waiting != nil {
			result[i].State.Waiting = &core_v1.ContainerStateWaiting{Reason: status.State.Waiting.Reason}
		}
	}
	return result
}
