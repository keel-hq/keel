package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

var unsetenv = os.Unsetenv

func TestLoadDefaults(t *testing.T) {
	for _, name := range []string{
		"PUBSUB",
		"POLL",
		"PROJECT_ID",
		"CLUSTER_NAME",
		"XDG_DATA_HOME",
		"HELM3_PROVIDER",
		"UI_DIR",
		"HIPCHAT_CONNECTION_ATTEMPTS",
		"MAIL_SMTP_PORT",
	} {
		t.Setenv(name, "")
		require.NoError(t, unsetenv(name))
	}

	cfg, err := Load()
	require.NoError(t, err)
	require.False(t, cfg.Trigger.PubSub)
	require.True(t, cfg.Trigger.Poll)
	require.Equal(t, "/data", cfg.Storage.DataDir)
	require.False(t, cfg.Providers.Helm3)
	require.Equal(t, "www", cfg.UI.Dir)
	require.Equal(t, "info", cfg.Notifications.Level)
	require.Equal(t, "keel", cfg.Notifications.Hipchat.BotName)
	require.Equal(t, "general", cfg.Bots.Hipchat.ApprovalsChannel)
	require.Equal(t, 10, cfg.Bots.Hipchat.ConnectionAttempts)
	require.Equal(t, 25, cfg.Notifications.Mail.SMTPPort)
}

func TestLoad(t *testing.T) {
	t.Setenv("PUBSUB", "true")
	t.Setenv("POLL", "false")
	t.Setenv("PROJECT_ID", "project")
	t.Setenv("CLUSTER_NAME", "cluster")
	t.Setenv("XDG_DATA_HOME", "/var/lib/keel")
	t.Setenv("HELM3_PROVIDER", "1")
	t.Setenv("UI_DIR", "/opt/keel/ui")
	t.Setenv("HIPCHAT_TOKEN", "token")
	t.Setenv("HIPCHAT_BOT_NAME", "notifier")
	t.Setenv("HIPCHAT_CHANNELS", "deployments,operations")
	t.Setenv("HIPCHAT_APPROVALS_CHANNEL", "approvals")
	t.Setenv("HIPCHAT_APPROVALS_USER_NAME", "user")
	t.Setenv("HIPCHAT_APPROVALS_BOT_NAME", "approver")
	t.Setenv("HIPCHAT_APPROVALS_PASSWORT", "secret")
	t.Setenv("HIPCHAT_CONNECTION_ATTEMPTS", "4")

	cfg, err := Load()
	require.NoError(t, err)
	require.True(t, cfg.Trigger.PubSub)
	require.False(t, cfg.Trigger.Poll)
	require.Equal(t, "project", cfg.Trigger.ProjectID)
	require.Equal(t, "cluster", cfg.Trigger.ClusterName)
	require.Equal(t, "/var/lib/keel", cfg.Storage.DataDir)
	require.True(t, cfg.Providers.Helm3)
	require.Equal(t, "/opt/keel/ui", cfg.UI.Dir)
	require.Equal(t, HipchatNotificationConfig{
		Token: "token", BotName: "notifier", Channels: "deployments,operations",
	}, cfg.Notifications.Hipchat)
	require.Equal(t, HipchatBotConfig{
		ApprovalsChannel: "approvals", ApprovalsUserName: "user", ApprovalsBotName: "approver",
		ApprovalsPassword: "secret", ConnectionAttempts: 4,
	}, cfg.Bots.Hipchat)
}
