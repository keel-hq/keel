package slack

import (
	"context"
	"errors"
	"fmt"
	"github.com/keel-hq/keel/bot"
	"github.com/keel-hq/keel/constants"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
	"os"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

// Bot - main slack bot container
type Bot struct {
	id   string // bot id
	name string // bot name

	users map[string]string

	msgPrefix string

	slackSocket *socketmode.Client

	// the approval channel-name, provided by the bot configuration
	approvalsChannel string

	// the identifier of the approval channel, this is retrieved when the bot is starting
	approvalChannelId string

	ctx                context.Context
	botMessagesChannel chan *bot.BotMessage
	approvalsRespCh    chan *bot.ApprovalResponse
}

func init() {
	bot.RegisterBot("slack", &Bot{})
}

func (b *Bot) Configure(approvalsRespCh chan *bot.ApprovalResponse, botMessagesChannel chan *bot.BotMessage) bool {
	botToken := os.Getenv(constants.EnvSlackBotToken)

	if !strings.HasPrefix(botToken, "xoxb-") {
		log.Infof("bot.slack.Configure(): %s must have the prefix \"xoxb-\", skip bot configuration.", constants.EnvSlackBotToken)
		return false
	}

	appToken := os.Getenv(constants.EnvSlackAppToken)
	if !strings.HasPrefix(appToken, "xapp-") {
		log.Infof("bot.slack.Configure(): %s must have the previf \"xapp-\".", constants.EnvSlackAppToken)
		return false
	}

	botName, botNameConfigured := os.LookupEnv(constants.EnvSlackBotName)
	if !botNameConfigured {
		botName = "keel"
	}
	b.name = botName

	channel, channelConfigured := os.LookupEnv(constants.EnvSlackApprovalsChannel)
	if !channelConfigured {
		channel = "general"
	}

	b.approvalsChannel = strings.TrimPrefix(channel, "#")

	log.Debugf("Configuring slack with approval channel '%s' and bot '%s'", b.approvalsChannel, b.name)

	debug, _ := strconv.ParseBool(os.Getenv("DEBUG"))
	api := slack.New(
		botToken,
		slack.OptionDebug(debug),
		slack.OptionAppLevelToken(appToken),
	)

	client := socketmode.New(
		api,
		socketmode.OptionDebug(debug),
	)

	b.slackSocket = client
	b.approvalsRespCh = approvalsRespCh
	b.botMessagesChannel = botMessagesChannel

	return true
}

// Start - start bot
func (b *Bot) Start(ctx context.Context) error {
	// setting root context
	b.ctx = ctx

	users, err := b.slackSocket.GetUsers()
	if err != nil {
		return err
	}

	b.users = map[string]string{}

	// -- retrieve the bot user identifier from the bot name
	var foundBots []string

	for _, user := range users {
		if user.IsBot {
			foundBots = append(foundBots, user.Name)
			if user.Name == b.name {
				b.id = user.ID
				break
			}
		}
	}
	if b.id == "" {
		return errors.New("could not find bot in the list of names, check if the bot is called \"" + b.name + "\", found bots: " + strings.Join(foundBots[:], ", "))
	}

	// -- mentions and direct messages start with this message prefix. It is used from trimming the messages
	b.msgPrefix = strings.ToLower("<@" + b.id + ">")

	// -- retrieve the channel identifier from the approval channel name
	b.approvalChannelId, err = b.findChannelId(b.approvalsChannel)
	if err != nil {
		return err
	}

	go b.listenForSocketEvents()

	return nil
}

func (b *Bot) findChannelId(channelName string) (string, error) {
	var channelId string
	var cursor string

	// -- while the channel is not found, fetch pages
	for channelId == "" {
		channels, nextCursor, err := b.slackSocket.GetConversationsForUser(&slack.GetConversationsForUserParameters{ExcludeArchived: true, Cursor: cursor})
		if err != nil {
			return "", err
		}

		for _, channel := range channels {
			if channel.Name == channelName {
				channelId = channel.ID
				break
			}
		}

		// -- channel not found on this page, check if there are more pages
		if nextCursor == "" {
			break
		}

		// -- continue to the next page
		cursor = nextCursor
	}

	if channelId == "" {
		return "", errors.New("Unable to retrieve the channel named \"" + channelName + "\". Check that the bot is invited to that channel and define the proper scope in the Slack app settings.")
	} else {
		return channelId, nil
	}
}

func (b *Bot) listenForSocketEvents() error {
	go func() {
		for evt := range b.slackSocket.Events {
			switch evt.Type {
			case socketmode.EventTypeConnecting:
				log.Info("Connecting to Slack with Socket Mode...")
			case socketmode.EventTypeConnectionError:
				if "missing_scope" == evt.Data {
					log.Error("The application token is missing scopes, verify to provide an application token with the scope 'connections:write'", evt.Data)
				} else {
					log.Error("Connection failed. Retrying later... ", evt.Data)
				}
			case socketmode.EventTypeConnected:
				log.Info("Connected to Slack with Socket Mode.")
			case socketmode.EventTypeInvalidAuth:
				log.Error("Invalid authentication parameter provided.", evt.Data)
			case socketmode.EventTypeDisconnect:
				log.Info("Disconnected from Slack socket.")
			case socketmode.EventTypeIncomingError:
				log.Error("An error occurred while processing an incoming event.", evt.Data)
			case socketmode.EventTypeErrorBadMessage:
				log.Error("Bad message error.", evt.Data)
			case socketmode.EventTypeErrorWriteFailed:
				log.Error("Error while responding to a message.", evt.Data)
			case socketmode.EventTypeSlashCommand:
				// ignore slash commands
			case socketmode.EventTypeEventsAPI:
				// The bot can receive mention events only when the bot has the Event Subscriptions enabled
				// AND has a subscription to "app_mention" events
				eventsAPIEvent, isEventApiEvent := evt.Data.(slackevents.EventsAPIEvent)
				if !isEventApiEvent {
					continue
				}

				innerEvent := eventsAPIEvent.InnerEvent
				mentionEvent, isAppMentionEvent := innerEvent.Data.(*slackevents.AppMentionEvent)
				if isAppMentionEvent && eventsAPIEvent.Type == slackevents.CallbackEvent {
					// -- the bot was mentioned in a message, try to process the command
					b.handleMentionEvent(mentionEvent)
					b.slackSocket.Ack(*evt.Request)
				}
			case socketmode.EventTypeInteractive:
				callback, isInteractionCallback := evt.Data.(slack.InteractionCallback)
				if !isInteractionCallback {
					log.Debugf("Ignoring Event %+v\n", evt)
					continue
				}

				if callback.Type == slack.InteractionTypeBlockActions {
					if (len(callback.ActionCallback.BlockActions)) == 0 {
						log.Error("No block actions found")
						continue
					}

					// callback.ResponseURL
					blockAction := callback.ActionCallback.BlockActions[0]
					b.handleAction(callback.User.ID, blockAction)
					b.slackSocket.Ack(*evt.Request)
				}
			}
		}
	}()

	b.slackSocket.Run()

	return fmt.Errorf("No more events?")
}

// handleMentionEvent - Handle a mention event. The bot will only receive its own mention event. No need to check that the message is for him.
func (b *Bot) handleMentionEvent(event *slackevents.AppMentionEvent) {
	if event.BotID != "" || event.User == "" {
		log.WithFields(log.Fields{
			"event_bot_ID": event.BotID,
			"event_user":   event.User,
			"msg":          event.Text,
		}).Debug("handleMessage: ignoring message")
		return
	}

	// -- clean the text message to have only the action (approve or reject) followed by the resource identifier
	// -- (e.g. approve k8s/project/repo:1.2.3)
	eventText := strings.Trim(strings.ToLower(event.Text), " \n\r")

	eventText = b.trimBotName(eventText)

	// -- first, try to handle the message as an approval response
	approval, isAnApprovalResponse := bot.IsApproval(event.User, eventText)

	if isAnApprovalResponse && b.isEventFromApprovalsChannel(event) {
		// -- the message is processed by bot\approvals.go in ProcessApprovalResponses
		b.approvalsRespCh <- approval
		return
	} else if isAnApprovalResponse {
		log.WithFields(log.Fields{
			"received_on":    event.Channel,
			"approvals_chan": b.approvalsChannel,
		}).Warnf("The message was not received in the approval channel: %s", event.Channel)
		b.Respond(fmt.Sprintf("Please use approvals channel '%s'", b.approvalsChannel), event.Channel)
		return
	}

	// -- the message is not an approval response, try to handle the message as a generic bot command
	b.botMessagesChannel <- &bot.BotMessage{
		Message: eventText,
		User:    event.User,
		Channel: event.Channel,
		Name:    "slack",
	}
}

func (b *Bot) trimBotName(msg string) string {
	msg = strings.Replace(msg, strings.ToLower(b.msgPrefix), "", 1)
	msg = strings.TrimPrefix(msg, b.name)
	msg = strings.Trim(msg, " :\n")

	return msg
}

// isEventFromApprovalsChannel - checking if message was received in approvals channel
func (b *Bot) isEventFromApprovalsChannel(event *slackevents.AppMentionEvent) bool {
	if b.approvalChannelId == event.Channel {
		return true
	} else {
		log.Debug("Message was not received on the approvals channel, ignoring")
		return false
	}
}

// handleAction - Handle an action performed by using the slack block action feature.
// The bot will only receive events coming from its own action blocks. Block action can only be used to approve
// or reject an approval request (other commands should be managed by user bot mentions).
func (b *Bot) handleAction(username string, blockAction *slack.BlockAction) {
	eventText := fmt.Sprintf("%s %s", blockAction.ActionID, blockAction.Value)
	approval, ok := bot.IsApproval(username, eventText)

	if !ok {
		// -- only react to approval requests (approve or reject actions)
		log.WithFields(log.Fields{
			"action_user":  username,
			"action_id":    blockAction.ActionID,
			"action_value": blockAction.Value,
		}).Debug("handleAction: ignoring action, clicked on unknown button")
		return
	}

	b.approvalsRespCh <- approval
}

// postApprovalMessageBlock - effectively post a message to the approval channel
func (b *Bot) postApprovalMessageBlock(approvalId string, blocks slack.Blocks) error {
	channelID := b.approvalsChannel
	_, _, err := b.slackSocket.PostMessage(
		channelID,
		slack.MsgOptionBlocks(blocks.BlockSet...),
		createApprovalMetadata(approvalId),
	)

	return err
}

// Respond - This method sent the text message to the provided channel
func (b *Bot) Respond(text string, channel string) {
	// if message is short, replying directly via socket
	if len(text) < 3000 {
		b.slackSocket.SendMessage(channel, slack.MsgOptionText(formatAsSnippet(text), true))
		return
	}

	// longer messages are getting uploaded as files
	f := slack.FileUploadParameters{
		Filename: "keel response",
		Content:  text,
		Filetype: "text",
		Channels: []string{channel},
	}

	_, err := b.slackSocket.UploadFile(f)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Respond: failed to send message")
	}
}

// upsertApprovalMessage - update the approval message that was sent for the given resource identifier (deployment/default/wd:0.0.15).
// if the message is not found in the approval channel it will be created. That way even it the message is deleted,
// we will see the approval status
func (b *Bot) upsertApprovalMessage(approvalId string, blocks slack.Blocks) {
	// Retrieve the message history
	historyParams := &slack.GetConversationHistoryParameters{
		ChannelID:          b.approvalChannelId,
		Limit:              250,
		IncludeAllMetadata: true,
	}

	history, err := b.slackSocket.GetConversationHistory(historyParams)
	if err != nil {
		log.Debugf("Unable to get the conversation history to edit the message, post new one: %v", err)
		b.postApprovalMessageBlock(approvalId, blocks)
	}

	// Find the message to update; the channel id and the message timestamp is the identifier of a message for slack
	var messageTs string
	for _, message := range history.Messages {
		if isMessageOfApprovalRequest(message, approvalId) {
			messageTs = message.Timestamp
			break // Found the message
		}
	}

	if messageTs == "" {
		log.Debug("Unable to find the approval message for the identifier. Post a new message instead")
		b.postApprovalMessageBlock(approvalId, blocks)
		return
	} else {
		b.slackSocket.UpdateMessage(
			b.approvalChannelId,
			messageTs,
			slack.MsgOptionBlocks(blocks.BlockSet...),
			slack.MsgOptionAsUser(true),
			createApprovalMetadata(approvalId),
		)
	}
}

// isMessageOfApprovalRequest - Check whether the given message is the approval message sent for the given approval identifier.
// Helps to identify the interactive message corresponding to the approval request in order to update the latest status of the approval.
// returns true if it is, false otherwise.
func isMessageOfApprovalRequest(message slack.Message, approvalId string) bool {
	if message.Metadata.EventType != "approval" {
		return false
	}

	if message.Metadata.EventPayload == nil {
		return false
	}
	approvalID, ok := message.Metadata.EventPayload["approval_id"].(string)
	if !ok {
		return false
	}

	return approvalID == approvalId
}

// createApprovalMetadata - create message metadata, the metadata includes the approval identifier.
// That way, it is possible to identify clearly the approval message for a given approval request when looking into the
// history.
func createApprovalMetadata(approvalId string) slack.MsgOption {
	return slack.MsgOptionMetadata(
		slack.SlackMetadata{
			EventType: "approval",
			EventPayload: map[string]interface{}{
				"approval_id": approvalId,
			},
		},
	)
}

func formatAsSnippet(response string) string {
	return "```" + response + "```"
}
