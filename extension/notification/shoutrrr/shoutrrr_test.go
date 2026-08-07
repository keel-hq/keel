package shoutrrr

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/keel-hq/keel/extension/notification"
	appconfig "github.com/keel-hq/keel/pkg/config"
	"github.com/keel-hq/keel/types"

	shoutrrrtypes "github.com/nicholas-fedor/shoutrrr/pkg/types"
)

// genericURL builds a shoutrrr generic:// URL pointing at a local test server.
// disabletls is required because generic:// defaults to HTTPS, and template=json
// makes the service post every param so the test can assert on them.
func genericURL(serverURL string) string {
	return "generic://" + strings.TrimPrefix(serverURL, "http://") + "?disabletls=yes&template=json"
}

func testEvent() types.EventNotification {
	return types.EventNotification{
		Name:         "update deployment",
		Message:      "message here",
		CreatedAt:    time.Now(),
		Type:         types.NotificationPreDeploymentUpdate,
		Level:        types.LevelSuccess,
		ResourceKind: "deployment",
		Identifier:   "default/whatever",
		Metadata:     map[string]string{"zebra": "last", "alpha": "first"},
	}
}

// TestSendDeliversNotification exercises the whole path: env configuration, router
// construction and delivery through a real shoutrrr service.
func TestSendDeliversNotification(t *testing.T) {
	received := make(chan map[string]string, 1)

	ts := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Errorf("failed to read body: %s", err)
			return
		}

		payload := make(map[string]string)
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Errorf("failed to parse body %q: %s", string(body), err)
			return
		}

		received <- payload
	}))
	defer ts.Close()

	testURLs := genericURL(ts.URL)

	s := &sender{}

	testTimeout := "10s"
	configured, err := s.Configure(&notification.Config{Application: appconfig.Config{Notifications: appconfig.NotificationConfig{Shoutrrr: appconfig.ShoutrrrConfig{URLs: testURLs, Timeout: testTimeout}}}})
	if err != nil {
		t.Fatalf("unexpected configure error: %s", err)
	}
	if !configured {
		t.Fatal("expected sender to be configured")
	}

	if err := s.Send(testEvent()); err != nil {
		t.Fatalf("unexpected send error: %s", err)
	}

	select {
	case payload := <-received:
		if !strings.Contains(payload["title"], types.NotificationPreDeploymentUpdate.String()) {
			t.Errorf("title missing notification type: %q", payload["title"])
		}
		if !strings.Contains(payload["title"], "update deployment") {
			t.Errorf("title missing event name: %q", payload["title"])
		}

		// LevelSuccess has no shoutrrr equivalent and folds into Info.
		if payload["level"] != shoutrrrtypes.Info.String() {
			t.Errorf("got level %q, want %q", payload["level"], shoutrrrtypes.Info.String())
		}

		message := payload["message"]
		for _, want := range []string{"message here", "deployment", "default/whatever", "success", "alpha: first"} {
			if !strings.Contains(message, want) {
				t.Errorf("message missing %q, got: %q", want, message)
			}
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for notification")
	}
}

// TestSendPartialFailure covers the case where one target is down. Send must not
// report an error, because Keel retries the whole sender and would re-deliver to
// the target that already succeeded.
func TestSendPartialFailure(t *testing.T) {
	delivered := make(chan struct{}, 1)

	ts := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		delivered <- struct{}{}
	}))
	defer ts.Close()

	// Port 1 is not listening, so this target fails at send time rather than at
	// configuration time.
	testURLs := genericURL(ts.URL) + "\ngeneric://127.0.0.1:1/hook?disabletls=yes"
	testTimeout := "2s"

	s := &sender{}
	if _, err := s.Configure(&notification.Config{Application: appconfig.Config{Notifications: appconfig.NotificationConfig{Shoutrrr: appconfig.ShoutrrrConfig{URLs: testURLs, Timeout: testTimeout}}}}); err != nil {
		t.Fatalf("unexpected configure error: %s", err)
	}

	if len(s.services) != 2 {
		t.Fatalf("expected 2 configured services, got %d", len(s.services))
	}

	if err := s.Send(testEvent()); err != nil {
		t.Errorf("expected no error on partial failure, got: %s", err)
	}

	select {
	case <-delivered:
	case <-time.After(5 * time.Second):
		t.Fatal("healthy target did not receive the notification")
	}
}

// TestSendTotalFailure is the inverse: when nothing gets through, Keel should retry.
func TestSendTotalFailure(t *testing.T) {
	testURLs := "generic://127.0.0.1:1/hook?disabletls=yes"
	testTimeout := "2s"

	s := &sender{}
	if _, err := s.Configure(&notification.Config{Application: appconfig.Config{Notifications: appconfig.NotificationConfig{Shoutrrr: appconfig.ShoutrrrConfig{URLs: testURLs, Timeout: testTimeout}}}}); err != nil {
		t.Fatalf("unexpected configure error: %s", err)
	}

	if err := s.Send(testEvent()); err == nil {
		t.Error("expected an error when every target fails")
	}
}

func TestConfigureDisabledWhenUnset(t *testing.T) {
	testURLs := ""

	s := &sender{}

	testTimeout := "10s"
	configured, err := s.Configure(&notification.Config{Application: appconfig.Config{Notifications: appconfig.NotificationConfig{Shoutrrr: appconfig.ShoutrrrConfig{URLs: testURLs, Timeout: testTimeout}}}})
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if configured {
		t.Error("expected sender to stay disabled when no URLs are set")
	}
}

// A single bad URL must not disable the working ones.
func TestConfigureSkipsInvalidURL(t *testing.T) {
	testURLs := "notaservice://nope discord://token@channel"

	s := &sender{}

	testTimeout := "10s"
	configured, err := s.Configure(&notification.Config{Application: appconfig.Config{Notifications: appconfig.NotificationConfig{Shoutrrr: appconfig.ShoutrrrConfig{URLs: testURLs, Timeout: testTimeout}}}})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !configured {
		t.Fatal("expected sender to be configured from the remaining valid URL")
	}

	if len(s.services) != 1 {
		t.Fatalf("expected 1 usable service, got %d: %v", len(s.services), s.services)
	}
}

func TestConfigureFailsWhenNoURLUsable(t *testing.T) {
	testURLs := "notaservice://nope"

	s := &sender{}

	testTimeout := "10s"
	configured, err := s.Configure(&notification.Config{Application: appconfig.Config{Notifications: appconfig.NotificationConfig{Shoutrrr: appconfig.ShoutrrrConfig{URLs: testURLs, Timeout: testTimeout}}}})
	if configured {
		t.Error("expected sender to be disabled")
	}
	if err == nil {
		t.Error("expected an error explaining that no URL was usable")
	}
}

func TestConfigureRejectsBadTimeout(t *testing.T) {
	testURLs := "discord://token@channel"
	testTimeout := "soon"

	s := &sender{}

	if configured, err := s.Configure(&notification.Config{Application: appconfig.Config{Notifications: appconfig.NotificationConfig{Shoutrrr: appconfig.ShoutrrrConfig{URLs: testURLs, Timeout: testTimeout}}}}); configured || err == nil {
		t.Errorf("expected configure to fail, got configured=%v err=%v", configured, err)
	}
}

func TestSplitURLs(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{"empty", "", nil},
		{"whitespace only", "   \n\t ", nil},
		{"single", "ntfy://ntfy.sh/keel", []string{"ntfy://ntfy.sh/keel"}},
		{
			"space separated",
			"ntfy://ntfy.sh/keel discord://token@channel",
			[]string{"ntfy://ntfy.sh/keel", "discord://token@channel"},
		},
		{
			"newline separated with padding",
			"  ntfy://ntfy.sh/keel \n\n discord://token@channel  \n",
			[]string{"ntfy://ntfy.sh/keel", "discord://token@channel"},
		},
		{
			// The reason commas are not separators: shoutrrr uses them inside the query.
			"comma inside url is preserved",
			"telegram://token@telegram?chats=111,222",
			[]string{"telegram://token@telegram?chats=111,222"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitURLs(tt.raw)

			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("index %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// Service URLs carry bot tokens, API keys and SMTP passwords. Nothing secret may
// survive redaction, since the result is written to logs.
func TestRedactRemovesSecrets(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		want    string
		secrets []string
	}{
		{
			name:    "userinfo token",
			rawURL:  "discord://sup3rs3cret@123456",
			want:    "discord://123456/***",
			secrets: []string{"sup3rs3cret"},
		},
		{
			name:    "smtp password",
			rawURL:  "smtp://user:hunter2@mail.example.com:587/?from=a@b.c",
			want:    "smtp://mail.example.com/***",
			secrets: []string{"hunter2", "user"},
		},
		{
			name:    "token in query",
			rawURL:  "generic://example.com/hook?token=abc123",
			want:    "generic://example.com/***",
			secrets: []string{"abc123"},
		},
		{
			name:    "token in path",
			rawURL:  "gotify://gotify.example.com/AzyoeNS.D-A_yolo",
			want:    "gotify://gotify.example.com/***",
			secrets: []string{"AzyoeNS.D-A_yolo"},
		},
		{
			name:   "not a url",
			rawURL: "://////nonsense",
			want:   "<unparsable service url>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redact(tt.rawURL)

			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}

			for _, secret := range tt.secrets {
				if strings.Contains(got, secret) {
					t.Errorf("redacted URL %q leaks secret %q", got, secret)
				}
			}
		})
	}
}

func TestParseTimeout(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    time.Duration
		wantErr bool
	}{
		{"default when unset", "", defaultTimeout, false},
		{"default when blank", "  ", defaultTimeout, false},
		{"valid duration", "30s", 30 * time.Second, false},
		{"unparsable", "soon", 0, true},
		{"zero rejected", "0s", 0, true},
		{"negative rejected", "-5s", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTimeout(tt.raw)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error for %q", tt.raw)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestMessageLevel(t *testing.T) {
	tests := []struct {
		level types.Level
		want  shoutrrrtypes.MessageLevel
	}{
		{types.LevelDebug, shoutrrrtypes.Debug},
		{types.LevelInfo, shoutrrrtypes.Info},
		{types.LevelSuccess, shoutrrrtypes.Info},
		{types.LevelWarn, shoutrrrtypes.Warning},
		{types.LevelError, shoutrrrtypes.Error},
		{types.LevelFatal, shoutrrrtypes.Error},
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			if got := messageLevel(tt.level); got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

// Metadata ordering must be stable so that identical events render identically.
func TestBodyIsDeterministic(t *testing.T) {
	event := testEvent()
	event.Metadata = map[string]string{"c": "3", "a": "1", "b": "2"}

	first := body(event)

	for i := 0; i < 20; i++ {
		if got := body(event); got != first {
			t.Fatalf("body is not deterministic:\n%q\n%q", first, got)
		}
	}

	if !strings.Contains(first, "a: 1\nb: 2\nc: 3") {
		t.Errorf("metadata is not sorted, got: %q", first)
	}
}
