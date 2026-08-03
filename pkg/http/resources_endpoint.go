package http

import (
	"net/http"

	"github.com/keel-hq/keel/internal/k8s"
	"github.com/keel-hq/keel/internal/policy"

	"github.com/keel-hq/keel/provider/kubernetes"
)

// ResourceResponse is the resource representation returned by the admin API.
type ResourceResponse struct {
	Provider    string            `json:"provider"`
	Identifier  string            `json:"identifier"`
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Kind        string            `json:"kind"`
	Policy      string            `json:"policy"`
	Images      []string          `json:"images"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Status      k8s.Status        `json:"status"`
}

// resourcesHandler lists Kubernetes resources known to Keel.
// @Summary List resources
// @Description Returns monitored Kubernetes resources, or JSON null when the source slice is nil. This route exists only when the authenticator is enabled.
// @Tags Admin
// @ID listResources
// @Produce json
// @Security BasicAuth
// @Security BearerAuth
// @Success 200 {array} ResourceResponse
// @Failure 401 {string} string "Unauthorized"
// @Router /v1/resources [get]
func (s *TriggerServer) resourcesHandler(resp http.ResponseWriter, req *http.Request) {

	vals := s.grc.Values()

	var res []ResourceResponse

	for _, v := range vals {

		p := policy.GetPolicyFromLabelsOrAnnotations(v.GetLabels(), v.GetAnnotations())
		filterFunc := kubernetes.GetMonitorContainersFromMeta(v.GetLabels(), v.GetAnnotations())

		res = append(res, ResourceResponse{
			Provider:    "kubernetes",
			Identifier:  v.Identifier,
			Name:        v.Name,
			Namespace:   v.Namespace,
			Kind:        v.Kind(),
			Policy:      p.Name(),
			Labels:      v.GetLabels(),
			Annotations: v.GetAnnotations(),
			Images:      v.GetImages(filterFunc),
			Status:      v.GetStatus(),
		})
	}

	response(res, 200, nil, resp, req)
}
