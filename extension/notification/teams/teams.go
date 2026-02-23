package teams

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/keel-hq/keel/constants"
	"github.com/keel-hq/keel/extension/notification"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/version"

	log "github.com/sirupsen/logrus"
)

const timeout = 5 * time.Second

type sender struct {
	endpoint string
	client   *http.Client
	format   string
}

// Config represents the configuration of a Teams Webhook Sender.
type Config struct {
	Endpoint string
	Format   string
}

func init() {
	notification.RegisterSender("teams", &sender{})
}

func (s *sender) Configure(config *notification.Config) (bool, error) {
	// Get configuration
	var httpConfig Config

	if os.Getenv(constants.EnvTeamsWebhookUrl) != "" {
		httpConfig.Endpoint = os.Getenv(constants.EnvTeamsWebhookUrl)
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
	s.client = &http.Client{
		Transport: http.DefaultTransport,
		Timeout:   timeout,
	}

	// Determine webhook format
	s.format = s.getWebhookFormat()

	log.WithFields(log.Fields{
		"name":     "teams",
		"webhook":  s.endpoint,
		"format":   s.format,
	}).Info("extension.notification.teams: sender configured")

	return true, nil
}

type SimpleTeamsMessageCard struct {
	AtContext string `json:"@context"`
	AtType    string `json:"@type"`
	Sections []TeamsMessageSection `json:"sections"`
	Summary    string `json:"summary"`
	ThemeColor string `json:"themeColor"`
}

type TeamsMessageSection struct {
	ActivityImage    string `json:"activityImage"`
	ActivitySubtitle string `json:"activitySubtitle"`
	ActivityText     string `json:"activityText"`
	ActivityTitle    string `json:"activityTitle"`
	Facts    []TeamsFact `json:"facts"`
	Markdown bool `json:"markdown"`
}

type TeamsFact struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// AdaptiveTeamsCard represents the new Adaptive Card format for Teams
type AdaptiveTeamsCard struct {
	Type        string       `json:"type"`
	Attachments []Attachment `json:"attachments"`
}

type Attachment struct {
	ContentType string           `json:"contentType"`
	Content     AdaptiveCardBody `json:"content"`
}

type AdaptiveCardBody struct {
	Schema  string        `json:"$schema"`
	Type    string        `json:"type"`
	Version string        `json:"version"`
	Body    []interface{} `json:"body"`
}

type TextBlock struct {
	Type   string `json:"type"`
	Text   string `json:"text"`
	Weight string `json:"weight,omitempty"`
	Color  string `json:"color,omitempty"`
	Wrap   bool   `json:"wrap,omitempty"`
}

type ImageBlock struct {
	Type string `json:"type"`
	URL  string `json:"url"`
	Size string `json:"size,omitempty"`
}

type FactSet struct {
	Type  string      `json:"type"`
	Facts []AdaptiveFact `json:"facts"`
}

type AdaptiveFact struct {
	Title string `json:"title"`
	Value string `json:"value"`
}

// Microsoft Teams expects the hexidecimal formatted color to not have a "#" at the front
// Source: https://stackoverflow.com/a/48798875/2199949
func TrimFirstChar(s string) string {
    for i := range s {
        if i > 0 {
            // The value i is the index in s of the second
            // character.  Slice to remove the first character.
            return s[i:]
        }
    }
    // There are 0 or 1 characters in the string.
    return ""
}

// detectWebhookType determines the webhook format based on URL patterns
func (s *sender) detectWebhookType(endpoint string) string {
	// New Power Automate workflow URLs
	if strings.Contains(endpoint, "workflows") && strings.Contains(endpoint, "triggers/manual") {
		return "adaptive"
	}
	// Legacy Office 365 Connector URLs
	if strings.Contains(endpoint, "outlook.office.com") || strings.Contains(endpoint, "webhook.office.com") {
		return "messagecard"
	}
	// Default to adaptive for unknown patterns (future-proof)
	return "adaptive"
}

// getWebhookFormat returns the webhook format to use, checking environment variable first
func (s *sender) getWebhookFormat() string {
	if format := os.Getenv(constants.EnvTeamsWebhookFormat); format != "" {
		return format
	}
	// Fallback to URL detection
	return s.detectWebhookType(s.endpoint)
}

func (s *sender) Send(event types.EventNotification) error {
	var jsonNotification []byte
	var err error

	switch s.format {
	case "adaptive":
		jsonNotification, err = s.marshalAdaptiveCard(event)
	case "messagecard":
		jsonNotification, err = s.marshalMessageCard(event)
	default:
		return fmt.Errorf("unsupported webhook format: %s", s.format)
	}

	if err != nil {
		return fmt.Errorf("could not marshal %s: %s", s.format, err)
	}

	// Send notification via HTTP POST.
	resp, err := s.client.Post(s.endpoint, "application/json", bytes.NewBuffer(jsonNotification))
	if err != nil || resp == nil {
		return err
	}
	defer resp.Body.Close()

	// Accept 200, 201, and 202 (Power Automate returns 202 Accepted)
	if resp.StatusCode != 200 && resp.StatusCode != 201 && resp.StatusCode != 202 {
		return fmt.Errorf("got status %d, expected 200/201/202", resp.StatusCode)
	}

	return nil
}

// marshalMessageCard creates the legacy MessageCard format
func (s *sender) marshalMessageCard(event types.EventNotification) ([]byte, error) {
	return json.Marshal(SimpleTeamsMessageCard{
		AtType: "MessageCard",
		AtContext: "http://schema.org/extensions",
		ThemeColor: TrimFirstChar(event.Level.Color()),
		Summary: event.Type.String(),
		Sections: []TeamsMessageSection{
			{
				ActivityImage: constants.KeelLogoURL,
				ActivityText: fmt.Sprintf("*%s*: %s", event.Name, event.Message),
				ActivityTitle: fmt.Sprintf("**%s**", event.Type.String()),
				Facts: []TeamsFact{
					{
						Name: "Version",
						Value: fmt.Sprintf("[https://keel.sh](https://keel.sh) %s", version.GetKeelVersion().Version),
					},
				},
				Markdown: true,
			},
		},
	})
}

// marshalAdaptiveCard creates the new Adaptive Card format
func (s *sender) marshalAdaptiveCard(event types.EventNotification) ([]byte, error) {
	// Convert level color to Adaptive Card color name
	colorName := s.levelToAdaptiveColor(event.Level)

	return json.Marshal(AdaptiveTeamsCard{
		Type: "message",
		Attachments: []Attachment{
			{
				ContentType: "application/vnd.microsoft.card.adaptive",
				Content: AdaptiveCardBody{
					Schema:  "http://adaptivecards.io/schemas/adaptive-card.json",
					Type:    "AdaptiveCard",
					Version: "1.2",
					Body: []interface{}{
						TextBlock{
							Type:   "TextBlock",
							Text:   event.Type.String(),
							Weight: "Bolder",
							Color:  colorName,
							Wrap:   true,
						},
						ImageBlock{
							Type: "Image",
							URL:  constants.KeelLogoURL,
							Size: "Small",
						},
						TextBlock{
							Type: "TextBlock",
							Text: fmt.Sprintf("**%s**: %s", event.Name, event.Message),
							Wrap: true,
						},
						FactSet{
							Type: "FactSet",
							Facts: []AdaptiveFact{
								{
									Title: "Version",
									Value: fmt.Sprintf("[Keel %s](https://keel.sh)", version.GetKeelVersion().Version),
								},
							},
						},
					},
				},
			},
		},
	})
}

// levelToAdaptiveColor converts Keel levels to Adaptive Card color names
func (s *sender) levelToAdaptiveColor(level types.Level) string {
	switch level {
	case types.LevelError:
		return "Attention"
	case types.LevelWarn:
		return "Warning"
	case types.LevelSuccess:
		return "Good"
	case types.LevelInfo:
		return "Accent"
	default:
		return "Default"
	}
}
