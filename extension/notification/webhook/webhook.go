package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

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

	httpConfig.Endpoint = config.Notifications.Webhook.Endpoint

	// Validate endpoint URL.
	if httpConfig.Endpoint == "" {
		return false, nil
	}
	if _, err := url.ParseRequestURI(httpConfig.Endpoint); err != nil {
		return false, fmt.Errorf("could not parse endpoint URL: %s\n", err)
	}
	s.endpoint = httpConfig.Endpoint

	// Setup HTTP client.
	s.client = &http.Client{
		Transport:     http.DefaultTransport,
		Timeout:       timeout,
		CheckRedirect: rejectRedirect,
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
	if err != nil {
		return fmt.Errorf("could not send webhook: %w", err)
	}
	if resp == nil {
		return fmt.Errorf("could not send webhook: empty HTTP response")
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
