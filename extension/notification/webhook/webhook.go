package webhook

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
	client   *http.Client
}

// Config represents the configuration of a Webhook Sender.
type Config struct {
	Endpoint string
}

func init() {
	notification.RegisterSender("webhook", &sender{})
}

func (s *sender) Configure(config *notification.Config) (bool, error) {
	// Get configuration
	var httpConfig Config

	if os.Getenv(constants.WebhookEndpointEnv) != "" {
		httpConfig.Endpoint = os.Getenv(constants.WebhookEndpointEnv)
	} else {
		return false, nil
	}

	// Validate endpoint URL.
	if httpConfig.Endpoint == "" {
		return false, nil
	}
	if _, err := url.ParseRequestURI(httpConfig.Endpoint); err != nil {
		return false, fmt.Errorf("could not parse endpoint URL: %s\n", err)
	}
	s.endpoint = httpConfig.Endpoint

	// Setup HTTP client.
	transport := &http.Transport{}
	s.client = &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	log.WithFields(log.Fields{
		"name":     "webhook",
		"endpoint": s.endpoint,
	}).Info("extension.notification.webhook: sender configured")

	return true, nil
}

type notificationEnvelope struct {
	types.EventNotification
}

func (s *sender) Send(event types.EventNotification) error {
	// Marshal notification.
	jsonNotification, err := json.Marshal(notificationEnvelope{event})
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
