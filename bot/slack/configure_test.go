package slack

import (
	"testing"

	"github.com/keel-hq/keel/bot"
	"github.com/keel-hq/keel/pkg/config"
)

func TestConfigureUsesTypedConfiguration(t *testing.T) {
	t.Setenv("SLACK_BOT_TOKEN", "invalid-environment")
	t.Setenv("SLACK_APP_TOKEN", "invalid-environment")
	responses := make(chan *bot.ApprovalResponse)
	messages := make(chan *bot.BotMessage)
	b := &Bot{}
	cfg := config.Config{Bots: config.BotConfig{Slack: config.SlackBotConfig{BotToken: "xoxb-typed", AppToken: "xapp-typed", BotName: "typed-name", ApprovalsChannel: "#typed-channel"}}}
	if !b.Configure(cfg, responses, messages) {
		t.Fatal("Configure() disabled a valid typed configuration")
	}
	if b.name != "typed-name" || b.approvalsChannel != "typed-channel" {
		t.Fatalf("typed fields not mapped: %#v", b)
	}
	if b.slackSocket == nil || b.approvalsRespCh != responses || b.botMessagesChannel != messages {
		t.Fatal("Configure() did not initialize the bot")
	}
}

func TestConfigureDisabledAndDefaults(t *testing.T) {
	for _, cfg := range []config.SlackBotConfig{{}, {BotToken: "xoxb-ok"}, {BotToken: "environment-token", AppToken: "xapp-ok"}} {
		b := &Bot{}
		if b.Configure(config.Config{Bots: config.BotConfig{Slack: cfg}}, nil, nil) {
			t.Fatalf("Configure() enabled invalid config %#v", cfg)
		}
	}
	b := &Bot{}
	cfg := config.Config{Bots: config.BotConfig{Slack: config.SlackBotConfig{BotToken: "xoxb-ok", AppToken: "xapp-ok", BotName: "keel", ApprovalsChannel: "general"}}}
	if !b.Configure(cfg, nil, nil) || b.name != "keel" || b.approvalsChannel != "general" {
		t.Fatalf("defaults not honored: %#v", b)
	}
}
