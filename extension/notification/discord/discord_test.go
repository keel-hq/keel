package discord

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/keel-hq/keel/types"
)

func TestDiscordWebhookRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/webhooks/123/token" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			t.Errorf("content-type = %q", r.Header.Get("Content-Type"))
		}
		var payload DiscordMessage
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload.Username != "Keel" || len(payload.Embeds) != 1 {
			t.Fatalf("invalid payload: %#v", payload)
		}
		embed := payload.Embeds[0]
		if embed.Title != types.NotificationPreDeploymentUpdate.String()+": update deployment" || embed.Description != "message here" || embed.Footer.Text != "debug" {
			t.Errorf("invalid embed: %#v", embed)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	s := &sender{endpoint: server.URL + "/api/webhooks/123/token", client: server.Client()}
	if err := s.Send(types.EventNotification{
		Name: "update deployment", Message: "message here",
		Type: types.NotificationPreDeploymentUpdate, Level: types.LevelDebug,
	}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
}

func TestDiscordWebhookStatusHandling(t *testing.T) {
	for _, tc := range []struct {
		status  int
		wantErr bool
	}{{http.StatusOK, false}, {http.StatusNoContent, false}, {http.StatusAccepted, true}, {http.StatusBadRequest, true}} {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
			}))
			defer server.Close()
			err := (&sender{endpoint: server.URL, client: server.Client()}).Send(types.EventNotification{Message: "message"})
			if (err != nil) != tc.wantErr {
				t.Fatalf("Send() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
