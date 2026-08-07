package slack

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/slack-go/slack"

	"github.com/keel-hq/keel/constants"
	"github.com/keel-hq/keel/extension/notification"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/version"

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
	cfg := config.Application.Notifications.Slack
	if cfg.BotToken == "" {
		return false, nil
	}
	s.botName = cfg.BotName
	if cfg.Channels == "" {
		s.channels = []string{"general"}
	} else {
		s.channels = strings.Split(cfg.Channels, ",")
	}
	s.slackClient = slack.New(cfg.BotToken)
	log.WithField("channels", s.channels).Info("extension.notification.slack: sender configured")
	return true, nil
}

func (s *sender) Send(event types.EventNotification) error {
	params := slack.NewPostMessageParameters()
	params.Username = s.botName
	params.IconURL = constants.KeelLogoURL

	attachements := []slack.Attachment{
		{
			Fallback: event.Message,
			Color:    event.Level.Color(),
			Fields: []slack.AttachmentField{
				{
					Title: event.Type.String(),
					Value: event.Message,
					Short: false,
				},
			},
			Footer: fmt.Sprintf("https://keel.sh %s", version.GetKeelVersion().Version),
			Ts:     json.Number(strconv.Itoa(int(event.CreatedAt.Unix()))),
		},
	}

	chans := s.channels
	if len(event.Channels) > 0 {
		chans = event.Channels
	}

	var mgsOpts []slack.MsgOption

	mgsOpts = append(mgsOpts, slack.MsgOptionPostMessageParameters(params))
	mgsOpts = append(mgsOpts, slack.MsgOptionAttachments(attachements...))

	for _, channel := range chans {
		_, _, err := s.slackClient.PostMessage(channel, mgsOpts...)
		if err != nil {
			log.WithFields(log.Fields{
				"error":   err,
				"channel": channel,
			}).Error("extension.notification.slack: failed to send notification")
		}
	}
	return nil
}
