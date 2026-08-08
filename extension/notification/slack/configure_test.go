package slack

import (
	"reflect"
	"testing"

	"github.com/keel-hq/keel/extension/notification"
	"github.com/keel-hq/keel/pkg/config"
)

func TestConfigureUsesTypedConfig(t *testing.T) {
	t.Setenv("SLACK_BOT_TOKEN", "environment")
	s := &sender{}
	enabled, err := s.Configure(&notification.Config{Application: config.Config{Notifications: config.NotificationConfig{Slack: config.SlackNotificationConfig{BotToken: "typed", BotName: "typed-bot", Channels: "ops,deployments"}}}})
	if err != nil || !enabled {
		t.Fatalf("Configure() = %v, %v", enabled, err)
	}
	if s.botName != "typed-bot" || !reflect.DeepEqual(s.channels, []string{"ops", "deployments"}) || s.slackClient == nil {
		t.Fatalf("typed fields not mapped: %#v", s)
	}
}

func TestConfigureDisabledAndDefaultChannel(t *testing.T) {
	if enabled, err := (&sender{}).Configure(&notification.Config{}); err != nil || enabled {
		t.Fatalf("empty config = %v, %v", enabled, err)
	}
	s := &sender{}
	enabled, err := s.Configure(&notification.Config{Application: config.Config{Notifications: config.NotificationConfig{Slack: config.SlackNotificationConfig{BotToken: "typed", BotName: "keel"}}}})
	if err != nil || !enabled || !reflect.DeepEqual(s.channels, []string{"general"}) {
		t.Fatalf("default channel not applied: %#v, %v", s, err)
	}
}
