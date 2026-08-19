package teams

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
}

// Config represents the configuration of a Teams Webhook Sender.
type Config struct {
	Endpoint string
}

func init() {
	notification.RegisterSender("teams", &sender{})
}

func (s *sender) Configure(config *notification.Config) (bool, error) {
	// Get configuration
	var httpConfig Config

	httpConfig.Endpoint = config.Notifications.Teams.WebhookURL

	// Validate endpoint URL.
	if httpConfig.Endpoint == "" {
		return false, nil
	}
	if _, err := url.ParseRequestURI(httpConfig.Endpoint); err != nil {
		// The parse error is not returned: url.ParseRequestURI echoes the
		// input in its message and the endpoint may carry a secret.
		return false, fmt.Errorf("could not parse endpoint URL: not a valid absolute URL")
	}
	s.endpoint = httpConfig.Endpoint

	// Setup HTTP client.
	s.client = &http.Client{
		Transport: http.DefaultTransport,
		Timeout:   timeout,
	}

	// The endpoint URL may embed the webhook signing secret in its path or
	// query string, so only a redacted form is ever logged.
	log.WithFields(log.Fields{
		"name":    "teams",
		"webhook": notification.SafeURL(s.endpoint),
	}).Info("extension.notification.teams: sender configured")
	if log.IsLevelEnabled(log.DebugLevel) {
		log.WithFields(log.Fields{
			"name":    "teams",
			"webhook": notification.DebugURL(s.endpoint),
		}).Debug("extension.notification.teams: sender endpoint (secrets redacted)")
	}

	return true, nil
}

// Teams incoming-webhook and Workflows payload documentation:
// https://learn.microsoft.com/en-us/microsoftteams/platform/webhooks-and-connectors/how-to/add-incoming-webhook
// Microsoft 365 connectors are nearing deprecation; Teams Workflows are recommended and accept Message Cards.
type SimpleTeamsMessageCard struct {
	AtContext  string                `json:"@context"`
	AtType     string                `json:"@type"`
	Sections   []TeamsMessageSection `json:"sections"`
	Summary    string                `json:"summary"`
	ThemeColor string                `json:"themeColor"`
}

type TeamsMessageSection struct {
	ActivityImage    string      `json:"activityImage"`
	ActivitySubtitle string      `json:"activitySubtitle"`
	ActivityText     string      `json:"activityText"`
	ActivityTitle    string      `json:"activityTitle"`
	Facts            []TeamsFact `json:"facts"`
	Markdown         bool        `json:"markdown"`
}

type TeamsFact struct {
	Name  string `json:"name"`
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

func (s *sender) Send(event types.EventNotification) error {
	// Marshal notification.
	jsonNotification, err := json.Marshal(SimpleTeamsMessageCard{
		AtType:     "MessageCard",
		AtContext:  "http://schema.org/extensions",
		ThemeColor: TrimFirstChar(event.Level.Color()),
		Summary:    event.Type.String(),
		Sections: []TeamsMessageSection{
			{
				ActivityImage: constants.KeelLogoURL,
				ActivityText:  fmt.Sprintf("*%s*: %s", event.Name, event.Message),
				ActivityTitle: fmt.Sprintf("**%s**", event.Type.String()),
				Facts: []TeamsFact{
					{
						Name:  "Version",
						Value: fmt.Sprintf("[https://keel.sh](https://keel.sh) %s", version.GetKeelVersion().Version),
					},
				},
				Markdown: true,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("could not marshal: %s", err)
	}

	// Send notification via HTTP POST.
	resp, err := s.client.Post(s.endpoint, "application/json", bytes.NewBuffer(jsonNotification))
	if err != nil {
		return err
	}
	if resp == nil {
		return fmt.Errorf("teams webhook returned no response")
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("got status %d, expected 2xx", resp.StatusCode)
	}

	return nil
}
