package slack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nlopes/slack"

	"github.com/keel-hq/keel/bot"
	"github.com/keel-hq/keel/constants"

	log "github.com/sirupsen/logrus"
)

// SlackImplementer - implementes slack HTTP functionality, used to
// send messages with attachments
type SlackImplementer interface {
	PostMessage(channel, text string, params slack.PostMessageParameters) (string, string, error)
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

	approvalsChannel string // slack approvals channel name

	ctx                context.Context
	botMessagesChannel chan *bot.BotMessage
	approvalsRespCh    chan *bot.ApprovalResponse
}

func init() {
	bot.RegisterBot("slack", &Bot{})
}

func (b *Bot) Configure(approvalsRespCh chan *bot.ApprovalResponse, botMessagesChannel chan *bot.BotMessage) bool {
	if os.Getenv(constants.EnvSlackToken) != "" {

		b.name = "keel"
		if os.Getenv(constants.EnvSlackBotName) != "" {
			b.name = os.Getenv(constants.EnvSlackBotName)
		}

		token := os.Getenv(constants.EnvSlackToken)
		client := slack.New(token)

		b.approvalsChannel = "general"
		if os.Getenv(constants.EnvSlackApprovalsChannel) != "" {
			b.approvalsChannel = os.Getenv(constants.EnvSlackApprovalsChannel)
		}

		b.slackClient = client
		b.slackHTTPClient = client
		b.approvalsRespCh = approvalsRespCh
		b.botMessagesChannel = botMessagesChannel

		return true
	}
	log.Info("bot.slack.Configure(): Slack approval bot is not configured")
	return false
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

	go b.startInternal()

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

	// only accepting approvals from approvals channel
	if b.isApprovalsChannel(event) {
		approval, ok := bot.IsApproval(event.User, eventText)
		if ok {
			b.approvalsRespCh <- approval
			return
		}
	}

	b.botMessagesChannel <- &bot.BotMessage{
		Message: eventText,
		User:    event.User,
		Channel: event.Channel,
		Name:    "slack",
	}
	return
}

func (b *Bot) Respond(text string, channel string) {
	b.slackRTM.SendMessage(b.slackRTM.NewOutgoingMessage(formatAsSnippet(text), channel))
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
