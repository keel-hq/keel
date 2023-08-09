package discord

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/keel-hq/keel/types"
)

func TestDiscordWebhookRequest(t *testing.T) {
	currentTime := time.Now()
	handler := func(resp http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Errorf("failed to parse body: %s", err)
		}

		bodyStr := string(body)

		if !strings.Contains(bodyStr, types.NotificationPreDeploymentUpdate.String()) {
			t.Errorf("missing deployment type")
		}

		if !strings.Contains(bodyStr, "debug") {
			t.Errorf("missing level")
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
	}

	s.Send(types.EventNotification{
		Name:      "update deployment",
		Message:   "message here",
		CreatedAt: currentTime,
		Type:      types.NotificationPreDeploymentUpdate,
		Level:     types.LevelDebug,
	})
}
