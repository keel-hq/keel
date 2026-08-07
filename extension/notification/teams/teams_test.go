package teams

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/keel-hq/keel/constants"
	"github.com/keel-hq/keel/types"
)

func TestTrimFirstChar(t *testing.T) {
	for _, tc := range []struct{ input, want string }{{"", ""}, {"H", ""}, {"#123456", "123456"}, {"世界", "界"}} {
		if got := TrimFirstChar(tc.input); got != tc.want {
			t.Errorf("TrimFirstChar(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestTeamsRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/workflow/hook" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			t.Errorf("content-type = %q", r.Header.Get("Content-Type"))
		}
		var payload SimpleTeamsMessageCard
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload.AtType != "MessageCard" || payload.AtContext != "http://schema.org/extensions" || payload.Summary != types.NotificationPreDeploymentUpdate.String() {
			t.Errorf("invalid card envelope: %#v", payload)
		}
		if len(payload.Sections) != 1 {
			t.Fatalf("sections = %#v", payload.Sections)
		}
		section := payload.Sections[0]
		if section.ActivityImage != constants.KeelLogoURL || section.ActivityText != "*update deployment*: message here" || !section.Markdown || len(section.Facts) != 1 {
			t.Errorf("invalid section: %#v", section)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	s := &sender{endpoint: server.URL + "/workflow/hook", client: server.Client()}
	if err := s.Send(types.EventNotification{
		Name: "update deployment", Message: "message here",
		Type: types.NotificationPreDeploymentUpdate, Level: types.LevelDebug,
	}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
}

func TestTeamsStatusHandling(t *testing.T) {
	for _, tc := range []struct {
		status  int
		wantErr bool
	}{{http.StatusOK, false}, {http.StatusCreated, false}, {http.StatusAccepted, false}, {http.StatusNoContent, false}, {http.StatusMultipleChoices, true}, {http.StatusBadRequest, true}} {
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
