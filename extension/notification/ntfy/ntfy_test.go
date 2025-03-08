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

func TestNtfyWebhookRequest(t *testing.T) {
	currentTime := time.Now()
	handler := func(resp http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Errorf("failed to parse body: %s", err)
		}

		bodyStr := string(body)

		if !strings.Contains(req.Header.Get("Title"), "message here") {
			t.Errorf("missing message")
		}

		if !strings.Contains(req.Header.Get("Tags"), "keel") {
			t.Errorf("missing deployment type")
		}

		if !strings.Contains(bodyStr, "update deployment") {
			t.Errorf("missing update deployment")
		}

		if !strings.Contains(bodyStr, types.NotificationPreDeploymentUpdate.String()) {
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
