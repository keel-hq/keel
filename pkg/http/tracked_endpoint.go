package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/keel-hq/keel/types"
)

// TrackedImage is the polling configuration returned for a tracked image.
type TrackedImage struct {
	Image        string `json:"image"`
	Trigger      string `json:"trigger"`
	PollSchedule string `json:"pollSchedule"`
	Provider     string `json:"provider"`
	Namespace    string `json:"namespace"`
	Policy       string `json:"policy"`
	Registry     string `json:"registry"`
}

// trackedHandler lists tracked images.
// @Summary List tracked images
// @Description Returns image polling configuration, or JSON null when the source slice is nil. This route exists only when the authenticator is enabled.
// @Tags Admin
// @ID listTrackedImages
// @Produce json
// @Security BasicAuth
// @Security BearerAuth
// @Success 200 {array} TrackedImage
// @Failure 401 {string} string "Unauthorized"
// @Failure 403 {string} string "Permission denied"
// @Failure 500 {string} string "Provider lookup failed"
// @Router /v1/tracked [get]
func (s *TriggerServer) trackedHandler(resp http.ResponseWriter, req *http.Request) {
	trackedImages, err := s.providers.TrackedImages()

	var imgs []TrackedImage

	for _, img := range trackedImages {
		imgs = append(imgs, TrackedImage{
			Image:        img.Image.Name(),
			Trigger:      img.Trigger.String(),
			PollSchedule: img.PollSchedule,
			Provider:     img.Provider,
			Namespace:    img.Namespace,
			Policy:       img.Policy.Name(),
			Registry:     img.Image.Registry(),
		})
	}

	response(&imgs, 200, err, resp, req)
}

// TrackRequest changes the trigger and polling schedule for a resource.
type TrackRequest struct {
	Provider   string `json:"provider"`
	Identifier string `json:"identifier"`
	Trigger    string `json:"trigger"`
	Schedule   string `json:"schedule"`
}

// trackSetHandler changes tracking for a resource.
// @Summary Update image tracking
// @Description Sets the trigger and polling schedule for a Kubernetes resource. This route exists only when the authenticator is enabled.
// @Tags Admin
// @ID updateTrackedImage
// @Accept json
// @Produce json
// @Security BasicAuth
// @Security BearerAuth
// @Param body body TrackRequest true "Tracking update"
// @Success 200 {object} APIResponse
// @Failure 400 {string} string "Malformed or unsupported request"
// @Failure 401 {string} string "Unauthorized"
// @Failure 403 {string} string "Permission denied"
// @Failure 404 {string} string "Resource not found"
// @Failure 500 {string} string "Update failed"
// @Router /v1/tracked [put]
func (s *TriggerServer) trackSetHandler(resp http.ResponseWriter, req *http.Request) {

	var trackReq TrackRequest
	dec := json.NewDecoder(req.Body)
	defer req.Body.Close()

	err := dec.Decode(&trackReq)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(resp, "%s", err)
		return
	}

	switch trackReq.Provider {
	case types.ProviderTypeKubernetes.String():
		// ok
	default:
		http.Error(resp, "unsupported provider, supported: 'kubernetes'", http.StatusBadRequest)
		return
	}

	switch trackReq.Trigger {
	case "default", "poll":
		// ok
	default:
		http.Error(resp, "unknown trigger type, supported: 'default', 'poll'", http.StatusBadRequest)
		return
	}

	if trackReq.Schedule != "" {
		_, err = time.ParseDuration(trackReq.Schedule)
		if err != nil {
			resp.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(resp, "%s", err)
			return
		}
	} else {
		trackReq.Schedule = types.KeelPollDefaultSchedule
	}

	for _, v := range s.grc.Values() {
		if v.Identifier == trackReq.Identifier {

			labels := v.GetLabels()
			delete(labels, types.KeelTriggerLabel)
			v.SetLabels(labels)

			ann := v.GetAnnotations()
			ann[types.KeelTriggerLabel] = trackReq.Trigger
			ann[types.KeelPollScheduleAnnotation] = trackReq.Schedule

			v.SetAnnotations(ann)

			err := s.kubernetesClient.Update(v)

			response(&APIResponse{Status: "updated"}, 200, err, resp, req)
			return
		}
	}

	resp.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(resp, "resource with identifier '%s' not found", trackReq.Identifier)
}
