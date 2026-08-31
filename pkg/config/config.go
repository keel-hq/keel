package config

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/kelseyhightower/envconfig"
)

var loadMutex sync.Mutex

var environmentVariables = []string{
	"DEBUG", "PUBSUB", "POLL", "POLL_SCAN_INTERVAL", "PROJECT_ID", "CLUSTER_NAME", "XDG_DATA_HOME", "HELM3_PROVIDER", "UI_DIR",
	"NOTIFICATION_LEVEL", "WEBHOOK_ENDPOINT", "SLACK_BOT_TOKEN", "SLACK_APP_TOKEN", "SLACK_BOT_NAME", "SLACK_CHANNELS", "SLACK_APPROVALS_CHANNEL",
	"HIPCHAT_SERVER", "HIPCHAT_TOKEN", "HIPCHAT_BOT_NAME", "HIPCHAT_CHANNELS", "HIPCHAT_APPROVALS_CHANNEL", "HIPCHAT_APPROVALS_USER_NAME",
	"HIPCHAT_APPROVALS_BOT_NAME", "HIPCHAT_APPROVALS_PASSWORT", "HIPCHAT_CONNECTION_ATTEMPTS", "MATTERMOST_ENDPOINT", "MATTERMOST_USERNAME",
	"TEAMS_WEBHOOK_URL", "DISCORD_WEBHOOK_URL", "SHOUTRRR_URLS", "SHOUTRRR_TIMEOUT", "MAIL_TO", "MAIL_FROM", "MAIL_SMTP_SERVER",
	"MAIL_SMTP_PORT", "MAIL_SMTP_USER", "MAIL_SMTP_PASS", "BASIC_AUTH_USER", "BASIC_AUTH_PASSWORD", "AUTHENTICATED_WEBHOOKS",
	"TOKEN_SECRET", "AUTH_MODE", "AUTH_PROXY_USER_HEADER", "AUTH_PROXY_LOGOUT_URL", "RESTRICTED_NAMESPACE",
}

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
	PubSub           bool          `ignored:"true"`
	Poll             bool          `envconfig:"POLL" default:"true"`
	PollScanInterval time.Duration `envconfig:"POLL_SCAN_INTERVAL" default:"1m"`
	ProjectID        string        `envconfig:"PROJECT_ID"`
	ClusterName      string        `envconfig:"CLUSTER_NAME"`
}

// StorageConfig controls where Keel stores its persistent application data.
type StorageConfig struct {
	DataDir string `envconfig:"XDG_DATA_HOME" default:"/data"`
}

// DefaultEventBufferSize is the default capacity of the per-provider event
// buffer that queues update events between the triggers (poll, webhook,
// pubsub, approvals) and the deployment sync workers.
const DefaultEventBufferSize = 512

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
	// BotToken and BotName intentionally share environment variables with the
	// Slack approvals bot to preserve the existing combined configuration.
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
	// BotToken and BotName intentionally share environment variables with the
	// Slack notification sender to preserve the existing combined configuration.
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
	ConnectionAttempts int    `envconfig:"HIPCHAT_CONNECTION_ATTEMPTS" default:"5"`
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
	loadMutex.Lock()
	defer loadMutex.Unlock()

	restore, err := normalizeEmptyEnvironment()
	if err != nil {
		return Config{}, err
	}
	defer restore()

	var cfg Config
	root := struct {
		Debug bool `envconfig:"DEBUG" default:"false"`
	}{}
	notifications := struct {
		Level string `envconfig:"NOTIFICATION_LEVEL" default:"info"`
	}{}

	sections := []interface{}{
		&root,
		&cfg.Trigger,
		&cfg.Storage,
		&cfg.Providers,
		&cfg.UI,
		&notifications,
		&cfg.Notifications.Webhook,
		&cfg.Notifications.Slack,
		&cfg.Notifications.Hipchat,
		&cfg.Notifications.Mattermost,
		&cfg.Notifications.Teams,
		&cfg.Notifications.Discord,
		&cfg.Notifications.Shoutrrr,
		&cfg.Notifications.Mail,
		&cfg.Bots.Slack,
		&cfg.Bots.Hipchat,
		&cfg.Auth,
		&cfg.Kubernetes,
	}
	for _, section := range sections {
		if err := envconfig.Process("", section); err != nil {
			return Config{}, err
		}
	}
	if cfg.Trigger.PollScanInterval <= 0 {
		return Config{}, fmt.Errorf("POLL_SCAN_INTERVAL must be greater than zero")
	}

	cfg.Debug = root.Debug
	cfg.Notifications.Level = notifications.Level
	if value, ok := os.LookupEnv("PUBSUB"); ok {
		cfg.Trigger.PubSub = legacyFeatureFlag(value)
	}
	return cfg, nil
}

// normalizeEmptyEnvironment preserves the historical behavior of treating an
// explicitly empty manifest value as unconfigured. envconfig otherwise tries
// to parse empty booleans and integers and turns a safe upgrade into a fatal
// startup error.
func normalizeEmptyEnvironment() (func(), error) {
	var empty []string
	for _, name := range environmentVariables {
		if value, ok := os.LookupEnv(name); ok && value == "" {
			if err := os.Unsetenv(name); err != nil {
				restoreEmptyEnvironment(empty)
				return nil, fmt.Errorf("temporarily unset empty configuration variable %s: %w", name, err)
			}
			empty = append(empty, name)
		}
	}
	return func() { restoreEmptyEnvironment(empty) }, nil
}

func restoreEmptyEnvironment(names []string) {
	for _, name := range names {
		_ = os.Setenv(name, "")
	}
}

// legacyFeatureFlag retains PUBSUB's historical any-non-empty behavior while
// honoring explicit false values introduced by typed configuration.
func legacyFeatureFlag(value string) bool {
	parsed, err := strconv.ParseBool(value)
	if err == nil {
		return parsed
	}
	return value != ""
}
