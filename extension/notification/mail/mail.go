package mail

import (
	"net/smtp"
	"os"
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
	// Server, from and to are mandatory
	if os.Getenv("MAIL_SMTP_SERVER") != "" {
		s.smtpServer = os.Getenv("MAIL_SMTP_SERVER")
	} else {
		return false, nil
	}
	if os.Getenv("MAIL_FROM") != "" {
		s.from = os.Getenv("MAIL_FROM")
	} else {
		return false, nil
	}
	if os.Getenv("MAIL_TO") != "" {
		s.to = os.Getenv("MAIL_TO")
	} else {
		return false, nil
	}
	// Port, user and pass are optional
	if os.Getenv("MAIL_SMTP_PORT") != "" {
		port, err := strconv.Atoi(os.Getenv("MAIL_SMTP_PORT"))
		if err != nil {
			log.WithFields(log.Fields{
				"name": "mail",
			}).Warn("extension.notification.mail: invalid SMTP port number")
			return false, nil
		}
		s.smtpPort = port
	} else {
		s.smtpPort = 25
	}
	if os.Getenv("MAIL_SMTP_USER") != "" {
		s.smtpUser = os.Getenv("MAIL_SMTP_USER")
	}
	if os.Getenv("MAIL_SMTP_PASS") != "" {
		s.smtpPass = os.Getenv("MAIL_SMTP_PASS")
	}

	log.WithFields(log.Fields{
		"name": "mail",
	}).Info("extension.notification.mail: sender configured")

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
