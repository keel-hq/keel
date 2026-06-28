package webhook

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/keel-hq/keel/types"
)

func TestWebhookRequest(t *testing.T) {
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

func TestWebhookAccepts2xxStatusCodes(t *testing.T) {
	for _, code := range []int{200, 201, 202, 204} {
		t.Run(fmt.Sprintf("status_%d", code), func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
			}))
			defer ts.Close()

			s := &sender{
				endpoint: ts.URL,
				client:   &http.Client{},
			}

			err := s.Send(types.EventNotification{
				Name:    "test",
				Message: "test",
				Type:    types.NotificationPreDeploymentUpdate,
				Level:   types.LevelDebug,
			})
			if err != nil {
				t.Errorf("expected status %d to be accepted, got error: %v", code, err)
			}
		})
	}
}

func TestWebhookRejectsNon2xxStatusCodes(t *testing.T) {
	for _, code := range []int{400, 500} {
		t.Run(fmt.Sprintf("status_%d", code), func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
			}))
			defer ts.Close()

			s := &sender{
				endpoint: ts.URL,
				client:   &http.Client{},
			}

			err := s.Send(types.EventNotification{
				Name:    "test",
				Message: "test",
				Type:    types.NotificationPreDeploymentUpdate,
				Level:   types.LevelDebug,
			})
			if err == nil {
				t.Errorf("expected status %d to be rejected, got no error", code)
			}
		})
	}
}
