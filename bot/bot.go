package bot

import (
	"sync"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/provider/kubernetes"
	"github.com/keel-hq/keel/types"

	log "github.com/Sirupsen/logrus"
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
	StaticBotCommands = map[string]bool{
		"get deployments": true,
		"get approvals":   true,
	}

	// dynamic bot command prefixes have to be matched
	DynamicBotCommandPrefixes = []string{RemoveApprovalPrefix}

	ApprovalResponseKeyword = "approve"
	RejectResponseKeyword   = "reject"
)

type Bot interface {
	Run(k8sImplementer kubernetes.Implementer, approvalsManager approvals.Manager) (teardown func(), err error)
}

type BotFactory func(k8sImplementer kubernetes.Implementer, approvalsManager approvals.Manager) (teardown func(), err error)
type teardown func()

var (
	botsM     sync.RWMutex
	bots      = make(map[string]BotFactory)
	teardowns = make(map[string]teardown)
)

// ApprovalResponse - used to track approvals once vote begins
type ApprovalResponse struct {
	User   string
	Status types.ApprovalStatus
	Text   string
}

// RegisterBot makes a BotRunner available by the provided name.
func RegisterBot(name string, b BotFactory) {
	if name == "" {
		panic("bot: could not register a BotFactory with an empty name")
	}

	if b == nil {
		panic("bot: could not register a nil BotFactory")
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

func Run(k8sImplementer kubernetes.Implementer, approvalsManager approvals.Manager) {
	for botName, runner := range bots {
		teardownBot, err := runner(k8sImplementer, approvalsManager)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatalf("main: failed to setup %s bot\n", botName)
		} else {
			log.Debugf(">>> Run [%s] bot", botName)
			teardowns[botName] = teardownBot
		}
	}
}

func Stop() {
	for botName, teardown := range teardowns {
		log.Infof("Teardown %s bot\n", botName)
		teardown()
	}
}
