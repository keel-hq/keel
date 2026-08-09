package discord

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

// Config represents the configuration of a Discord Webhook Sender.
type Config struct {
	Endpoint string
}

func init() {
	notification.RegisterSender("discord", &sender{})
}

func (s *sender) Configure(config *notification.Config) (bool, error) {
	// Get configuration
	var httpConfig Config

	httpConfig.Endpoint = config.Notifications.Discord.WebhookURL
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

// Discord execute-webhook API: https://discord.com/developers/docs/resources/webhook#execute-webhook
// At least one of content, embeds, components, file, or poll is required; this payload uses embeds.
type DiscordMessage struct {
	Username string  `json:"username"`
	Content  string  `json:"content"`
	Embeds   []Embed `json:"embeds"`
}

type Embed struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Footer      Footer `json:"footer"`
}

type Footer struct {
	Text string `json:"text"`
}

func (s *sender) Send(event types.EventNotification) error {
	discordMessage := DiscordMessage{
		Username: "Keel",
		Embeds: []Embed{
			{
				Title:       fmt.Sprintf("%s: %s", event.Type.String(), event.Name),
				Description: event.Message,
				Footer:      Footer{Text: event.Level.String()},
			},
		},
	}

	jsonMessage, err := json.Marshal(discordMessage)
	if err != nil {
		return fmt.Errorf("could not marshal: %s", err)
	}

	resp, err := s.client.Post(s.endpoint, "application/json", bytes.NewBuffer(jsonMessage))
	if err != nil {
		return err
	}
	if resp == nil {
		return fmt.Errorf("discord webhook returned no response")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("got status %d, expected 200/204", resp.StatusCode)
	}

	return nil
}
