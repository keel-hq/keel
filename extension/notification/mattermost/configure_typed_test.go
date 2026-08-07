package mattermost

import (
	"testing"

	"github.com/keel-hq/keel/extension/notification"
	"github.com/keel-hq/keel/pkg/config"
)

func TestConfigureTypedConfigEnabledDisabled(t *testing.T) {
	t.Setenv("MATTERMOST_ENDPOINT", "https://environment.invalid")
	s := &sender{}
	cfg := config.MattermostConfig{Endpoint: "https://typed.example/hook", Username: "typed-name"}
	enabled, err := s.Configure(&notification.Config{Application: config.Config{Notifications: config.NotificationConfig{Mattermost: cfg}}})
	if err != nil || !enabled || s.endpoint != cfg.Endpoint || s.name != cfg.Username || s.client == nil {
		t.Fatalf("typed config not used: enabled=%v err=%v sender=%#v", enabled, err, s)
	}
	if enabled, err = (&sender{}).Configure(&notification.Config{}); err != nil || enabled {
		t.Fatalf("empty config = %v, %v", enabled, err)
	}
	if enabled, err = (&sender{}).Configure(&notification.Config{Application: config.Config{Notifications: config.NotificationConfig{Mattermost: config.MattermostConfig{Endpoint: "://bad"}}}}); err == nil || enabled {
		t.Fatalf("invalid config = %v, %v", enabled, err)
	}
}
