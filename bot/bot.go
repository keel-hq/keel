package bot

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/nlopes/slack"

	"github.com/rusenask/keel/provider/kubernetes"

	log "github.com/Sirupsen/logrus"
)

var (
	botEventTextToResponse = map[string][]string{
		"help": {
			`Here's a list of supported commands`,
			`- "get deployments" -> get a list of keel watched deployments`,
			`- "get deployments all" -> get a list of all deployments`,
			`- "describe deployment <deployment>" -> get details for specified deployment`,
		},
	}

	// static bot commands can be used straight away
	staticBotCommands = map[string]bool{
		"get deployments":     true,
		"get deployments all": true,
	}

	// dynamic bot command prefixes have to be matched
	dynamicBotCommandPrefixes = []string{"describe deployment"}
)

type Bot struct {
	id   string // bot id
	name string // bot name

	users map[string]string

	msgPrefix string

	slackClient *slack.Client
	slackRTM    *slack.RTM

	k8sImplementer kubernetes.Implementer

	ctx context.Context
}

func New(name, token string, k8sImplementer kubernetes.Implementer) *Bot {
	client := slack.New(token)

	return &Bot{
		slackClient:    client,
		k8sImplementer: k8sImplementer,
		name:           name,
	}
}

// Start - start bot
func (b *Bot) Start(ctx context.Context) error {

	// setting root context
	b.ctx = ctx

	users, err := b.slackClient.GetUsers()

	if err != nil {
		panic(err)
	}

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
				fmt.Printf("Presence Change: %v\n", ev)

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

	// All messages past this point are directed to @gopher itself
	if !b.isBotMessage(event, eventText) {
		log.Info("not a bot message")
		return
	}

	eventText = b.trimBot(eventText)

	// Responses that are just a canned string response
	if responseLines, ok := botEventTextToResponse[eventText]; ok {
		response := strings.Join(responseLines, "\n")
		b.respond(event, response)
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
	}).Info("handleMessage: bot couldn't recognise command")

	// b.slackRTM.SendMessage(b.slackRTM.NewOutgoingMessage("bot couldn't recognise command :(", event.Channel))
	// responseLines := botEventTextToResponse["help"]
	// response := strings.Join(responseLines, "\n")
	// b.respond(event, response)
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
	log.Infof("handling command %s", eventText)
	switch eventText {
	case "get deployments":
		log.Info("getting deployments")
		response := b.deploymentsResponse(Filter{})
		b.respond(event, response)
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
