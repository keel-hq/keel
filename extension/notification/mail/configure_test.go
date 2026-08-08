package mail

import (
	"testing"

	"github.com/keel-hq/keel/extension/notification"
	"github.com/keel-hq/keel/pkg/config"
)

func TestConfigureUsesTypedConfig(t *testing.T) {
	t.Setenv("MAIL_SMTP_SERVER", "environment")
	s := &sender{}
	cfg := config.MailConfig{SMTPServer: "typed.smtp", SMTPPort: 2525, From: "from@typed", To: "to@typed", SMTPUser: "user", SMTPPass: "pass"}
	enabled, err := s.Configure(&notification.Config{Application: config.Config{Notifications: config.NotificationConfig{Mail: cfg}}})
	if err != nil || !enabled {
		t.Fatalf("Configure() = %v, %v", enabled, err)
	}
	if s.smtpServer != cfg.SMTPServer || s.smtpPort != cfg.SMTPPort || s.from != cfg.From || s.to != cfg.To || s.smtpUser != cfg.SMTPUser || s.smtpPass != cfg.SMTPPass {
		t.Fatalf("typed fields not mapped: %#v", s)
	}
}

func TestConfigureRequiresServerFromAndTo(t *testing.T) {
	for _, cfg := range []config.MailConfig{{}, {SMTPServer: "smtp", From: "from"}, {SMTPServer: "smtp", To: "to"}, {From: "from", To: "to"}} {
		enabled, err := (&sender{}).Configure(&notification.Config{Application: config.Config{Notifications: config.NotificationConfig{Mail: cfg}}})
		if err != nil || enabled {
			t.Fatalf("incomplete config %#v = %v, %v", cfg, enabled, err)
		}
	}
}
