package hipchat

import (
	"fmt"
	"strings"

	"net/url"

	"github.com/tbruyelle/hipchat-go/hipchat"

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
	cfg := config.Application.Notifications.Hipchat
	if cfg.Token == "" {
		return false, nil
	}
	s.botName = cfg.BotName
	if cfg.Channels == "" {
		s.channels = []string{"general"}
	} else {
		s.channels = strings.Split(cfg.Channels, ",")
	}
	s.hipchatClient = hipchat.NewClient(cfg.Token)
	if cfg.Server != "" {
		server, err := url.Parse(cfg.Server)
		if err != nil {
			return false, err
		}
		s.hipchatClient.BaseURL = server
	}
	log.WithField("channels", s.channels).Info("extension.notification.hipchat: sender configured")
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
