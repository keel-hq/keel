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

	"github.com/slack-go/slack"

	"github.com/keel-hq/keel/bot"
	"github.com/keel-hq/keel/constants"
	"github.com/keel-hq/keel/version"

	log "github.com/sirupsen/logrus"
)

// Bot - main slack bot container
type Bot struct {
	id   string // bot id
	name string // bot name

	users map[string]string

	msgPrefix string

	slackClient *slack.Client
	slackRTM    *slack.RTM

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
		if botName := os.Getenv(constants.EnvSlackBotName); botName != "" {
			b.name = botName
		}

		token := os.Getenv(constants.EnvSlackToken)

		debug, _ := strconv.ParseBool(os.Getenv("DEBUG"))
		client := slack.New(token, slack.OptionDebug(debug))

		b.approvalsChannel = "general"
		if channel := os.Getenv(constants.EnvSlackApprovalsChannel); channel != "" {
			b.approvalsChannel = strings.TrimPrefix(channel, "#")
		}

		b.slackClient = client
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

	for msg := range b.slackRTM.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.HelloEvent:
			// Ignore hello
		case *slack.ConnectedEvent:
			// nothing to do
		case *slack.MessageEvent:
			b.handleMessage(ev)
		case *slack.PresenceChangeEvent:
			// nothing to do
		case *slack.RTMError:
			log.Errorf("Error: %s", ev.Error())
		case *slack.InvalidAuthEvent:
			log.Error("Invalid credentials")
			return fmt.Errorf("invalid credentials")

		default:

			// Ignore other events..
			// fmt.Printf("Unexpected: %v\n", msg.Data)
		}
	}
	return fmt.Errorf("No more events?")
}

func (b *Bot) postMessage(title, message, color string, fields []slack.AttachmentField) error {
	params := slack.NewPostMessageParameters()
	params.Username = b.name
	params.IconURL = b.getBotUserIconURL()

	attachments := []slack.Attachment{
		{
			Fallback: message,
			Color:    color,
			Fields:   fields,
			Footer:   fmt.Sprintf("https://keel.sh %s", version.GetKeelVersion().Version),
			Ts:       json.Number(strconv.Itoa(int(time.Now().Unix()))),
		},
	}

	var mgsOpts []slack.MsgOption

	mgsOpts = append(mgsOpts, slack.MsgOptionPostMessageParameters(params))
	mgsOpts = append(mgsOpts, slack.MsgOptionAttachments(attachments...))

	_, _, err := b.slackClient.PostMessage(b.approvalsChannel, mgsOpts...)
	if err != nil {
		log.WithFields(log.Fields{
			"error":             err,
			"approvals_channel": b.approvalsChannel,
		}).Error("bot.postMessage: failed to send message")
	}
	return err
}

// checking if message was received in approvals channel
func (b *Bot) isApprovalsChannel(event *slack.MessageEvent) bool {

	channel, err := b.slackClient.GetConversationInfo(&slack.GetConversationInfoInput{ChannelID: event.Channel})
	if err != nil {
		if channel != (&slack.Channel{}) && channel != nil {
			if channel.GroupConversation.Name == b.approvalsChannel &&
				channel.GroupConversation.Conversation.IsPrivate {
				return true
			} else {
				log.Errorf("couldn't find amongst private conversations: %s", err)
			}
		}
		log.WithError(err).Errorf("channel with ID %s could not be retrieved", event.Channel)
		return false
	}

	log.Debugf("checking if approvals channel: %s==%s", channel.GroupConversation.Name, b.approvalsChannel)
	if channel.GroupConversation.Name == b.approvalsChannel {
		return true
	}

	log.Debugf("message was received not on approvals channel (%s)", channel.GroupConversation.Name)

	return false
}

func (b *Bot) handleMessage(event *slack.MessageEvent) {
	if event.BotID != "" || event.User == "" || event.SubType == "bot_message" {
		log.WithFields(log.Fields{
			"event_bot_ID":  event.BotID,
			"event_user":    event.User,
			"msg":           event.Text,
			"event_subtype": event.SubType,
		}).Debug("handleMessage: ignoring message")
		return
	}

	eventText := strings.Trim(strings.ToLower(event.Text), " \n\r")

	if !b.isBotMessage(event, eventText) {
		return
	}

	eventText = b.trimBot(eventText)

	approval, ok := bot.IsApproval(event.User, eventText)
	// only accepting approvals from approvals channel
	if ok && b.isApprovalsChannel(event) {
		b.approvalsRespCh <- approval
		return
	} else if ok {
		log.WithFields(log.Fields{
			"received_on":    event.Channel,
			"approvals_chan": b.approvalsChannel,
		}).Warnf("message was received not in approvals channel: %s", event.Channel)
		b.Respond(fmt.Sprintf("please use approvals channel '%s'", b.approvalsChannel), event.Channel)
		return
	}

	b.botMessagesChannel <- &bot.BotMessage{
		Message: eventText,
		User:    event.User,
		Channel: event.Channel,
		Name:    "slack",
	}
}

func (b *Bot) Respond(text string, channel string) {

	// if message is short, replying directly via slack RTM
	if len(text) < 3000 {
		b.slackRTM.SendMessage(b.slackRTM.NewOutgoingMessage(formatAsSnippet(text), channel))
		return
	}

	// longer messages are getting uploaded as files

	f := slack.FileUploadParameters{
		Filename: "keel response",
		Content:  text,
		Filetype: "text",
		Channels: []string{channel},
	}

	_, err := b.slackClient.UploadFile(f)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Respond: failed to send message")
	}
}

func (b *Bot) isBotMessage(event *slack.MessageEvent, eventText string) bool {
	prefixes := []string{
		b.msgPrefix,
		b.name,
		// "kel",
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

func (b *Bot) getBotUserIconURL() string {
	res, err := b.slackClient.GetUserInfo(b.id)
	if err != nil {
		log.WithFields(log.Fields{
			"error":  err,
			"bot_id": b.id,
		}).Error("bot.postMessage: failed to retrieve bot user icon url")
	}

	return res.Profile.ImageOriginal
}
