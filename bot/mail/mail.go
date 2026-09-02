package mail

import (
	"context"
	"os"
	"strconv"

	"net/smtp"

	"github.com/keel-hq/keel/bot"
	"github.com/keel-hq/keel/constants"
	"github.com/keel-hq/keel/types"

	log "github.com/sirupsen/logrus"
)


// Bot - main mail notification bot container
type Bot struct {
	from       string
	to         string
	smtpServer string
	smtpPort   int
	smtpUser   string
	smtpPass   string


	ctx                context.Context
}

func init() {
	bot.RegisterBot("mail", &Bot{})
}

func (b *Bot) Configure(approvalsRespCh chan *bot.ApprovalResponse, botMessagesChannel chan *bot.BotMessage) bool {
	// Server, from and to are mandatory
	if os.Getenv(constants.EnvMailSmtpServer) == "" ||
	   	os.Getenv(constants.EnvMailFrom) != "" ||
		os.Getenv(constants.EnvMailTo) != "" {
		b.smtpServer = os.Getenv(constants.EnvMailSmtpServer)
		b.from = os.Getenv(constants.EnvMailFrom)
		b.to = os.Getenv(constants.EnvMailTo)

		// Port, user and pass are optional
		if os.Getenv(constants.EnvMailSmtpPort) != "" {
			port, err := strconv.Atoi(os.Getenv(constants.EnvMailSmtpPort))
			if err != nil {
				log.WithFields(log.Fields{
					"name": "mail",
				}).Warn("bot.mail.Configure(): invalid SMTP port number")
				return false
			}
			b.smtpPort = port
		} else {
			b.smtpPort = 25
		}
		if os.Getenv(constants.EnvMailSmtpUser) != "" {
			b.smtpUser = os.Getenv(constants.EnvMailSmtpUser)
		}
		if os.Getenv(constants.EnvMailSmtpPass) != "" {
			b.smtpPass = os.Getenv(constants.EnvMailSmtpPass)
		}
		
		log.WithFields(log.Fields{
			"name": "mail",
		}).Info("bot.mail.Configure():: mail box configured")
		
		return true;
	}
	log.Info("bot.mail.Configure(): mail bot not configured")
	return false
}

// Start - start bot
func (b *Bot) Start(ctx context.Context) error {
	// setting root context
	b.ctx = ctx

	return nil
}

func (b *Bot) postMessage(title, message string) error {
	log.Info("bot.mail.postMessage(): post a message " + message)
	body := message
	msg := "From: " + b.from + "\n" +
		"To: " + b.to + "\n" +
		"Subject: Keel notification " + title + "\n\n" +
		body

	// Support only plain auth
	var auth smtp.Auth = nil
	if b.smtpUser != "" {
		auth = smtp.PlainAuth(
			"",
			b.smtpUser,
			b.smtpPass,
			b.smtpServer,
		)
	}

	err := smtp.SendMail(b.smtpServer+":"+strconv.Itoa(b.smtpPort), auth, b.from, []string{b.to}, []byte(msg))
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("extension.notification.mail: failed to send notification")
	}
	
	return err
}

func (b *Bot) Respond(text string, channel string) {
}

// Request - request approval
func (b *Bot) RequestApproval(req *types.Approval) error {
	return b.postMessage(
		"Approval required",
		req.Message)
}

func (b *Bot) ReplyToApproval(approval *types.Approval) error {
	return nil
}
