package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

var unsetenv = os.Unsetenv

var configurationEnvironment = []string{
	"DEBUG", "PUBSUB", "POLL", "PROJECT_ID", "CLUSTER_NAME", "XDG_DATA_HOME", "HELM3_PROVIDER", "UI_DIR",
	"NOTIFICATION_LEVEL", "WEBHOOK_ENDPOINT", "SLACK_BOT_TOKEN", "SLACK_APP_TOKEN", "SLACK_BOT_NAME", "SLACK_CHANNELS", "SLACK_APPROVALS_CHANNEL",
	"HIPCHAT_SERVER", "HIPCHAT_TOKEN", "HIPCHAT_BOT_NAME", "HIPCHAT_CHANNELS", "HIPCHAT_APPROVALS_CHANNEL", "HIPCHAT_APPROVALS_USER_NAME",
	"HIPCHAT_APPROVALS_BOT_NAME", "HIPCHAT_APPROVALS_PASSWORT", "HIPCHAT_CONNECTION_ATTEMPTS", "MATTERMOST_ENDPOINT", "MATTERMOST_USERNAME",
	"TEAMS_WEBHOOK_URL", "DISCORD_WEBHOOK_URL", "SHOUTRRR_URLS", "SHOUTRRR_TIMEOUT", "MAIL_TO", "MAIL_FROM", "MAIL_SMTP_SERVER",
	"MAIL_SMTP_PORT", "MAIL_SMTP_USER", "MAIL_SMTP_PASS", "BASIC_AUTH_USER", "BASIC_AUTH_PASSWORD", "AUTHENTICATED_WEBHOOKS",
	"TOKEN_SECRET", "AUTH_MODE", "AUTH_PROXY_USER_HEADER", "AUTH_PROXY_LOGOUT_URL", "RESTRICTED_NAMESPACE",
}

func clearConfigurationEnvironment(t *testing.T) {
	t.Helper()
	for _, name := range configurationEnvironment {
		t.Setenv(name, "")
		require.NoError(t, unsetenv(name))
	}
}

func TestLoadDefaults(t *testing.T) {
	clearConfigurationEnvironment(t)
	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, Config{
		Trigger: TriggerConfig{Poll: true}, Storage: StorageConfig{DataDir: "/data"}, UI: UIConfig{Dir: "www"},
		Notifications: NotificationConfig{
			Level: "info", Slack: SlackNotificationConfig{BotName: "keel"}, Hipchat: HipchatNotificationConfig{BotName: "keel"},
			Mattermost: MattermostConfig{Username: "keel"}, Shoutrrr: ShoutrrrConfig{Timeout: "10s"}, Mail: MailConfig{SMTPPort: 25},
		},
		Bots: BotConfig{Slack: SlackBotConfig{BotName: "keel", ApprovalsChannel: "general"}, Hipchat: HipchatBotConfig{ApprovalsChannel: "general", ApprovalsBotName: "keel", ConnectionAttempts: 10}},
	}, cfg)
}

func TestLoadMapsEveryTypedPath(t *testing.T) {
	clearConfigurationEnvironment(t)
	values := map[string]string{
		"DEBUG": "true", "PUBSUB": "true", "POLL": "false", "PROJECT_ID": "project", "CLUSTER_NAME": "cluster", "XDG_DATA_HOME": "/var/lib/keel", "HELM3_PROVIDER": "true", "UI_DIR": "/ui",
		"NOTIFICATION_LEVEL": "warn", "WEBHOOK_ENDPOINT": "https://webhook", "SLACK_BOT_TOKEN": "xoxb-typed", "SLACK_APP_TOKEN": "xapp-typed", "SLACK_BOT_NAME": "typed-bot", "SLACK_CHANNELS": "one,two", "SLACK_APPROVALS_CHANNEL": "approvals",
		"HIPCHAT_SERVER": "https://hipchat", "HIPCHAT_TOKEN": "hip-token", "HIPCHAT_BOT_NAME": "hip-notifier", "HIPCHAT_CHANNELS": "ops,dev", "HIPCHAT_APPROVALS_CHANNEL": "hip-approvals", "HIPCHAT_APPROVALS_USER_NAME": "hip-user", "HIPCHAT_APPROVALS_BOT_NAME": "hip-bot", "HIPCHAT_APPROVALS_PASSWORT": "hip-pass", "HIPCHAT_CONNECTION_ATTEMPTS": "4",
		"MATTERMOST_ENDPOINT": "https://mattermost", "MATTERMOST_USERNAME": "matter-bot", "TEAMS_WEBHOOK_URL": "https://teams", "DISCORD_WEBHOOK_URL": "https://discord", "SHOUTRRR_URLS": "discord://token@id", "SHOUTRRR_TIMEOUT": "3s",
		"MAIL_TO": "to@example.com", "MAIL_FROM": "from@example.com", "MAIL_SMTP_SERVER": "smtp.example.com", "MAIL_SMTP_PORT": "2525", "MAIL_SMTP_USER": "smtp-user", "MAIL_SMTP_PASS": "smtp-pass",
		"BASIC_AUTH_USER": "admin", "BASIC_AUTH_PASSWORD": "secret", "AUTHENTICATED_WEBHOOKS": "true", "TOKEN_SECRET": "token-secret", "AUTH_MODE": "proxy", "AUTH_PROXY_USER_HEADER": "X-User", "AUTH_PROXY_LOGOUT_URL": "https://logout", "RESTRICTED_NAMESPACE": "production",
	}
	for key, value := range values {
		t.Setenv(key, value)
	}
	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, Config{
		Debug: true, Trigger: TriggerConfig{PubSub: true, ProjectID: "project", ClusterName: "cluster"}, Storage: StorageConfig{DataDir: "/var/lib/keel"}, Providers: ProviderConfig{Helm3: true}, UI: UIConfig{Dir: "/ui"},
		Notifications: NotificationConfig{Level: "warn", Webhook: WebhookConfig{Endpoint: "https://webhook"}, Slack: SlackNotificationConfig{BotToken: "xoxb-typed", BotName: "typed-bot", Channels: "one,two"}, Hipchat: HipchatNotificationConfig{Server: "https://hipchat", Token: "hip-token", BotName: "hip-notifier", Channels: "ops,dev"}, Mattermost: MattermostConfig{Endpoint: "https://mattermost", Username: "matter-bot"}, Teams: TeamsConfig{WebhookURL: "https://teams"}, Discord: DiscordConfig{WebhookURL: "https://discord"}, Shoutrrr: ShoutrrrConfig{URLs: "discord://token@id", Timeout: "3s"}, Mail: MailConfig{To: "to@example.com", From: "from@example.com", SMTPServer: "smtp.example.com", SMTPPort: 2525, SMTPUser: "smtp-user", SMTPPass: "smtp-pass"}},
		Bots:          BotConfig{Slack: SlackBotConfig{BotToken: "xoxb-typed", AppToken: "xapp-typed", BotName: "typed-bot", ApprovalsChannel: "approvals"}, Hipchat: HipchatBotConfig{ApprovalsChannel: "hip-approvals", ApprovalsUserName: "hip-user", ApprovalsBotName: "hip-bot", ApprovalsPassword: "hip-pass", ConnectionAttempts: 4}},
		Auth:          AuthConfig{BasicUser: "admin", BasicPassword: "secret", AuthenticatedWebhooks: true, TokenSecret: "token-secret", Mode: "proxy", ProxyUserHeader: "X-User", ProxyLogoutURL: "https://logout"}, Kubernetes: KubernetesConfig{RestrictedNamespace: "production"},
	}, cfg)
}

func TestLoadRejectsInvalidTypedValues(t *testing.T) {
	for _, tt := range []struct{ name, key, value string }{{"boolean", "POLL", "sometimes"}, {"integer", "MAIL_SMTP_PORT", "smtp"}, {"nested integer", "HIPCHAT_CONNECTION_ATTEMPTS", "many"}} {
		t.Run(tt.name, func(t *testing.T) {
			clearConfigurationEnvironment(t)
			t.Setenv(tt.key, tt.value)
			_, err := Load()
			require.Error(t, err)
		})
	}
}
