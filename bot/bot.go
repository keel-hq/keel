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

	"github.com/rusenask/keel/extension/approval"

	"github.com/rusenask/keel/approvals"
	"github.com/rusenask/keel/provider/kubernetes"
	"github.com/rusenask/keel/types"

	log "github.com/Sirupsen/logrus"
)

var (
	botEventTextToResponse = map[string][]string{
		"help": {
			`Here's a list of supported commands`,
			`- "get deployments" -> get a list of all deployments`,
			// `- "get deployments all" -> get a list of all deployments`,
			// `- "describe deployment <deployment>" -> get details for specified deployment`,
		},
	}

	// static bot commands can be used straight away
	staticBotCommands = map[string]bool{
		"get deployments":     true,
		"get deployments all": true,
	}

	// dynamic bot command prefixes have to be matched
	dynamicBotCommandPrefixes = []string{"describe deployment"}

	approvalResponseKeyword = "lgtm"
	rejectResponseKeyword   = "reject"
)

// Bot - main slack bot container
type Bot struct {
	id   string // bot id
	name string // bot name

	users map[string]string

	msgPrefix string

	slackClient *slack.Client
	slackRTM    *slack.RTM

	approvalsCh chan *approvalResponse

	approvalsManager approvals.Manager

	k8sImplementer kubernetes.Implementer

	ctx context.Context
}

// New - create new bot instance
func New(name, token string, k8sImplementer kubernetes.Implementer) *Bot {
	client := slack.New(token)

	bot := &Bot{
		slackClient:    client,
		k8sImplementer: k8sImplementer,
		name:           name,
		approvalsCh:    make(chan *approvalResponse), // don't add buffer to make it blocking
	}

	// register slack bot as approval collector
	approval.RegisterCollector("slack", bot)

	return bot
}

// Configure - sets approval manager during init
func (b *Bot) Configure(approvalsManager approvals.Manager) (bool, error) {
	b.approvalsManager = approvalsManager
	return true, nil
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
				fmt.Printf("Unexpected: %v\n", msg.Data)
			}
		}
	}
}

func (b *Bot) subscribeForApprovals() error {
	approvalsCh, err := b.approvalsManager.Subscribe(b.ctx)
	if err != nil {
		return err
	}

	for {
		select {
		case <-b.ctx.Done():
			return nil
		case a := <-approvalsCh:
			err = b.request(a)
			if err != nil {
				log.WithFields(log.Fields{
					"error":    err,
					"approval": a.Identifier,
				}).Error("bot.subscribeForApprovals: approval request failed")
			}

		}
	}
}

// Request - request approval
func (b *Bot) request(req *types.Approval) error {

	err := b.postMessage(
		"Approval required",
		req.Message,
		types.LevelSuccess.Color(),
		[]slack.AttachmentField{
			slack.AttachmentField{
				Title: "Approval required!",
				Value: req.Message,
				Short: false,
			},
			slack.AttachmentField{
				Title: "Required",
				Value: fmt.Sprint(req.VotesRequired),
				Short: true,
			},
			slack.AttachmentField{
				Title: "Current",
				Value: "0",
				Short: true,
			},
		})
	if err != nil {
		return err
	}

	collected := make(map[string]*approvalResponse)

	voteEnds := time.Now().Add(req.Requirements.Timeout)

	// start waiting for responses
	for {
		select {
		case resp := <-b.approvalsCh:

			// if rejected - ending vote
			if !resp.Approved {
				b.postMessage(
					"Change rejected",
					req.Message,
					types.LevelWarn.Color(),
					[]slack.AttachmentField{
						slack.AttachmentField{
							Title: "Change rejected",
							Value: "Change was manually rejected. Thanks for voting!",
							Short: false,
						},
						slack.AttachmentField{
							Title: "Required",
							Value: fmt.Sprint(req.Requirements.MinimumApprovals),
							Short: true,
						},
						slack.AttachmentField{
							Title: "Current",
							Value: fmt.Sprint(len(collected)),
							Short: true,
						},
					})

				return false, nil
			}

			collected[resp.User] = resp
			if len(collected) >= req.Requirements.MinimumApprovals {
				var voters []string
				for k := range collected {
					voters = append(voters, k)
				}

				b.postMessage(
					"Approval received",
					"All approvals received, thanks for voting!",
					types.LevelSuccess.Color(),
					[]slack.AttachmentField{
						slack.AttachmentField{
							Title: "Update approved!",
							Value: "All approvals received, thanks for voting!",
							Short: false,
						},
						slack.AttachmentField{
							Title: "Required",
							Value: fmt.Sprint(req.Requirements.MinimumApprovals),
							Short: true,
						},
						slack.AttachmentField{
							Title: "Current",
							Value: fmt.Sprint(len(collected)),
							Short: true,
						},
					})

				return true, nil
			}

			// inform about approval and how many votes required
			b.postMessage(
				"Approve received",
				"",
				types.LevelInfo.Color(),
				[]slack.AttachmentField{
					slack.AttachmentField{
						Title: "Approve received",
						Value: "All approvals received, thanks for voting!",
						Short: false,
					},
					slack.AttachmentField{
						Title: "Required",
						Value: fmt.Sprint(req.Requirements.MinimumApprovals),
						Short: true,
					},
					slack.AttachmentField{
						Title: "Current",
						Value: fmt.Sprint(len(collected)),
						Short: true,
					},
					slack.AttachmentField{
						Title: "Vote ends",
						Value: voteEnds.Format("2006/01/02 15:04:05"),
						Short: true,
					},
				})

			continue
		case <-time.After(req.Requirements.Timeout):
			// inform about timeout

			b.postMessage(
				"Vote deadline reached!",
				"",
				types.LevelFatal.Color(),
				[]slack.AttachmentField{
					slack.AttachmentField{
						Title: "Vote deadline reached!",
						Value: "Deadline reached, skipping update.",
						Short: false,
					},
					slack.AttachmentField{
						Title: "Required",
						Value: fmt.Sprint(req.Requirements.MinimumApprovals),
						Short: true,
					},
					slack.AttachmentField{
						Title: "Current",
						Value: fmt.Sprint(len(collected)),
						Short: true,
					},
					slack.AttachmentField{
						Title: "Vote ends",
						Value: voteEnds.Format("2006/01/02 15:04:05"),
						Short: true,
					},
				})

			return false, nil
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

	_, _, err := b.slackClient.PostMessage("general", "", params)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("bot.postMessage: failed to send message")
	}
	return err
}

// approvalResponse - used to track approvals once vote begins
type approvalResponse struct {
	User     string
	Approved bool // can be either approved or rejected
}

func (b *Bot) isApproval(event *slack.MessageEvent, eventText string) (resp *approvalResponse, ok bool) {
	if strings.ToLower(eventText) == approvalResponseKeyword {
		log.WithFields(log.Fields{
			"user":     event.User,
			"username": event.Username,
			"approved": true,
		}).Info("bot.isApproval: approval received")
		return &approvalResponse{
			User:     event.Username,
			Approved: true,
		}, true
	}
	if strings.ToLower(eventText) == rejectResponseKeyword {
		log.WithFields(log.Fields{
			"user":     event.User,
			"username": event.Username,
			"approved": false,
		}).Info("bot.isApproval: approval received")
		return &approvalResponse{
			User:     event.Username,
			Approved: false,
		}, true
	}

	return nil, false
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

	approval, ok := b.isApproval(event, eventText)
	if ok {
		b.approvalsCh <- approval
		return
	}

	if !b.isBotMessage(event, eventText) {
		return
	}

	eventText = b.trimBot(eventText)

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
