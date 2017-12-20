package constants

// DefaultDockerRegistry - default docker registry
const DefaultDockerRegistry = "https://index.docker.io"

// DefaultNamespace - default namespace to initialise configmaps based kv
const DefaultNamespace = "kube-system"

// WebhookEndpointEnv if set - enables webhook notifications
const WebhookEndpointEnv = "WEBHOOK_ENDPOINT"

// slack bot/token
const (
	EnvSlackToken            = "SLACK_TOKEN"
	EnvSlackBotName          = "SLACK_BOT_NAME"
	EnvSlackChannels         = "SLACK_CHANNELS"
	EnvSlackApprovalsChannel = "SLACK_APPROVALS_CHANNEL"

	EnvHipchatToken    = "HIPCHAT_TOKEN"
	EnvHipchatBotName  = "HIPCHAT_BOT_NAME"
	EnvHipchatChannels = "HIPCHAT_CHANNELS"

	EnvHipchatApprovalsChannel  = "HIPCHAT_APPROVALS_CHANNEL"
	EnvHipchatApprovalsUserName = "HIPCHAT_APPROVALS_USER_NAME"
	EnvHipchatApprovalsBotName  = "HIPCHAT_APPROVALS_BOT_NAME"
	EnvHipchatApprovalsPasswort = "HIPCHAT_APPROVALS_PASSWORT"
)

// EnvNotificationLevel - minimum level for notifications, defaults to info
const EnvNotificationLevel = "NOTIFICATION_LEVEL"
