package discord

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
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

// Config represents the configuration of a Discord Webhook Sender.
type Config struct {
	Endpoint string
}

func init() {
	notification.RegisterSender("ntfy", &sender{})
}

func (s *sender) Configure(config *notification.Config) (bool, error) {
	// Get configuration
	var httpConfig Config

	if os.Getenv(constants.EnvNtfyWebhookUrl) != "" {
		httpConfig.Endpoint = os.Getenv(constants.EnvNtfyWebhookUrl)
	} else {
		return false, nil
	}
	// Validate endpoint URL.
	if httpConfig.Endpoint == "" {
		return false, nil
	}
	if _, err := url.ParseRequestURI(httpConfig.Endpoint); err != nil {
		return false, fmt.Errorf("could not parse endpoint URL: %s", err)
	}
	s.endpoint = httpConfig.Endpoint

	// Setup HTTP client
	s.client = &http.Client{
		Transport: http.DefaultTransport,
		Timeout:   timeout,
	}

	log.WithFields(log.Fields{
		"name":     "ntfy",
		"endpoint": s.endpoint,
	}).Info("extension.notification.ntfy: sender configured")
	return true, nil
}

func (s *sender) Send(event types.EventNotification) error {

	req, _ := http.NewRequest("POST", s.endpoint, strings.NewReader(fmt.Sprintf("%s: %s", event.Type.String(), event.Name)))
	req.Header.Set("Title", event.Message)
	req.Header.Set("Tags", "keel")
	req.Header.Set("X-Icon", constants.KeelLogoURL)

	resp, err := s.client.Do(req)
	if err != nil || resp == nil || (resp.StatusCode != 200 && resp.StatusCode != 204) {
		if resp != nil {
			return fmt.Errorf("got status %d, expected 200/204", resp.StatusCode)
		}
		return err
	}
	defer resp.Body.Close()

	return nil
}
