package teams

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/keel-hq/keel/constants"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/version"
)

func TestTrimFirstChar(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Empty string", "", ""},
		{"Single ASCII char", "H", ""},
		{"Single unicode char", "世", ""},
		{"ASCII string", "Hello", "ello"},
		{"Unicode string", "世界", "界"},
		{"Hex color", "#FF0000", "FF0000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TrimFirstChar(tt.input)
			if result != tt.expected {
				t.Errorf("TrimFirstChar(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDetectWebhookType(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		expected string
	}{
		{
			name:     "Power Automate workflow URL",
			endpoint: "https://prod-12.westus.logic.azure.com:443/workflows/abc123/triggers/manual/paths/invoke",
			expected: "adaptive",
		},
		{
			name:     "Legacy Office 365 connector URL (outlook)",
			endpoint: "https://outlook.office.com/webhook/abc123/IncomingWebhook/def456",
			expected: "messagecard",
		},
		{
			name:     "Legacy Office 365 connector URL (webhook)",
			endpoint: "https://webhook.office.com/webhookb2/abc123/def456",
			expected: "messagecard",
		},
		{
			name:     "Unknown URL defaults to adaptive",
			endpoint: "https://example.com/webhook",
			expected: "adaptive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &sender{}
			result := s.detectWebhookType(tt.endpoint)
			if result != tt.expected {
				t.Errorf("detectWebhookType() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTeamsMessageCardRequest(t *testing.T) {
	handler := func(resp http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Errorf("failed to parse body: %s", err)
		}

		bodyStr := string(body)

		if !strings.Contains(bodyStr, "MessageCard") {
			t.Errorf("missing MessageCard indicator")
		}

		if !strings.Contains(bodyStr, "themeColor") {
			t.Errorf("missing themeColor")
		}

		if !strings.Contains(bodyStr, constants.KeelLogoURL) {
			t.Errorf("missing logo url")
		}

		if !strings.Contains(bodyStr, "**"+types.NotificationPreDeploymentUpdate.String()+"**") {
			t.Errorf("missing deployment type")
		}

		if !strings.Contains(bodyStr, version.GetKeelVersion().Version) {
			t.Errorf("missing version")
		}

		if !strings.Contains(bodyStr, "update deployment") {
			t.Errorf("missing name")
		}
		if !strings.Contains(bodyStr, "message here") {
			t.Errorf("missing message")
		}

		t.Log(bodyStr)
	}

	// create test server with handler
	ts := httptest.NewServer(http.HandlerFunc(handler))
	defer ts.Close()

	s := &sender{
		endpoint: ts.URL,
		client:   &http.Client{},
		format:   "messagecard", // Force MessageCard format for this test
	}

	s.Send(types.EventNotification{
		Name:    "update deployment",
		Message: "message here",
		Type:    types.NotificationPreDeploymentUpdate,
	})
}

func TestTeamsAdaptiveCardRequest(t *testing.T) {
	handler := func(resp http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Errorf("failed to parse body: %s", err)
		}

		bodyStr := string(body)

		if !strings.Contains(bodyStr, "AdaptiveCard") {
			t.Errorf("missing AdaptiveCard indicator")
		}

		if !strings.Contains(bodyStr, "application/vnd.microsoft.card.adaptive") {
			t.Errorf("missing adaptive card content type")
		}

		if !strings.Contains(bodyStr, constants.KeelLogoURL) {
			t.Errorf("missing logo url")
		}

		if !strings.Contains(bodyStr, types.NotificationPreDeploymentUpdate.String()) {
			t.Errorf("missing deployment type")
		}

		if !strings.Contains(bodyStr, version.GetKeelVersion().Version) {
			t.Errorf("missing version")
		}

		if !strings.Contains(bodyStr, "update deployment") {
			t.Errorf("missing name")
		}
		if !strings.Contains(bodyStr, "message here") {
			t.Errorf("missing message")
		}

		t.Log(bodyStr)
	}

	// create test server with handler
	ts := httptest.NewServer(http.HandlerFunc(handler))
	defer ts.Close()

	s := &sender{
		endpoint: ts.URL,
		client:   &http.Client{},
		format:   "adaptive", // Force Adaptive Card format for this test
	}

	s.Send(types.EventNotification{
		Name:    "update deployment",
		Message: "message here",
		Type:    types.NotificationPreDeploymentUpdate,
	})
}

func TestLevelToAdaptiveColor(t *testing.T) {
	tests := []struct {
		name     string
		level    types.Level
		expected string
	}{
		{"Error level", types.LevelError, "Attention"},
		{"Warning level", types.LevelWarn, "Warning"},
		{"Success level", types.LevelSuccess, "Good"},
		{"Info level", types.LevelInfo, "Accent"},
		{"Debug level", types.LevelDebug, "Default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &sender{}
			result := s.levelToAdaptiveColor(tt.level)
			if result != tt.expected {
				t.Errorf("levelToAdaptiveColor() = %v, want %v", result, tt.expected)
			}
		})
	}
}
