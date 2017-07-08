package constants

// DefaultDockerRegistry - default docker registry
const DefaultDockerRegistry = "https://index.docker.io"

// WebhookEndpointEnv if set - enables webhook notifications
const WebhookEndpointEnv = "WEBHOOK_ENDPOINT"

// slack bot/token
const (
	EnvSlackToken    = "SLACK_TOKEN"
	EnvSlackBotName  = "SLACK_BOT_NAME"
	EnvSlackChannels = "SLACK_CHANNELS"
)
