package mattermost

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/keel-hq/keel/constants"
	"github.com/keel-hq/keel/extension/notification"
	"github.com/keel-hq/keel/types"

	log "github.com/sirupsen/logrus"
)

const timeout = 5 * time.Second

type sender struct {
	endpoint string
	name     string
	client   *http.Client
}

// Config represents the configuration of a Webhook Sender.
type Config struct {
	Endpoint string
	Name     string
}

func init() {
	notification.RegisterSender("mattermost", &sender{})
}

func (s *sender) Configure(config *notification.Config) (bool, error) {
	// name in the notifications
	s.name = "keel"
	// Get configuration
	var httpConfig Config

	if os.Getenv(constants.EnvMattermostEndpoint) != "" {
		httpConfig.Endpoint = os.Getenv(constants.EnvMattermostEndpoint)
	} else {
		return false, nil
	}

	if os.Getenv(constants.EnvMattermostName) != "" {
		httpConfig.Name = os.Getenv(constants.EnvMattermostName)
	}

	// Validate endpoint URL.
	if httpConfig.Endpoint == "" {
		return false, nil
	}

	if httpConfig.Name != "" {
		s.name = httpConfig.Name // setting default name
	}
	if _, err := url.ParseRequestURI(httpConfig.Endpoint); err != nil {
		log.WithFields(log.Fields{
			"endpoint": httpConfig.Endpoint,
			"error":    err,
		}).Error("extension.notification.mattermost: endpoint invalid")
		return false, fmt.Errorf("could not parse endpoint URL: %s", err)
	}
	s.endpoint = httpConfig.Endpoint

	// Setup HTTP client.
	s.client = &http.Client{
		Transport:     http.DefaultTransport,
		Timeout:       timeout,
		CheckRedirect: rejectRedirect,
	}

	log.WithFields(log.Fields{
		"name":     "mattermost",
		"endpoint": s.endpoint,
	}).Info("extension.notification.mattermost: sender configured")

	return true, nil
}

type notificationEnvelope struct {
	Username string `json:"username"`
	IconURL  string `json:"icon_url"`
	Text     string `json:"text"`
}

func (s *sender) Send(event types.EventNotification) error {
	// Marshal notification.
	jsonNotification, err := json.Marshal(notificationEnvelope{
		IconURL:  constants.KeelLogoURL,
		Username: s.name,
		Text:     fmt.Sprintf("#### %s \n %s", event.Type.String(), event.Message),
	})
	if err != nil {
		return fmt.Errorf("could not marshal: %s", err)
	}

	// Send notification via HTTP POST.
	resp, err := s.client.Post(s.endpoint, "application/json", bytes.NewBuffer(jsonNotification))
	if err != nil {
		return fmt.Errorf("could not send Mattermost notification: %w", err)
	}
	if resp == nil {
		return fmt.Errorf("could not send Mattermost notification: empty HTTP response")
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("got HTTP status %s, expected 2xx", resp.Status)
	}

	return nil
}

func rejectRedirect(_ *http.Request, _ []*http.Request) error {
	return http.ErrUseLastResponse
}
