package constants

// DefaultDockerRegistry - default docker registry
const DefaultDockerRegistry = "https://index.docker.io"

// DefaultNamespace - default namespace to initialise configmaps based kv
const DefaultNamespace = "kube-system"

// Deprecated: environment variable names now live in pkg/config. These aliases
// remain while packages are migrated to receive typed configuration directly.
const (
	WebhookEndpointEnv           = "WEBHOOK_ENDPOINT"
	EnvSlackBotToken             = "SLACK_BOT_TOKEN"
	EnvSlackAppToken             = "SLACK_APP_TOKEN"
	EnvSlackBotName              = "SLACK_BOT_NAME"
	EnvSlackChannels             = "SLACK_CHANNELS"
	EnvSlackApprovalsChannel     = "SLACK_APPROVALS_CHANNEL"
	EnvHipchatToken              = "HIPCHAT_TOKEN"
	EnvHipchatBotName            = "HIPCHAT_BOT_NAME"
	EnvHipchatChannels           = "HIPCHAT_CHANNELS"
	EnvHipchatApprovalsChannel   = "HIPCHAT_APPROVALS_CHANNEL"
	EnvHipchatApprovalsUserName  = "HIPCHAT_APPROVALS_USER_NAME"
	EnvHipchatApprovalsBotName   = "HIPCHAT_APPROVALS_BOT_NAME"
	EnvHipchatApprovalsPasswort  = "HIPCHAT_APPROVALS_PASSWORT"
	EnvHipchatConnectionAttempts = "HIPCHAT_CONNECTION_ATTEMPTS"
	EnvMattermostEndpoint        = "MATTERMOST_ENDPOINT"
	EnvMattermostName            = "MATTERMOST_USERNAME"
	EnvTeamsWebhookUrl           = "TEAMS_WEBHOOK_URL"
	EnvDiscordWebhookUrl         = "DISCORD_WEBHOOK_URL"
	EnvShoutrrrURLs              = "SHOUTRRR_URLS"
	EnvShoutrrrTimeout           = "SHOUTRRR_TIMEOUT"
	EnvMailTo                    = "MAIL_TO"
	EnvMailFrom                  = "MAIL_FROM"
	EnvMailSmtpServer            = "MAIL_SMTP_SERVER"
	EnvMailSmtpPort              = "MAIL_SMTP_PORT"
	EnvMailSmtpUser              = "MAIL_SMTP_USER"
	EnvMailSmtpPass              = "MAIL_SMTP_PASS"
	EnvNotificationLevel         = "NOTIFICATION_LEVEL"
	EnvBasicAuthUser             = "BASIC_AUTH_USER"
	EnvBasicAuthPassword         = "BASIC_AUTH_PASSWORD"
	EnvAuthenticatedWebhooks     = "AUTHENTICATED_WEBHOOKS"
	EnvTokenSecret               = "TOKEN_SECRET"
	EnvAuthMode                  = "AUTH_MODE"
	EnvAuthProxyUserHeader       = "AUTH_PROXY_USER_HEADER"
	EnvAuthProxyLogoutURL        = "AUTH_PROXY_LOGOUT_URL"
	EnvRestrictedNamespace       = "RESTRICTED_NAMESPACE"
)

// KeelLogoURL - is a logo URL for bot icon
const KeelLogoURL = "https://keel.sh/img/logo.png"
