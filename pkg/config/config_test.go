package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func clearConfigurationEnvironment(t *testing.T) {
	t.Helper()
	for _, name := range environmentVariables {
		// t.Setenv records the original value so the direct unset below remains
		// isolated to this test.
		t.Setenv(name, "")
		require.NoError(t, os.Unsetenv(name))
	}
}

func TestLoadDefaults(t *testing.T) {
	clearConfigurationEnvironment(t)
	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, Config{
		Trigger: TriggerConfig{Poll: true, PollScanInterval: time.Minute}, Storage: StorageConfig{DataDir: "/data"}, UI: UIConfig{Dir: "www"},
		Notifications: NotificationConfig{
			Level: "info", Slack: SlackNotificationConfig{BotName: "keel"}, Hipchat: HipchatNotificationConfig{BotName: "keel"},
			Mattermost: MattermostConfig{Username: "keel"}, Shoutrrr: ShoutrrrConfig{Timeout: "10s"}, Mail: MailConfig{SMTPPort: 25},
		},
		Bots: BotConfig{Slack: SlackBotConfig{BotName: "keel", ApprovalsChannel: "general"}, Hipchat: HipchatBotConfig{ApprovalsChannel: "general", ApprovalsBotName: "keel", ConnectionAttempts: 5}},
	}, cfg)
}

func TestLoadTreatsExplicitlyEmptyValuesAsUnset(t *testing.T) {
	clearConfigurationEnvironment(t)
	for _, name := range []string{"POLL", "POLL_SCAN_INTERVAL", "PUBSUB", "DEBUG", "HELM3_PROVIDER", "AUTHENTICATED_WEBHOOKS", "MAIL_SMTP_PORT", "HIPCHAT_CONNECTION_ATTEMPTS"} {
		t.Setenv(name, "")
	}

	cfg, err := Load()
	require.NoError(t, err)
	require.True(t, cfg.Trigger.Poll)
	require.Equal(t, time.Minute, cfg.Trigger.PollScanInterval)
	require.False(t, cfg.Trigger.PubSub)
	require.False(t, cfg.Debug)
	require.False(t, cfg.Providers.Helm3)
	require.False(t, cfg.Auth.AuthenticatedWebhooks)
	require.Equal(t, 25, cfg.Notifications.Mail.SMTPPort)
	require.Equal(t, 5, cfg.Bots.Hipchat.ConnectionAttempts)

	for _, name := range []string{"POLL", "POLL_SCAN_INTERVAL", "PUBSUB", "DEBUG", "HELM3_PROVIDER", "AUTHENTICATED_WEBHOOKS", "MAIL_SMTP_PORT", "HIPCHAT_CONNECTION_ATTEMPTS"} {
		value, ok := os.LookupEnv(name)
		require.True(t, ok, "%s should remain set", name)
		require.Empty(t, value, "%s should remain empty", name)
	}
}

func TestLoadPreservesLegacyPubSubValues(t *testing.T) {
	for _, tt := range []struct {
		value   string
		enabled bool
	}{
		{value: "1", enabled: true},
		{value: "true", enabled: true},
		{value: "yes", enabled: true},
		{value: "on", enabled: true},
		{value: "enabled", enabled: true},
		{value: "0", enabled: false},
		{value: "false", enabled: false},
	} {
		t.Run(tt.value, func(t *testing.T) {
			clearConfigurationEnvironment(t)
			t.Setenv("PUBSUB", tt.value)
			cfg, err := Load()
			require.NoError(t, err)
			require.Equal(t, tt.enabled, cfg.Trigger.PubSub)
		})
	}
}

func TestLoadIgnoresNestedAliasNames(t *testing.T) {
	clearConfigurationEnvironment(t)
	t.Setenv("TRIGGER_POLL", "false")
	t.Setenv("STORAGE_XDG_DATA_HOME", "/shadow")
	t.Setenv("AUTH_AUTHENTICATED_WEBHOOKS", "true")
	t.Setenv("BOTS_SLACK_SLACK_BOT_TOKEN", "shadow-token")

	cfg, err := Load()
	require.NoError(t, err)
	require.True(t, cfg.Trigger.Poll)
	require.Equal(t, "/data", cfg.Storage.DataDir)
	require.False(t, cfg.Auth.AuthenticatedWebhooks)
	require.Empty(t, cfg.Bots.Slack.BotToken)
}

func TestLoadMapsEveryTypedPath(t *testing.T) {
	clearConfigurationEnvironment(t)
	values := map[string]string{
		"DEBUG": "true", "PUBSUB": "true", "POLL": "false", "POLL_SCAN_INTERVAL": "45s", "PROJECT_ID": "project", "CLUSTER_NAME": "cluster", "XDG_DATA_HOME": "/var/lib/keel", "HELM3_PROVIDER": "true", "UI_DIR": "/ui",
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
		Debug: true, Trigger: TriggerConfig{PubSub: true, PollScanInterval: 45 * time.Second, ProjectID: "project", ClusterName: "cluster"}, Storage: StorageConfig{DataDir: "/var/lib/keel"}, Providers: ProviderConfig{Helm3: true}, UI: UIConfig{Dir: "/ui"},
		Notifications: NotificationConfig{Level: "warn", Webhook: WebhookConfig{Endpoint: "https://webhook"}, Slack: SlackNotificationConfig{BotToken: "xoxb-typed", BotName: "typed-bot", Channels: "one,two"}, Hipchat: HipchatNotificationConfig{Server: "https://hipchat", Token: "hip-token", BotName: "hip-notifier", Channels: "ops,dev"}, Mattermost: MattermostConfig{Endpoint: "https://mattermost", Username: "matter-bot"}, Teams: TeamsConfig{WebhookURL: "https://teams"}, Discord: DiscordConfig{WebhookURL: "https://discord"}, Shoutrrr: ShoutrrrConfig{URLs: "discord://token@id", Timeout: "3s"}, Mail: MailConfig{To: "to@example.com", From: "from@example.com", SMTPServer: "smtp.example.com", SMTPPort: 2525, SMTPUser: "smtp-user", SMTPPass: "smtp-pass"}},
		Bots:          BotConfig{Slack: SlackBotConfig{BotToken: "xoxb-typed", AppToken: "xapp-typed", BotName: "typed-bot", ApprovalsChannel: "approvals"}, Hipchat: HipchatBotConfig{ApprovalsChannel: "hip-approvals", ApprovalsUserName: "hip-user", ApprovalsBotName: "hip-bot", ApprovalsPassword: "hip-pass", ConnectionAttempts: 4}},
		Auth:          AuthConfig{BasicUser: "admin", BasicPassword: "secret", AuthenticatedWebhooks: true, TokenSecret: "token-secret", Mode: "proxy", ProxyUserHeader: "X-User", ProxyLogoutURL: "https://logout"}, Kubernetes: KubernetesConfig{RestrictedNamespace: "production"},
	}, cfg)
}

func TestLoadRejectsInvalidTypedValues(t *testing.T) {
	for _, tt := range []struct{ name, key, value string }{{"boolean", "POLL", "sometimes"}, {"duration", "POLL_SCAN_INTERVAL", "soon"}, {"non-positive duration", "POLL_SCAN_INTERVAL", "0s"}, {"integer", "MAIL_SMTP_PORT", "smtp"}, {"nested integer", "HIPCHAT_CONNECTION_ATTEMPTS", "many"}} {
		t.Run(tt.name, func(t *testing.T) {
			clearConfigurationEnvironment(t)
			t.Setenv(tt.key, tt.value)
			_, err := Load()
			require.Error(t, err)
		})
	}
}

func TestLoadReportsDocumentedVariableName(t *testing.T) {
	clearConfigurationEnvironment(t)
	t.Setenv("POLL", "sometimes")
	_, err := Load()
	require.Error(t, err)
	require.Contains(t, err.Error(), "POLL")
	require.NotContains(t, err.Error(), "TRIGGER_POLL")
}
