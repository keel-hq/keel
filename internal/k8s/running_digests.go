package k8s

import (
	"sort"
	"strings"

	"github.com/opencontainers/go-digest"
	core_v1 "k8s.io/api/core/v1"
)

// PodLister supplies the pods belonging to a workload.
type PodLister interface {
	Pods(namespace, labelSelector string) (*core_v1.PodList, error)
}

// RunningDigestResolver reports the image digests that a workload is actually
// running, as opposed to the images requested by its pod template.
type RunningDigestResolver struct {
	pods PodLister
}

// NewRunningDigestResolver creates a resolver backed by the Kubernetes pod API.
func NewRunningDigestResolver(pods PodLister) *RunningDigestResolver {
	return &RunningDigestResolver{pods: pods}
}

// Resolve returns the running digests keyed by the container image reference
// they were pulled for. Images without an observable digest are omitted, so
// callers must treat a missing entry as "unknown" rather than "no drift".
func (r *RunningDigestResolver) Resolve(resource *GenericResource) map[string][]string {
	if r == nil || r.pods == nil || resource == nil {
		return nil
	}
	selector, ok := resource.GetPodSelector()
	if !ok {
		return nil
	}
	podList, err := r.pods.Pods(resource.Namespace, selector)
	if err != nil || podList == nil {
		return nil
	}

	digests := make(map[string]map[string]struct{})
	for i := range podList.Items {
		pod := &podList.Items[i]
		// pods on their way out tell us nothing about what should be running
		if pod.DeletionTimestamp != nil {
			continue
		}
		images := podContainerImages(pod)
		statuses := append(append([]core_v1.ContainerStatus(nil), pod.Status.InitContainerStatuses...), pod.Status.ContainerStatuses...)
		for _, status := range statuses {
			image, known := images[status.Name]
			if !known {
				continue
			}
			// a waiting container has not pulled anything yet, its ImageID is
			// either empty or left over from a previous run
			if status.State.Waiting != nil {
				continue
			}
			id := parseImageID(status.ImageID)
			if id == "" {
				continue
			}
			if digests[image] == nil {
				digests[image] = make(map[string]struct{})
			}
			digests[image][id] = struct{}{}
		}
	}

	if len(digests) == 0 {
		return nil
	}

	result := make(map[string][]string, len(digests))
	for image, set := range digests {
		values := make([]string, 0, len(set))
		for value := range set {
			values = append(values, value)
		}
		sort.Strings(values)
		result[image] = values
	}
	return result
}

func podContainerImages(pod *core_v1.Pod) map[string]string {
	images := make(map[string]string, len(pod.Spec.Containers)+len(pod.Spec.InitContainers))
	for _, container := range pod.Spec.InitContainers {
		images[container.Name] = container.Image
	}
	for _, container := range pod.Spec.Containers {
		images[container.Name] = container.Image
	}
	return images
}

// parseImageID normalizes the runtime specific forms of ContainerStatus.ImageID
// (docker-pullable://repo@sha256:..., repo@sha256:..., sha256:...) into a bare
// digest. It returns an empty string when no digest can be recovered, which is
// the case for images identified only by a legacy docker image ID.
func parseImageID(imageID string) string {
	if imageID == "" {
		return ""
	}
	if idx := strings.Index(imageID, "://"); idx != -1 {
		imageID = imageID[idx+len("://"):]
	}
	if idx := strings.LastIndex(imageID, "@"); idx != -1 {
		imageID = imageID[idx+1:]
	}
	parsed, err := digest.Parse(imageID)
	if err != nil {
		return ""
	}
	return parsed.String()
}
