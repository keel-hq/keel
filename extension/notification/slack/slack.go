package slack

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nlopes/slack"

	"github.com/keel-hq/keel/constants"
	"github.com/keel-hq/keel/extension/notification"
	"github.com/keel-hq/keel/types"

	log "github.com/sirupsen/logrus"
)

const timeout = 5 * time.Second

type sender struct {
	slackClient *slack.Client
	channels    []string
	botName     string
}

func init() {
	notification.RegisterSender("slack", &sender{})
}

func (s *sender) Configure(config *notification.Config) (bool, error) {
	var token string
	// Get configuration
	if os.Getenv(constants.EnvSlackToken) != "" {
		token = os.Getenv(constants.EnvSlackToken)
	} else {
		return false, nil
	}
	if os.Getenv(constants.EnvSlackBotName) != "" {
		s.botName = os.Getenv(constants.EnvSlackBotName)
	} else {
		s.botName = "keel"
	}

	if os.Getenv(constants.EnvSlackChannels) != "" {
		channels := os.Getenv(constants.EnvSlackChannels)
		s.channels = strings.Split(channels, ",")
	} else {
		s.channels = []string{"general"}
	}

	s.slackClient = slack.New(token)

	log.WithFields(log.Fields{
		"name":     "slack",
		"channels": s.channels,
	}).Info("extension.notification.slack: sender configured")

	return true, nil
}

func (s *sender) Send(event types.EventNotification) error {
	params := slack.NewPostMessageParameters()
	params.Username = s.botName
	params.IconURL = constants.KeelLogoURL

	params.Attachments = []slack.Attachment{
		slack.Attachment{
			Fallback: event.Message,
			Color:    event.Level.Color(),
			Fields: []slack.AttachmentField{
				slack.AttachmentField{
					Title: event.Type.String(),
					Value: event.Message,
					Short: false,
				},
			},
			Footer: "keel.sh",
			Ts:     json.Number(strconv.Itoa(int(event.CreatedAt.Unix()))),
		},
	}

	chans := s.channels
	if len(event.Channels) > 0 {
		chans = event.Channels
	}

	for _, channel := range chans {
		_, _, err := s.slackClient.PostMessage(channel, "", params)
		if err != nil {
			log.WithFields(log.Fields{
				"error":   err,
				"channel": channel,
			}).Error("extension.notification.slack: failed to send notification")
		}
	}
	return nil
}
