package hipchat

import (
	"context"
	"errors"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/bot"
	"github.com/keel-hq/keel/constants"
	"github.com/keel-hq/keel/provider/kubernetes"
	"github.com/keel-hq/keel/types"

	log "github.com/Sirupsen/logrus"
	"github.com/daneharrigan/hipchat"
)

// Bot - main hipchat bot container
type Bot struct {
	id          string // bot id
	name        string // bot name
	mentionName string

	userName string // bot user name
	password string // bot user password

	users map[string]string

	msgPrefix string

	hipchatClient *hipchat.Client

	approvalsRespCh chan *bot.ApprovalResponse

	approvalsManager approvals.Manager
	approvalsChannel string // hipchat approvals channel name

	k8sImplementer kubernetes.Implementer

	ctx context.Context
}

func init() {
	bot.RegisterBot("hipchat", Run)
}

// Run ...
func Run(k8sImplementer kubernetes.Implementer, approvalsManager approvals.Manager) (teardown func(), err error) {

	if os.Getenv(constants.EnvHipchatApprovalsPasswort) != "" &&
		os.Getenv(constants.EnvHipchatApprovalsUserName) != "" {
		botName := "keel"
		if os.Getenv(constants.EnvHipchatApprovalsBotName) != "" {
			botName = os.Getenv(constants.EnvHipchatApprovalsBotName)
		}

		botUserName := ""
		if os.Getenv(constants.EnvHipchatApprovalsUserName) != "" {
			botUserName = os.Getenv(constants.EnvHipchatApprovalsUserName)
		}

		pass := os.Getenv(constants.EnvHipchatApprovalsPasswort)

		approvalsChannel := "general"
		if os.Getenv(constants.EnvHipchatApprovalsChannel) != "" {
			approvalsChannel = os.Getenv(constants.EnvHipchatApprovalsChannel)
		}

		bot := new(botName, botUserName, pass, approvalsChannel, k8sImplementer, approvalsManager)

		ctx, cancel := context.WithCancel(context.Background())

		err := bot.Start(ctx)
		if err != nil {
			cancel()
			return nil, err
		}

		teardown := func() {
			// cancelling context
			cancel()
		}

		return teardown, nil
	}

	return func() {}, nil
}

//--------------------- <XMPP client> -------------------------------------

func connect(username, password string) *hipchat.Client {
	attempts := 10
	for {
		log.Debug("try to connect to hipchat")
		client, err := hipchat.NewClient(username, password, "bot", "plain")
		// could not authenticate
		if err != nil {
			log.Errorf("bot.hipchat.connect: Error=%s", err)
			if err.Error() == "could not authenticate" {
				return nil
			}
		}
		if attempts == 0 {
			return nil
		}
		if client != nil && err == nil {
			return client
		}
		log.Debugln("wait fo 30 seconds")
		time.Sleep(30 * time.Second)
		attempts--
	}
}

//--------------------- </XMPP client> -------------------------------------

func new(name, username, pass, approvalsChannel string, k8sImplementer kubernetes.Implementer, approvalsManager approvals.Manager) *Bot {

	client := connect(username, pass)

	bot := &Bot{
		hipchatClient:    client,
		k8sImplementer:   k8sImplementer,
		name:             name,
		mentionName:      "@" + strings.Replace(name, " ", "", -1),
		approvalsManager: approvalsManager,
		approvalsChannel: approvalsChannel,                 // roomJid
		approvalsRespCh:  make(chan *bot.ApprovalResponse), // don't add buffer to make it blocking
	}

	return bot
}

// Start the bot
func (b *Bot) Start(ctx context.Context) error {
	log.Debugln(">>> bot.hipchat.Start()")

	if b.hipchatClient == nil {
		return errors.New("could not conect to hipchat server")
	}

	// setting root context
	b.ctx = ctx

	// processing messages coming from slack RTM client
	go b.startInternal()

	// processing slack approval responses
	go b.processApprovalResponses()

	// subscribing for approval requests
	go b.subscribeForApprovals()

	return nil
}

func (b *Bot) startInternal() error {
	client := b.hipchatClient
	log.Debugf("startInternal(): channel=%s, userName=%s\n", b.approvalsChannel, b.name)
	client.Status("chat") // chat, away or idle
	client.Join(b.approvalsChannel, b.name)
	go client.KeepAlive()
	go func() {
		for {
			select {
			case message := <-client.Messages():
				b.handleMessage(message)
			}
		}
	}()

	return nil
}

func (b *Bot) handleMessage(message *hipchat.Message) {
	msg := b.trimXMPPMessage(message)
	if msg.From == "" || msg.To == "" {
		return
	}

	if !b.isBotMessage(msg) {
		log.Debugln("hipchat.handleMessage(): is not a bot message")
		return
	}

	approval, ok := b.isApproval(msg)
	if ok {
		b.approvalsRespCh <- approval
		return
	}

	if responseLines, ok := bot.BotEventTextToResponse[msg.Body]; ok {
		response := strings.Join(responseLines, "\n")
		b.respond(formatAsSnippet(response))
		return
	}

	if b.isCommand(msg) {
		b.handleCommand(msg)
		return
	}

	log.WithFields(log.Fields{
		"name":      b.name,
		"bot_id":    b.id,
		"command":   msg.Body,
		"untrimmed": message.Body,
	}).Debug("handleMessage: bot couldn't recognise command")
}

func (b *Bot) respond(response string) {
	b.hipchatClient.Say(b.approvalsChannel, b.name, response)
}

func (b *Bot) handleCommand(message *hipchat.Message) {
	switch message.Body {
	case "get deployments":
		log.Info("getting deployments")
		response := bot.DeploymentsResponse(bot.Filter{}, b.k8sImplementer)
		b.respond(formatAsSnippet(response))
		return
	case "get approvals":
		log.Info("getting approvals")
		response := b.approvalsResponse()
		b.respond(formatAsSnippet(response))
		return
	}
}

func formatAsSnippet(msg string) string {
	return "/code " + msg
}

func (b *Bot) isCommand(message *hipchat.Message) bool {

	if bot.StaticBotCommands[message.Body] {
		return true
	}

	for _, prefix := range bot.DynamicBotCommandPrefixes {
		if strings.HasPrefix(message.Body, prefix) {
			return true
		}
	}

	return false
}

func (b *Bot) trimXMPPMessage(message *hipchat.Message) *hipchat.Message {
	msg := hipchat.Message{}
	msg.MentionName = trimMentionName(message.Body)
	msg.Body = b.trimBot(message.Body)
	msg.From = b.trimUser(message.From)
	msg.To = b.trimUser(message.To)

	return &msg
}

func trimMentionName(message string) string {
	re := regexp.MustCompile(`^(@\w+)`)
	match := re.FindStringSubmatch(message)
	if match == nil {
		return ""
	}
	if len(match) != 0 {
		return match[1]
	}
	return ""
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
	msg = strings.TrimPrefix(msg, b.mentionName)
	msg = strings.Trim(msg, "\n")
	msg = strings.TrimSpace(msg)
	return strings.ToLower(msg)
}

func (b *Bot) isApproval(message *hipchat.Message) (resp *bot.ApprovalResponse, ok bool) {

	if strings.HasPrefix(message.Body, bot.ApprovalResponseKeyword) {
		return &bot.ApprovalResponse{
			User:   message.From,
			Status: types.ApprovalStatusApproved,
			Text:   message.Body,
		}, true
	}

	if strings.HasPrefix(message.Body, bot.RejectResponseKeyword) {
		return &bot.ApprovalResponse{
			User:   message.From,
			Status: types.ApprovalStatusRejected,
			Text:   message.Body,
		}, true
	}

	return nil, false
}

func (b *Bot) isBotMessage(message *hipchat.Message) bool {
	if message.MentionName == b.mentionName {
		return true
	}
	return false
}
