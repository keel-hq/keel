package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/keel-hq/keel/types"
	slackapi "github.com/slack-go/slack"
)

func TestSlackChatPostMessageRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/chat.postMessage" {
			t.Errorf("path = %s, want /api/chat.postMessage", r.URL.Path)
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
			t.Errorf("content-type = %q, want application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.Form.Get("token") != "xoxb-test" {
			t.Errorf("token auth = %q", r.Form.Get("token"))
		}
		if r.Form.Get("channel") != "deployments" || r.Form.Get("text") != "message here" {
			t.Errorf("unexpected form payload: %#v", r.Form)
		}
		if r.Form.Get("username") != "Keel" || r.Form.Get("icon_url") == "" {
			t.Errorf("missing customization payload: %#v", r.Form)
		}
		var attachments []slackapi.Attachment
		if err := json.Unmarshal([]byte(r.Form.Get("attachments")), &attachments); err != nil {
			t.Fatalf("decode attachments: %v", err)
		}
		if len(attachments) != 1 || attachments[0].Fallback != "message here" || len(attachments[0].Fields) != 1 {
			t.Errorf("invalid attachment payload: %#v", attachments)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"channel":"deployments","ts":"1"}`))
	}))
	defer server.Close()

	s := &sender{
		slackClient: slackapi.New("xoxb-test", slackapi.OptionAPIURL(server.URL+"/api/")),
		channels:    []string{"deployments"},
		botName:     "Keel",
	}
	if err := s.Send(types.EventNotification{Message: "message here", Type: types.NotificationPreDeploymentUpdate}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
}

func TestSlackSendReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":false,"error":"invalid_auth"}`))
	}))
	defer server.Close()

	s := &sender{
		slackClient: slackapi.New("bad-token", slackapi.OptionAPIURL(server.URL+"/")),
		channels:    []string{"general"},
	}
	if err := s.Send(types.EventNotification{Message: "message"}); err == nil {
		t.Fatal("Send() swallowed Slack API error")
	}
}
