package discord

import (
	"testing"

	"github.com/keel-hq/keel/extension/notification"
	"github.com/keel-hq/keel/pkg/config"
)

func TestConfigureTypedConfigEnabledDisabled(t *testing.T) {
	t.Setenv("DISCORD_WEBHOOK_URL", "https://environment.invalid")
	s := &sender{}
	endpoint := "https://typed.example/hook"
	enabled, err := s.Configure(&notification.Config{Notifications: config.NotificationConfig{Discord: config.DiscordConfig{WebhookURL: endpoint}}})
	if err != nil || !enabled || s.endpoint != endpoint || s.client == nil {
		t.Fatalf("typed config not used: enabled=%v err=%v sender=%#v", enabled, err, s)
	}
	enabled, err = (&sender{}).Configure(&notification.Config{})
	if err != nil || enabled {
		t.Fatalf("empty typed config enabled sender: %v, %v", enabled, err)
	}
	enabled, err = (&sender{}).Configure(&notification.Config{Notifications: config.NotificationConfig{Discord: config.DiscordConfig{WebhookURL: "://bad"}}})
	if err == nil || enabled {
		t.Fatalf("invalid typed config = %v, %v", enabled, err)
	}
}
