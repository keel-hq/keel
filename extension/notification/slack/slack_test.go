package slack

import (
	"encoding/json"
	"errors"
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
		if r.Form.Get("channel") != "deployments" || r.Form.Get("text") != "" {
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

type recordingSlackClient struct {
	channels []string
	errors   map[string]error
}

func (c *recordingSlackClient) PostMessage(channel string, _ ...slackapi.MsgOption) (string, string, error) {
	c.channels = append(c.channels, channel)
	return channel, "", c.errors[channel]
}

func TestSlackSendDoesNotRetrySuccessfulChannelsAfterPartialFailure(t *testing.T) {
	client := &recordingSlackClient{errors: map[string]error{"broken": errors.New("channel not found")}}
	s := &sender{slackClient: client, channels: []string{"working", "broken"}}

	if err := s.Send(types.EventNotification{Message: "message"}); err != nil {
		t.Fatalf("Send() returned partial failure and would retry successful channels: %v", err)
	}
	if strings.Join(client.channels, ",") != "working,broken" {
		t.Fatalf("channels called = %v", client.channels)
	}
}

func TestSlackSendReturnsErrorWhenEveryChannelFails(t *testing.T) {
	client := &recordingSlackClient{errors: map[string]error{
		"one": errors.New("first failure"),
		"two": errors.New("second failure"),
	}}
	s := &sender{slackClient: client, channels: []string{"one", "two"}}

	if err := s.Send(types.EventNotification{Message: "message"}); err == nil {
		t.Fatal("Send() returned nil when every Slack channel failed")
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
