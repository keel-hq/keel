package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/nlopes/slack"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/provider/kubernetes"
	"github.com/keel-hq/keel/types"

	log "github.com/Sirupsen/logrus"
)

const (
	removeApprovalPrefix = "rm approval"
)

var (
	botEventTextToResponse = map[string][]string{
		"help": {
			`Here's a list of supported commands`,
			`- "get deployments" -> get a list of all deployments`,
			`- "get approvals" -> get a list of approvals`,
			`- "rm approval <approval identifier>" -> remove approval`,
			`- "approve <approval identifier>" -> approve update request`,
			`- "reject <approval identifier>" -> reject update request`,
			// `- "get deployments all" -> get a list of all deployments`,
			// `- "describe deployment <deployment>" -> get details for specified deployment`,
		},
	}

	// static bot commands can be used straight away
	staticBotCommands = map[string]bool{
		"get deployments": true,
		"get approvals":   true,
	}

	// dynamic bot command prefixes have to be matched
	dynamicBotCommandPrefixes = []string{removeApprovalPrefix}

	approvalResponseKeyword = "approve"
	rejectResponseKeyword   = "reject"
)

// SlackImplementer - implementes slack HTTP functionality, used to
// send messages with attachments
type SlackImplementer interface {
	PostMessage(channel, text string, params slack.PostMessageParameters) (string, string, error)
}

// approvalResponse - used to track approvals once vote begins
type approvalResponse struct {
	User   string
	Status types.ApprovalStatus
	Text   string
}

// Bot - main slack bot container
type Bot struct {
	id   string // bot id
	name string // bot name

	users map[string]string

	msgPrefix string

	slackClient *slack.Client
	slackRTM    *slack.RTM

	slackHTTPClient SlackImplementer

	approvalsRespCh chan *approvalResponse

	approvalsManager approvals.Manager
	approvalsChannel string // slack approvals channel name

	k8sImplementer kubernetes.Implementer

	ctx context.Context
}

// New - create new bot instance
func New(name, token, approvalsChannel string, k8sImplementer kubernetes.Implementer, approvalsManager approvals.Manager) *Bot {
	client := slack.New(token)

	bot := &Bot{
		slackClient:      client,
		slackHTTPClient:  client,
		k8sImplementer:   k8sImplementer,
		name:             name,
		approvalsManager: approvalsManager,
		approvalsChannel: approvalsChannel,
		approvalsRespCh:  make(chan *approvalResponse), // don't add buffer to make it blocking
	}

	return bot
}

// Start - start bot
func (b *Bot) Start(ctx context.Context) error {

	// setting root context
	b.ctx = ctx

	users, err := b.slackClient.GetUsers()
	if err != nil {
		return err
	}

	b.users = map[string]string{}

	for _, user := range users {
		switch user.Name {
		case b.name:
			if user.IsBot {
				b.id = user.ID
			}
		default:
			continue
		}
	}
	if b.id == "" {
		return errors.New("could not find bot in the list of names, check if the bot is called \"" + b.name + "\" ")
	}

	b.msgPrefix = strings.ToLower("<@" + b.id + ">")

	// processing messages coming from slack RTM client
	go b.startInternal()

	// processing slack approval responses
	go b.processApprovalResponses()

	// subscribing for approval requests
	go b.subscribeForApprovals()

	return nil
}

func (b *Bot) startInternal() error {
	b.slackRTM = b.slackClient.NewRTM()

	go b.slackRTM.ManageConnection()

	for {
		select {
		case <-b.ctx.Done():
			return nil

		case msg := <-b.slackRTM.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.HelloEvent:
				// Ignore hello

			case *slack.ConnectedEvent:
				// fmt.Println("Infos:", ev.Info)
				// fmt.Println("Connection counter:", ev.ConnectionCount)
				// Replace #general with your Channel ID
				// b.slackRTM.SendMessage(b.slackRTM.NewOutgoingMessage("Hello world", "#general"))

			case *slack.MessageEvent:
				b.handleMessage(ev)
			case *slack.PresenceChangeEvent:
				// fmt.Printf("Presence Change: %v\n", ev)

			// case *slack.LatencyReport:
			// 	fmt.Printf("Current latency: %v\n", ev.Value)

			case *slack.RTMError:
				fmt.Printf("Error: %s\n", ev.Error())

			case *slack.InvalidAuthEvent:
				fmt.Printf("Invalid credentials")
				return fmt.Errorf("invalid credentials")

			default:

				// Ignore other events..
				// fmt.Printf("Unexpected: %v\n", msg.Data)
			}
		}
	}
}

func (b *Bot) postMessage(title, message, color string, fields []slack.AttachmentField) error {
	params := slack.NewPostMessageParameters()
	params.Username = b.name

	params.Attachments = []slack.Attachment{
		slack.Attachment{
			Fallback: message,
			Color:    color,
			Fields:   fields,
			Footer:   "https://keel.sh",
			Ts:       json.Number(strconv.Itoa(int(time.Now().Unix()))),
		},
	}

	_, _, err := b.slackHTTPClient.PostMessage(b.approvalsChannel, "", params)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("bot.postMessage: failed to send message")
	}
	return err
}

func (b *Bot) isApproval(event *slack.MessageEvent, eventText string) (resp *approvalResponse, ok bool) {

	// only accepting approvals from approvals channel
	if !b.isApprovalsChannel(event) {
		return nil, false
	}
	if strings.HasPrefix(strings.ToLower(eventText), approvalResponseKeyword) {
		return &approvalResponse{
			User:   event.User,
			Status: types.ApprovalStatusApproved,
			Text:   eventText,
		}, true
	}

	if strings.HasPrefix(strings.ToLower(eventText), rejectResponseKeyword) {
		return &approvalResponse{
			User:   event.User,
			Status: types.ApprovalStatusRejected,
			Text:   eventText,
		}, true
	}

	return nil, false
}

// TODO(k): cache results in a map or get this info on startup. Although
// if channel was then recreated (unlikely), we would miss results
func (b *Bot) isApprovalsChannel(event *slack.MessageEvent) bool {
	for _, ch := range b.slackRTM.GetInfo().Channels {
		if ch.ID == event.Channel && ch.Name == b.approvalsChannel {
			return true
		}
	}
	return false
}

func (b *Bot) handleMessage(event *slack.MessageEvent) {
	if event.BotID != "" || event.User == "" || event.SubType == "bot_message" {
		log.WithFields(log.Fields{
			"event_bot_ID":  event.BotID,
			"event_user":    event.User,
			"event_subtype": event.SubType,
		}).Info("handleMessage: ignoring message")
		return
	}

	eventText := strings.Trim(strings.ToLower(event.Text), " \n\r")

	if !b.isBotMessage(event, eventText) {
		return
	}

	eventText = b.trimBot(eventText)
	approval, ok := b.isApproval(event, eventText)
	if ok {
		b.approvalsRespCh <- approval
		return
	}

	// Responses that are just a canned string response
	if responseLines, ok := botEventTextToResponse[eventText]; ok {
		response := strings.Join(responseLines, "\n")
		b.respond(event, formatAsSnippet(response))
		return
	}

	if b.isCommand(event, eventText) {
		b.handleCommand(event, eventText)
		return
	}

	log.WithFields(log.Fields{
		"name":      b.name,
		"bot_id":    b.id,
		"command":   eventText,
		"untrimmed": strings.Trim(strings.ToLower(event.Text), " \n\r"),
	}).Debug("handleMessage: bot couldn't recognise command")
}

func (b *Bot) isCommand(event *slack.MessageEvent, eventText string) bool {
	if staticBotCommands[eventText] {
		return true
	}

	for _, prefix := range dynamicBotCommandPrefixes {
		if strings.HasPrefix(eventText, prefix) {
			return true
		}
	}

	return false
}

func (b *Bot) handleCommand(event *slack.MessageEvent, eventText string) {
	switch eventText {
	case "get deployments":
		log.Info("getting deployments")
		response := b.deploymentsResponse(Filter{})
		b.respond(event, formatAsSnippet(response))
		return
	case "get approvals":
		response := b.approvalsResponse()
		b.respond(event, formatAsSnippet(response))
		return
	}

	// handle dynamic commands
	if strings.HasPrefix(eventText, removeApprovalPrefix) {
		b.respond(event, formatAsSnippet(b.removeApprovalHandler(strings.TrimSpace(strings.TrimPrefix(eventText, removeApprovalPrefix)))))
		return
	}

	log.Info("command not found")
}

func (b *Bot) respond(event *slack.MessageEvent, response string) {
	b.slackRTM.SendMessage(b.slackRTM.NewOutgoingMessage(response, event.Channel))
}

func (b *Bot) isBotMessage(event *slack.MessageEvent, eventText string) bool {
	prefixes := []string{
		b.msgPrefix,
		"keel",
	}

	for _, p := range prefixes {
		if strings.HasPrefix(eventText, p) {
			return true
		}
	}

	// Direct message channels always starts with 'D'
	return strings.HasPrefix(event.Channel, "D")
}

func (b *Bot) trimBot(msg string) string {
	msg = strings.Replace(msg, strings.ToLower(b.msgPrefix), "", 1)
	msg = strings.TrimPrefix(msg, b.name)
	msg = strings.Trim(msg, " :\n")

	return msg
}

func formatAsSnippet(response string) string {
	return "```" + response + "```"
}
