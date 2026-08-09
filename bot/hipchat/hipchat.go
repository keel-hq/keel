package hipchat

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/keel-hq/keel/bot"
	"github.com/keel-hq/keel/pkg/config"

	h "github.com/daneharrigan/hipchat"
	log "github.com/sirupsen/logrus"
)

// Bot - main hipchat bot container
type Bot struct {
	id   string // bot id
	name string // bot name

	userName string // bot user name
	password string // bot user password

	hipchatClient    XmppImplementer
	approvalsChannel string
	ctx              context.Context

	botMessagesChannel chan *bot.BotMessage
	approvalsRespCh    chan *bot.ApprovalResponse
}

func init() {
	bot.RegisterBot("hipchat", &Bot{})
}

func (b *Bot) Configure(appConfig config.Config, approvalsRespCh chan *bot.ApprovalResponse, botMessagesChannel chan *bot.BotMessage) bool {
	cfg := appConfig.Bots.Hipchat
	if cfg.ApprovalsPassword == "" || cfg.ApprovalsUserName == "" {
		log.Info("bot.hipchat.Configure(): HipChat approval bot is not configured")
		return false
	}
	b.name = cfg.ApprovalsBotName
	b.userName = cfg.ApprovalsUserName
	b.password = cfg.ApprovalsPassword
	if cfg.ConnectionAttempts > 0 {
		if cli := connect(b.userName, b.password, cfg.ConnectionAttempts); cli != nil {
			b.hipchatClient = cli
		}
	}
	b.botMessagesChannel = botMessagesChannel
	b.approvalsRespCh = approvalsRespCh
	b.approvalsChannel = cfg.ApprovalsChannel
	return true
}

// Start the bot
func (b *Bot) Start(ctx context.Context) error {
	if b.hipchatClient == nil {
		return errors.New("could not conect to hipchat server")
	}

	// setting root context
	b.ctx = ctx
	client := b.hipchatClient
	client.Status("chat")
	client.Join(b.approvalsChannel, b.name)
	b.postMessage("Keel bot was started")
	go client.KeepAlive()
	go func() {
		for {
			select {
			case <-b.ctx.Done():
				return
			case message := <-client.Messages():
				b.handleMessage(message)
			}
		}
	}()

	return nil
}

func (b *Bot) Respond(response string, channel string) {
	b.hipchatClient.Say(channel, b.name, formatAsSnippet(response))
}

func (b *Bot) handleMessage(message *h.Message) {
	msg := b.trimXMPPMessage(message)
	if msg.From == "" || msg.To == "" {
		log.Debugln("hipchat.handleMessage(): fields 'From:' or 'To:' are empty, ignore")
		return
	}

	if !b.isBotMessage(msg) {
		log.Debugf("handleMessage(): is not a bot message [%#v]", msg)
		return
	}

	approval, ok := bot.IsApproval(msg.From, msg.Body)
	if ok {
		b.approvalsRespCh <- approval
		return
	}

	b.botMessagesChannel <- &bot.BotMessage{
		Message: msg.Body,
		User:    msg.From,
		Channel: b.approvalsChannel,
		Name:    "hipchat",
	}

	return
}

func formatAsSnippet(msg string) string {
	return "/code " + msg
}

func (b *Bot) trimXMPPMessage(message *h.Message) *h.Message {
	msg := h.Message{}
	msg.Body = b.trimBot(message.Body)
	msg.From = b.trimUser(message.From)
	msg.To = b.trimUser(message.To)

	return &msg
}

func (b *Bot) trimUser(user string) string {
	re := regexp.MustCompile("/(.*?)$")
	match := re.FindStringSubmatch(user)
	if match == nil {
		return ""
	}
	if len(match) != 0 {
		return match[1]
	}
	return ""
}

func (b *Bot) postMessage(msg string) error {
	b.hipchatClient.Say(b.approvalsChannel, b.name, msg)
	return nil
}

func (b *Bot) trimBot(msg string) string {
	var re = regexp.MustCompile(`(^@\w+)`)
	msg = re.ReplaceAllString(msg, "")
	msg = strings.Trim(msg, "\n")
	msg = strings.TrimSpace(msg)
	return strings.ToLower(msg)
}

func (b *Bot) isBotMessage(message *h.Message) bool {
	if message.To == "bot" {
		return true
	}
	return false
}
