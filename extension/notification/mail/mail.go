package mail

import (
	"net/smtp"
	"strconv"

	"github.com/keel-hq/keel/extension/notification"
	"github.com/keel-hq/keel/types"

	log "github.com/sirupsen/logrus"
)

type sender struct {
	from       string
	to         string
	smtpServer string
	smtpPort   int
	smtpUser   string
	smtpPass   string
}

func init() {
	notification.RegisterSender("mail", &sender{})
}

func (s *sender) Configure(config *notification.Config) (bool, error) {
	cfg := config.Application.Notifications.Mail
	if cfg.SMTPServer == "" || cfg.From == "" || cfg.To == "" {
		return false, nil
	}
	s.smtpServer, s.from, s.to, s.smtpPort = cfg.SMTPServer, cfg.From, cfg.To, cfg.SMTPPort
	s.smtpUser, s.smtpPass = cfg.SMTPUser, cfg.SMTPPass
	log.WithField("name", "mail").Info("extension.notification.mail: sender configured")
	return true, nil
}

func (s *sender) Send(event types.EventNotification) error {
	body := event.CreatedAt.String() + "\n" + event.Level.String() + "-" +
		event.Type.String() + "\n" + event.Message
	msg := "From: " + s.from + "\n" +
		"To: " + s.to + "\n" +
		"Subject: Keel notification\n\n" +
		body

	// Support only plain auth
	var auth smtp.Auth = nil
	if s.smtpUser != "" {
		auth = smtp.PlainAuth(
			"",
			s.smtpUser,
			s.smtpPass,
			s.smtpServer,
		)
	}

	err := smtp.SendMail(s.smtpServer+":"+strconv.Itoa(s.smtpPort), auth, s.from, []string{s.to}, []byte(msg))
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("extension.notification.mail: failed to send notification")
	}

	return nil
}
