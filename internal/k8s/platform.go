package k8s

import (
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/keel-hq/keel/types"
	"github.com/sirupsen/logrus"
	core_v1 "k8s.io/api/core/v1"
)

const (
	nodeCacheTTL         = 30 * time.Second
	nodeErrorLogThrottle = 30 * time.Second
	nodeRBACRemediation  = "grant Keel's service account get, list, and watch access to the core/v1 nodes resource"
)

// NodeLister supplies Kubernetes node scheduling and platform metadata.
type NodeLister interface {
	Nodes() (*core_v1.NodeList, error)
}

type podLister interface {
	Pods(namespace, labelSelector string) (*core_v1.PodList, error)
}

// PlatformResolver derives a conservative eligible platform set for workloads.
type PlatformResolver struct {
	nodes NodeLister
	log   logrus.FieldLogger

	mu                      sync.Mutex
	cached                  []core_v1.Node
	expiresAt               time.Time
	lastNodeListError       string
	lastNodeListErrorLogged time.Time
}

// NewPlatformResolver creates a resolver backed by Kubernetes Node metadata.
func NewPlatformResolver(nodes NodeLister) *PlatformResolver {
	return &PlatformResolver{nodes: nodes, log: logrus.StandardLogger()}
}

// Resolve returns every platform on which the workload may be scheduled.
func (r *PlatformResolver) Resolve(resource *GenericResource) ([]types.Platform, types.PlatformError) {
	if resource == nil || resource.GetPodSpec() == nil {
		return nil, types.PlatformErrorWorkloadMetadata
	}
	nodes, err := r.listNodes()
	if err != nil {
		return nil, types.PlatformErrorNodeMetadata
	}

	platforms := make(map[types.Platform]struct{})
	nodesByName := make(map[string]*core_v1.Node, len(nodes))
	for i := range nodes {
		node := &nodes[i]
		nodesByName[node.Name] = node
		if node.Spec.Unschedulable || !nodeMatchesPod(node, resource.GetPodSpec()) {
			continue
		}
		platform, ok := nodePlatform(node)
		if !ok {
			return nil, types.PlatformErrorNodeMetadata
		}
		platforms[platform] = struct{}{}
	}
	if pods, ok := r.nodes.(podLister); ok {
		if selector, hasSelector := resource.GetPodSelector(); hasSelector {
			podList, err := pods.Pods(resource.Namespace, selector)
			if err == nil && podList != nil {
				for i := range podList.Items {
					pod := &podList.Items[i]
					if pod.Spec.NodeName == "" {
						continue
					}
					node := nodesByName[pod.Spec.NodeName]
					platform, found := nodePlatform(node)
					if !found {
						return nil, types.PlatformErrorNodeMetadata
					}
					platforms[platform] = struct{}{}
				}
			}
		}
	}
	if len(platforms) == 0 {
		return nil, types.PlatformErrorNoEligibleNodes
	}

	result := make([]types.Platform, 0, len(platforms))
	for platform := range platforms {
		result = append(result, platform)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].OS != result[j].OS {
			return result[i].OS < result[j].OS
		}
		if result[i].Architecture != result[j].Architecture {
			return result[i].Architecture < result[j].Architecture
		}
		return result[i].Variant < result[j].Variant
	})
	return result, types.PlatformErrorNone
}

func (r *PlatformResolver) listNodes() ([]core_v1.Node, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if time.Now().Before(r.expiresAt) {
		return append([]core_v1.Node(nil), r.cached...), nil
	}
	list, err := r.nodes.Nodes()
	if err != nil {
		r.logNodeListFailure(err)
		return nil, err
	}
	r.lastNodeListError = ""
	r.lastNodeListErrorLogged = time.Time{}
	if list == nil {
		return nil, fmt.Errorf("node list is nil")
	}
	r.cached = append(r.cached[:0], list.Items...)
	r.expiresAt = time.Now().Add(nodeCacheTTL)
	return append([]core_v1.Node(nil), r.cached...), nil
}

// logNodeListFailure emits the Kubernetes API error with an actionable RBAC
// hint. Node resolution runs once per workload, so identical errors are
// throttled to avoid flooding the logs when a cluster contains many workloads.
// r.mu must be held by the caller.
func (r *PlatformResolver) logNodeListFailure(err error) {
	message := err.Error()
	now := time.Now()
	if message == r.lastNodeListError && now.Sub(r.lastNodeListErrorLogged) < nodeErrorLogThrottle {
		return
	}
	r.lastNodeListError = message
	r.lastNodeListErrorLogged = now
	r.log.WithFields(logrus.Fields{
		"error":       err,
		"remediation": nodeRBACRemediation,
	}).Warn("platform resolver: failed to list Kubernetes nodes; workload updates are blocked because platform compatibility cannot be verified")
}

func nodePlatform(node *core_v1.Node) (types.Platform, bool) {
	if node == nil {
		return types.Platform{}, false
	}
	os := node.Status.NodeInfo.OperatingSystem
	architecture := node.Status.NodeInfo.Architecture
	if os == "" {
		os = firstLabel(node.Labels, core_v1.LabelOSStable, "beta.kubernetes.io/os")
	}
	if architecture == "" {
		architecture = firstLabel(node.Labels, core_v1.LabelArchStable, "beta.kubernetes.io/arch")
	}
	if os == "" || architecture == "" {
		return types.Platform{}, false
	}
	return types.Platform{OS: os, Architecture: architecture}, true
}

func firstLabel(labels map[string]string, keys ...string) string {
	for _, key := range keys {
		if labels[key] != "" {
			return labels[key]
		}
	}
	return ""
}

func nodeMatchesPod(node *core_v1.Node, pod *core_v1.PodSpec) bool {
	if pod.NodeName != "" && pod.NodeName != node.Name {
		return false
	}
	for key, value := range pod.NodeSelector {
		if node.Labels[key] != value {
			return false
		}
	}
	if pod.OS != nil && pod.OS.Name != "" {
		if platform, ok := nodePlatform(node); !ok || platform.OS != string(pod.OS.Name) {
			return false
		}
	}

	required := pod.Affinity
	if required == nil || required.NodeAffinity == nil || required.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		return true
	}
	terms := required.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
	for _, term := range terms {
		if nodeMatchesTerm(node, term) {
			return true
		}
	}
	return false
}

func nodeMatchesTerm(node *core_v1.Node, term core_v1.NodeSelectorTerm) bool {
	for _, expression := range term.MatchExpressions {
		if !matchesRequirement(node.Labels[expression.Key], expression.Operator, expression.Values) {
			return false
		}
	}
	for _, expression := range term.MatchFields {
		if expression.Key != "metadata.name" {
			continue
		}
		if !matchesRequirement(node.Name, expression.Operator, expression.Values) {
			return false
		}
	}
	return true
}

func matchesRequirement(value string, operator core_v1.NodeSelectorOperator, values []string) bool {
	switch operator {
	case core_v1.NodeSelectorOpIn:
		return contains(values, value)
	case core_v1.NodeSelectorOpNotIn:
		return !contains(values, value)
	case core_v1.NodeSelectorOpExists:
		return value != ""
	case core_v1.NodeSelectorOpDoesNotExist:
		return value == ""
	case core_v1.NodeSelectorOpGt, core_v1.NodeSelectorOpLt:
		if len(values) != 1 {
			return true
		}
		actual, actualErr := strconv.Atoi(value)
		expected, expectedErr := strconv.Atoi(values[0])
		if actualErr != nil || expectedErr != nil {
			return true
		}
		if operator == core_v1.NodeSelectorOpGt {
			return actual > expected
		}
		return actual < expected
	default:
		return true
	}
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
