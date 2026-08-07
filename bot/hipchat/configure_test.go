package hipchat

import (
	"testing"

	"github.com/keel-hq/keel/bot"
	"github.com/keel-hq/keel/pkg/config"
)

func TestConfigureUsesTypedConfigurationWithoutConnecting(t *testing.T) {
	t.Setenv("HIPCHAT_APPROVALS_USER_NAME", "environment-user")
	t.Setenv("HIPCHAT_APPROVALS_PASSWORT", "environment-password")
	responses := make(chan *bot.ApprovalResponse)
	messages := make(chan *bot.BotMessage)
	b := &Bot{}
	cfg := config.Config{Bots: config.BotConfig{Hipchat: config.HipchatBotConfig{ApprovalsUserName: "typed-user", ApprovalsPassword: "typed-password", ApprovalsBotName: "typed-bot", ApprovalsChannel: "typed-channel", ConnectionAttempts: 0}}}
	if !b.Configure(cfg, responses, messages) {
		t.Fatal("Configure() disabled valid typed configuration")
	}
	if b.userName != "typed-user" || b.password != "typed-password" || b.name != "typed-bot" || b.approvalsChannel != "typed-channel" {
		t.Fatalf("typed fields not mapped: %#v", b)
	}
	if b.hipchatClient != nil {
		t.Fatal("zero attempts should not make an external connection")
	}
	if b.approvalsRespCh != responses || b.botMessagesChannel != messages {
		t.Fatal("channels not mapped")
	}
}

func TestConfigureDisabledAndDefaults(t *testing.T) {
	for _, cfg := range []config.HipchatBotConfig{{}, {ApprovalsUserName: "user"}, {ApprovalsPassword: "password"}} {
		if (&Bot{}).Configure(config.Config{Bots: config.BotConfig{Hipchat: cfg}}, nil, nil) {
			t.Fatalf("Configure() enabled incomplete config %#v", cfg)
		}
	}
	b := &Bot{}
	cfg := config.Config{Bots: config.BotConfig{Hipchat: config.HipchatBotConfig{ApprovalsUserName: "user", ApprovalsPassword: "password", ApprovalsBotName: "keel", ApprovalsChannel: "general"}}}
	if !b.Configure(cfg, nil, nil) || b.name != "keel" || b.approvalsChannel != "general" {
		t.Fatalf("defaults not honored: %#v", b)
	}
}
