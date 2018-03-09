package hipchat

import (
	"fmt"
	"os"
	"strings"

	"github.com/tbruyelle/hipchat-go/hipchat"

	"github.com/keel-hq/keel/constants"
	"github.com/keel-hq/keel/extension/notification"
	"github.com/keel-hq/keel/types"

	log "github.com/sirupsen/logrus"
)

type sender struct {
	hipchatClient *hipchat.Client
	channels      []string
	botName       string
}

func init() {
	notification.RegisterSender("hipchat", &sender{})
}

func (s *sender) Configure(config *notification.Config) (bool, error) {
	var token string

	if os.Getenv(constants.EnvHipchatToken) != "" {
		token = os.Getenv(constants.EnvHipchatToken)
	} else {
		return false, nil
	}
	if os.Getenv(constants.EnvHipchatBotName) != "" {
		s.botName = os.Getenv(constants.EnvHipchatBotName)
	} else {
		s.botName = "keel"
	}

	if os.Getenv(constants.EnvHipchatChannels) != "" {
		channels := os.Getenv(constants.EnvHipchatChannels)
		s.channels = strings.Split(channels, ",")
	} else {
		s.channels = []string{"general"}
	}

	s.hipchatClient = hipchat.NewClient(token)

	log.WithFields(log.Fields{
		"name":     "hipchat",
		"channels": s.channels,
	}).Info("extension.notification.hipchat: sender configured")

	return true, nil
}

func (s *sender) Send(event types.EventNotification) error {
	msg := fmt.Sprintf("<b>%s</b><br>%s", event.Type.String(), event.Message)

	notification := &hipchat.NotificationRequest{
		Color:   getHipchatColor(event.Level.String()),
		Message: msg,
		Notify:  true,
		From:    s.botName,
	}

	channels := s.channels
	if len(event.Channels) > 0 {
		channels = event.Channels
	}

	for _, channel := range channels {
		_, err := s.hipchatClient.Room.Notification(channel, notification)
		if err != nil {
			log.WithFields(log.Fields{
				"error":   err,
				"channel": channel,
			}).Error("extension.notification.hipchat: failed to send notification")
		}
	}

	return nil
}

func getHipchatColor(eventLevel string) hipchat.Color {
	switch eventLevel {
	case "error":
		return "red"
	case "info":
		return "gray"
	case "success":
		return "green"
	case "fatal":
		return "purple"
	case "warn":
		return "yellow"
	default:
		return "gray"
	}
}
