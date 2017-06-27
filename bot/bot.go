package bot

import (
	"github.com/nlopes/slack"
)

type Bot struct {
	slackClient *slack.Client
}

func New(token string) *Bot {
	client := slack.New(token)

	return &Bot{
		slackClient: client,
	}
}
