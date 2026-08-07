package config

import "github.com/kelseyhightower/envconfig"

// Config contains Keel's application configuration loaded from environment variables.
type Config struct {
	Debug         bool `envconfig:"DEBUG" default:"false"`
	Trigger       TriggerConfig
	Storage       StorageConfig
	Providers     ProviderConfig
	UI            UIConfig
	Notifications NotificationConfig
	Bots          BotConfig
	Auth          AuthConfig
	Kubernetes    KubernetesConfig
}

// TriggerConfig controls the event sources that detect and initiate image updates.
type TriggerConfig struct {
	PubSub      bool   `envconfig:"PUBSUB" default:"false"`
	Poll        bool   `envconfig:"POLL" default:"true"`
	ProjectID   string `envconfig:"PROJECT_ID"`
	ClusterName string `envconfig:"CLUSTER_NAME"`
}

// StorageConfig controls where Keel stores its persistent application data.
type StorageConfig struct {
	DataDir string `envconfig:"XDG_DATA_HOME" default:"/data"`
}

// ProviderConfig controls which workload update providers Keel enables.
type ProviderConfig struct {
	Helm3 bool `envconfig:"HELM3_PROVIDER" default:"false"`
}

// UIConfig controls where the HTTP server finds the web UI static files.
type UIConfig struct {
	Dir string `envconfig:"UI_DIR" default:"www"`
}

// NotificationConfig controls the minimum notification level and notification delivery integrations.
type NotificationConfig struct {
	Level      string `envconfig:"NOTIFICATION_LEVEL" default:"info"`
	Webhook    WebhookConfig
	Slack      SlackNotificationConfig
	Hipchat    HipchatNotificationConfig
	Mattermost MattermostConfig
	Teams      TeamsConfig
	Discord    DiscordConfig
	Shoutrrr   ShoutrrrConfig
	Mail       MailConfig
}

// WebhookConfig configures notifications sent to a generic webhook endpoint.
type WebhookConfig struct {
	Endpoint string `envconfig:"WEBHOOK_ENDPOINT"`
}

// SlackNotificationConfig configures outbound update notifications sent through Slack.
type SlackNotificationConfig struct {
	BotToken string `envconfig:"SLACK_BOT_TOKEN"`
	BotName  string `envconfig:"SLACK_BOT_NAME" default:"keel"`
	Channels string `envconfig:"SLACK_CHANNELS"`
}

// HipchatNotificationConfig configures outbound update notifications sent through HipChat.
type HipchatNotificationConfig struct {
	Server   string `envconfig:"HIPCHAT_SERVER"`
	Token    string `envconfig:"HIPCHAT_TOKEN"`
	BotName  string `envconfig:"HIPCHAT_BOT_NAME" default:"keel"`
	Channels string `envconfig:"HIPCHAT_CHANNELS"`
}

// MattermostConfig configures outbound notifications sent to a Mattermost webhook.
type MattermostConfig struct {
	Endpoint string `envconfig:"MATTERMOST_ENDPOINT"`
	Username string `envconfig:"MATTERMOST_USERNAME" default:"keel"`
}

// TeamsConfig configures outbound notifications sent to a Microsoft Teams webhook.
type TeamsConfig struct {
	WebhookURL string `envconfig:"TEAMS_WEBHOOK_URL"`
}

// DiscordConfig configures outbound notifications sent to a Discord webhook.
type DiscordConfig struct {
	WebhookURL string `envconfig:"DISCORD_WEBHOOK_URL"`
}

// ShoutrrrConfig configures outbound notifications delivered through Shoutrrr services.
type ShoutrrrConfig struct {
	URLs    string `envconfig:"SHOUTRRR_URLS"`
	Timeout string `envconfig:"SHOUTRRR_TIMEOUT" default:"10s"`
}

// MailConfig configures outbound email notifications and their SMTP connection.
type MailConfig struct {
	To         string `envconfig:"MAIL_TO"`
	From       string `envconfig:"MAIL_FROM"`
	SMTPServer string `envconfig:"MAIL_SMTP_SERVER"`
	SMTPPort   int    `envconfig:"MAIL_SMTP_PORT" default:"25"`
	SMTPUser   string `envconfig:"MAIL_SMTP_USER"`
	SMTPPass   string `envconfig:"MAIL_SMTP_PASS"`
}

// BotConfig groups interactive chat bots used to request and process update approvals.
type BotConfig struct {
	Slack   SlackBotConfig
	Hipchat HipchatBotConfig
}

// SlackBotConfig configures the Slack approvals bot and its target channel.
type SlackBotConfig struct {
	BotToken         string `envconfig:"SLACK_BOT_TOKEN"`
	AppToken         string `envconfig:"SLACK_APP_TOKEN"`
	BotName          string `envconfig:"SLACK_BOT_NAME" default:"keel"`
	ApprovalsChannel string `envconfig:"SLACK_APPROVALS_CHANNEL" default:"general"`
}

// HipchatBotConfig configures the HipChat approvals bot, credentials, and connection behavior.
type HipchatBotConfig struct {
	ApprovalsChannel   string `envconfig:"HIPCHAT_APPROVALS_CHANNEL" default:"general"`
	ApprovalsUserName  string `envconfig:"HIPCHAT_APPROVALS_USER_NAME"`
	ApprovalsBotName   string `envconfig:"HIPCHAT_APPROVALS_BOT_NAME" default:"keel"`
	ApprovalsPassword  string `envconfig:"HIPCHAT_APPROVALS_PASSWORT"`
	ConnectionAttempts int    `envconfig:"HIPCHAT_CONNECTION_ATTEMPTS" default:"10"`
}

// AuthConfig controls administrator authentication, webhook authentication, and external proxy integration.
type AuthConfig struct {
	BasicUser             string `envconfig:"BASIC_AUTH_USER"`
	BasicPassword         string `envconfig:"BASIC_AUTH_PASSWORD"`
	AuthenticatedWebhooks bool   `envconfig:"AUTHENTICATED_WEBHOOKS" default:"false"`
	TokenSecret           string `envconfig:"TOKEN_SECRET"`
	Mode                  string `envconfig:"AUTH_MODE"`
	ProxyUserHeader       string `envconfig:"AUTH_PROXY_USER_HEADER"`
	ProxyLogoutURL        string `envconfig:"AUTH_PROXY_LOGOUT_URL"`
}

// KubernetesConfig controls the scope of Kubernetes resources watched by Keel.
type KubernetesConfig struct {
	RestrictedNamespace string `envconfig:"RESTRICTED_NAMESPACE"`
}

// Load reads configuration from environment variables.
func Load() (Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
