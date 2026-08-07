package config

import "github.com/kelseyhightower/envconfig"

// Config contains Keel's application configuration loaded from environment variables.
type Config struct {
	Trigger       TriggerConfig
	Storage       StorageConfig
	Providers     ProviderConfig
	UI            UIConfig
	Notifications NotificationConfig
	Bots          BotConfig
	Auth          AuthConfig
	Kubernetes    KubernetesConfig
}

type TriggerConfig struct {
	PubSub      bool   `envconfig:"PUBSUB" default:"false"`
	Poll        bool   `envconfig:"POLL" default:"true"`
	ProjectID   string `envconfig:"PROJECT_ID"`
	ClusterName string `envconfig:"CLUSTER_NAME"`
}

type StorageConfig struct {
	DataDir string `envconfig:"XDG_DATA_HOME" default:"/data"`
}

type ProviderConfig struct {
	Helm3 bool `envconfig:"HELM3_PROVIDER" default:"false"`
}

type UIConfig struct {
	Dir string `envconfig:"UI_DIR" default:"www"`
}

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

type WebhookConfig struct {
	Endpoint string `envconfig:"WEBHOOK_ENDPOINT"`
}

type SlackNotificationConfig struct {
	BotToken string `envconfig:"SLACK_BOT_TOKEN"`
	BotName  string `envconfig:"SLACK_BOT_NAME" default:"keel"`
	Channels string `envconfig:"SLACK_CHANNELS"`
}

type HipchatNotificationConfig struct {
	Token    string `envconfig:"HIPCHAT_TOKEN"`
	BotName  string `envconfig:"HIPCHAT_BOT_NAME" default:"keel"`
	Channels string `envconfig:"HIPCHAT_CHANNELS"`
}

type MattermostConfig struct {
	Endpoint string `envconfig:"MATTERMOST_ENDPOINT"`
	Username string `envconfig:"MATTERMOST_USERNAME" default:"keel"`
}

type TeamsConfig struct {
	WebhookURL string `envconfig:"TEAMS_WEBHOOK_URL"`
}

type DiscordConfig struct {
	WebhookURL string `envconfig:"DISCORD_WEBHOOK_URL"`
}

type ShoutrrrConfig struct {
	URLs    string `envconfig:"SHOUTRRR_URLS"`
	Timeout string `envconfig:"SHOUTRRR_TIMEOUT" default:"10s"`
}

type MailConfig struct {
	To         string `envconfig:"MAIL_TO"`
	From       string `envconfig:"MAIL_FROM"`
	SMTPServer string `envconfig:"MAIL_SMTP_SERVER"`
	SMTPPort   int    `envconfig:"MAIL_SMTP_PORT" default:"25"`
	SMTPUser   string `envconfig:"MAIL_SMTP_USER"`
	SMTPPass   string `envconfig:"MAIL_SMTP_PASS"`
}

type BotConfig struct {
	Slack   SlackBotConfig
	Hipchat HipchatBotConfig
}

type SlackBotConfig struct {
	BotToken         string `envconfig:"SLACK_BOT_TOKEN"`
	AppToken         string `envconfig:"SLACK_APP_TOKEN"`
	BotName          string `envconfig:"SLACK_BOT_NAME" default:"keel"`
	ApprovalsChannel string `envconfig:"SLACK_APPROVALS_CHANNEL" default:"general"`
}

type HipchatBotConfig struct {
	ApprovalsChannel   string `envconfig:"HIPCHAT_APPROVALS_CHANNEL" default:"general"`
	ApprovalsUserName  string `envconfig:"HIPCHAT_APPROVALS_USER_NAME"`
	ApprovalsBotName   string `envconfig:"HIPCHAT_APPROVALS_BOT_NAME" default:"keel"`
	ApprovalsPassword  string `envconfig:"HIPCHAT_APPROVALS_PASSWORT"`
	ConnectionAttempts int    `envconfig:"HIPCHAT_CONNECTION_ATTEMPTS" default:"10"`
}

type AuthConfig struct {
	BasicUser             string `envconfig:"BASIC_AUTH_USER"`
	BasicPassword         string `envconfig:"BASIC_AUTH_PASSWORD"`
	AuthenticatedWebhooks bool   `envconfig:"AUTHENTICATED_WEBHOOKS" default:"false"`
	TokenSecret           string `envconfig:"TOKEN_SECRET"`
	Mode                  string `envconfig:"AUTH_MODE"`
	ProxyUserHeader       string `envconfig:"AUTH_PROXY_USER_HEADER"`
	ProxyLogoutURL        string `envconfig:"AUTH_PROXY_LOGOUT_URL"`
}

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
