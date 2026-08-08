package hipchat

import (
	"reflect"
	"testing"

	"github.com/keel-hq/keel/extension/notification"
	"github.com/keel-hq/keel/pkg/config"
)

func TestConfigureUsesTypedConfig(t *testing.T) {
	t.Setenv("HIPCHAT_TOKEN", "environment")
	s := &sender{}
	cfg := config.HipchatNotificationConfig{Server: "https://typed.example/api/", Token: "typed-token", BotName: "typed-bot", Channels: "ops,deployments"}
	enabled, err := s.Configure(&notification.Config{Application: config.Config{Notifications: config.NotificationConfig{Hipchat: cfg}}})
	if err != nil || !enabled {
		t.Fatalf("Configure() = %v, %v", enabled, err)
	}
	if s.botName != "typed-bot" || !reflect.DeepEqual(s.channels, []string{"ops", "deployments"}) || s.hipchatClient == nil || s.hipchatClient.BaseURL.String() != cfg.Server {
		t.Fatalf("typed fields not mapped: %#v", s)
	}
}

func TestConfigureDisabledDefaultAndInvalidServer(t *testing.T) {
	if enabled, err := (&sender{}).Configure(&notification.Config{}); err != nil || enabled {
		t.Fatalf("empty config = %v, %v", enabled, err)
	}
	s := &sender{}
	enabled, err := s.Configure(&notification.Config{Application: config.Config{Notifications: config.NotificationConfig{Hipchat: config.HipchatNotificationConfig{Token: "typed", BotName: "keel"}}}})
	if err != nil || !enabled || !reflect.DeepEqual(s.channels, []string{"general"}) {
		t.Fatalf("default channel not applied: %#v, %v", s, err)
	}
	enabled, err = (&sender{}).Configure(&notification.Config{Application: config.Config{Notifications: config.NotificationConfig{Hipchat: config.HipchatNotificationConfig{Token: "typed", Server: "://bad"}}}})
	if err == nil || enabled {
		t.Fatalf("invalid server = %v, %v", enabled, err)
	}
}
