package http

import "net/http"

type trackedImage struct {
	Image        string `json:"image"`
	Trigger      string `json:"trigger"`
	PollSchedule string `json:"pollSchedule"`
	Provider     string `json:"provider"`
	Namespace    string `json:"namespace"`
	Policy       string `json:"policy"`
	Registry     string `json:"registry"`
}

func (s *TriggerServer) trackedHandler(resp http.ResponseWriter, req *http.Request) {
	trackedImages, err := s.providers.TrackedImages()

	var imgs []trackedImage

	for _, img := range trackedImages {
		imgs = append(imgs, trackedImage{
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
