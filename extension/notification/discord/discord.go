package discord

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

// Config represents the configuration of a Discord Webhook Sender.
type Config struct {
	Endpoint string
}

func init() {
	log.Error("RUNNING")
	notification.RegisterSender("discord", &sender{})
}

func (s *sender) Configure(config *notification.Config) (bool, error) {
	// Get configuration
	var httpConfig Config

	if os.Getenv(constants.EnvDiscordWebhookUrl) != "" {
		httpConfig.Endpoint = os.Getenv(constants.EnvDiscordWebhookUrl)
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

	// Setup HTTP client.
	s.client = &http.Client{
		Transport: http.DefaultTransport,
		Timeout:   timeout,
	}

	log.WithFields(log.Fields{
		"name":     "discord",
		"endpoint": s.endpoint,
	}).Info("extension.notification.discord: sender configured")

	return true, nil
}

// type notificationEnvelope struct {
// 	types.EventNotification
// }

type DiscordMessage struct {
	Content  string `json:"content"`
	Username string `json:"username"`
}

func (s *sender) Send(event types.EventNotification) error {
	discordMessage := DiscordMessage{
		Content:  fmt.Sprintf("**%s**\n%s", event.Name, event.Message),
		Username: "Keel",
	}

	jsonMessage, err := json.Marshal(discordMessage)
	if err != nil {
		return fmt.Errorf("could not marshal: %s", err)
	}

	resp, err := s.client.Post(s.endpoint, "application/json", bytes.NewBuffer(jsonMessage))
	if err != nil || resp == nil || (resp.StatusCode != 200 && resp.StatusCode != 204) {
		if resp != nil {
			return fmt.Errorf("got status %d, expected 200/204", resp.StatusCode)
		}
		return err
	}
	defer resp.Body.Close()

	return nil
}
