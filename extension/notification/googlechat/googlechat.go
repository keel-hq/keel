package googlechat

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
	notification.RegisterSender("googlechat", &sender{})
}

func (s *sender) Configure(config *notification.Config) (bool, error) {
	// name in the notifications
	s.name = "keel"
	// Get configuration
	var httpConfig Config

	if os.Getenv(constants.EnvGoogleChatEndpoint) != "" {
		httpConfig.Endpoint = os.Getenv(constants.EnvGoogleChatEndpoint)
	} else {
		return false, nil
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
		}).Error("extension.notification.googlechat: endpoint invalid")
		return false, fmt.Errorf("could not parse endpoint URL: %s", err)
	}
	s.endpoint = httpConfig.Endpoint

	// Setup HTTP client.
	s.client = &http.Client{
		Transport: http.DefaultTransport,
		Timeout:   timeout,
	}

	log.WithFields(log.Fields{
		"name":     "googlechat",
		"endpoint": s.endpoint,
	}).Info("extension.notification.googlechat: sender configured")

	return true, nil
}

type notificationEnvelope struct {
	Text     string `json:"text"`
}

func (s *sender) Send(event types.EventNotification) error {
	// Marshal notification.
	jsonNotification, err := json.Marshal(notificationEnvelope{
		Text:     fmt.Sprintf("*%s* \n %s", event.Type.String(), event.Message),
	})
	if err != nil {
		return fmt.Errorf("could not marshal: %s", err)
	}

	// Send notification via HTTP POST.
	resp, err := s.client.Post(s.endpoint, "application/json", bytes.NewBuffer(jsonNotification))
	if err != nil || resp == nil || (resp.StatusCode != 200 && resp.StatusCode != 201) {
		if resp != nil {
			return fmt.Errorf("got status %d, expected 200/201", resp.StatusCode)
		}
		return err
	}
	defer resp.Body.Close()

	return nil
}
