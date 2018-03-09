package bot

import (
	"context"
	"strings"
	"sync"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/provider/kubernetes"
	"github.com/keel-hq/keel/types"

	log "github.com/sirupsen/logrus"
)

const (
	RemoveApprovalPrefix = "rm approval"
)

var (
	BotEventTextToResponse = map[string][]string{
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
	dynamicBotCommandPrefixes = []string{RemoveApprovalPrefix}

	ApprovalResponseKeyword = "approve"
	RejectResponseKeyword   = "reject"
)

type Bot interface {
	Configure(approvalsRespCh chan *ApprovalResponse, botMessagesChannel chan *BotMessage) bool
	Start(ctx context.Context) error
	Respond(text string, channel string)
	RequestApproval(req *types.Approval) error
	ReplyToApproval(approval *types.Approval) error
}

type teardown func()
type BotMessageResponder func(response string, channel string)

var (
	botsM     sync.RWMutex
	bots      = make(map[string]Bot)
	teardowns = make(map[string]teardown)
)

// BotMessage represents abstract container for any bot Message
// add here more fields if you needed for a new bot implementation
type BotMessage struct {
	Message string
	User    string
	Name    string
	Channel string
}

// ApprovalResponse - used to track approvals once vote begins
type ApprovalResponse struct {
	User   string
	Status types.ApprovalStatus
	Text   string
}

// BotManager holds approvalsManager and k8sImplementer for every bot
type BotManager struct {
	approvalsManager   approvals.Manager
	k8sImplementer     kubernetes.Implementer
	botMessagesChannel chan *BotMessage
	approvalsRespCh    chan *ApprovalResponse
}

// RegisterBot makes a bot implementation available by the provided name.
func RegisterBot(name string, b Bot) {
	if name == "" {
		panic("bot: could not register a BotFactory with an empty name")
	}

	if b == nil {
		panic("bot: could not register a nil Bot interface")
	}

	botsM.Lock()
	defer botsM.Unlock()

	if _, dup := bots[name]; dup {
		panic("bot: RegisterBot called twice for " + name)
	}

	log.WithFields(log.Fields{
		"name": name,
	}).Info("bot: registered")

	bots[name] = b
}

// Run all implemented bots
func Run(k8sImplementer kubernetes.Implementer, approvalsManager approvals.Manager) {
	bm := &BotManager{
		approvalsManager:   approvalsManager,
		k8sImplementer:     k8sImplementer,
		approvalsRespCh:    make(chan *ApprovalResponse), // don't add buffer to make it blocking
		botMessagesChannel: make(chan *BotMessage),
	}
	for botName, bot := range bots {
		configured := bot.Configure(bm.approvalsRespCh, bm.botMessagesChannel)
		if configured {
			bm.SetupBot(botName, bot)
		} else {
			log.Errorf("bot.Run(): can not get configuration for bot [%s]", botName)
		}
	}
}

func (bm *BotManager) SetupBot(botName string, bot Bot) {
	ctx, cancel := context.WithCancel(context.Background())
	err := bot.Start(ctx)
	if err != nil {
		cancel()
		log.WithFields(log.Fields{
			"error": err,
		}).Fatalf("main: failed to setup %s bot\n", botName)
	} else {
		// store cancelling context for each bot
		teardowns[botName] = func() { cancel() }

		go bm.ProcessBotMessages(ctx, bot.Respond)
		go bm.ProcessApprovalResponses(ctx, bot.ReplyToApproval)
		go bm.SubscribeForApprovals(ctx, bot.RequestApproval)
	}
}

func (bm *BotManager) ProcessBotMessages(ctx context.Context, respond BotMessageResponder) {
	for {
		select {
		case <-ctx.Done():
			return
		case message := <-bm.botMessagesChannel:
			response := bm.handleBotMessage(message)
			if response != "" {
				respond(response, message.Channel)
			}
		}
	}
}

func Stop() {
	for botName, teardown := range teardowns {
		log.Infof("Teardown %s bot", botName)
		teardown()
		UnregisterBot(botName)
	}
}

func IsBotCommand(eventText string) bool {
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

func (bm *BotManager) handleCommand(eventText string) string {
	switch eventText {
	case "get deployments":
		log.Info("HandleCommand: getting deployments")
		return DeploymentsResponse(Filter{}, bm.k8sImplementer)
	case "get approvals":
		log.Info("HandleCommand: getting approvals")
		return ApprovalsResponse(bm.approvalsManager)
	}

	// handle dynamic commands
	if strings.HasPrefix(eventText, RemoveApprovalPrefix) {
		id := strings.TrimSpace(strings.TrimPrefix(eventText, RemoveApprovalPrefix))
		return RemoveApprovalHandler(id, bm.approvalsManager)
	}

	log.Infof("bot.HandleCommand(): command [%s] not found", eventText)
	return ""
}

func (bm *BotManager) handleBotMessage(m *BotMessage) string {
	command := m.Message

	if responseLines, ok := BotEventTextToResponse[command]; ok {
		return strings.Join(responseLines, "\n")
	}

	if IsBotCommand(command) {
		return bm.handleCommand(command)
	}

	log.WithFields(log.Fields{
		"user":    m.User,
		"bot":     m.Name,
		"command": command,
	}).Debug("handleMessage: bot couldn't recognise command")

	return ""
}

// UnregisterBot removes a Sender with a particular name from the list.
func UnregisterBot(name string) {
	botsM.Lock()
	defer botsM.Unlock()

	delete(bots, name)
}
