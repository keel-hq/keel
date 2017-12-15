package bot

import (
	"sync"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/provider/kubernetes"

	log "github.com/Sirupsen/logrus"
)

type Bot interface {
	Run(k8sImplementer kubernetes.Implementer, approvalsManager approvals.Manager) (teardown func(), err error)
}

type BotFactory func(k8sImplementer kubernetes.Implementer, approvalsManager approvals.Manager) (teardown func(), err error)
type teardown func()

// type Teardown func()

var (
	botsM     sync.RWMutex
	bots      = make(map[string]BotFactory)
	teardowns = make(map[string]teardown)
)

func RegisterBot(name string, b BotFactory) {
	log.Debug("bot.RegisterBot")
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

type DefaultBot struct {
}

func Run(k8sImplementer kubernetes.Implementer, approvalsManager approvals.Manager) {
	log.Debugf("bot.Run(): %#v\n", bots)
	for botName, runner := range bots {
		log.Debugf("bot.Run(): run bot %s\n", botName)
		teardownBot, err := runner(k8sImplementer, approvalsManager)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatalf("main: failed to setup %s bot\n", botName)
		} else {
			teardowns[botName] = teardownBot
		}
	}
	// return teardowns
}

func Stop() {
	log.Debug("bot.Stop()")
	for botName, teardown := range teardowns {
		log.Debugf("Teardown %s bot\n", botName)
		teardown()
	}
}

// Senders returns the list of the registered Senders.
func Bots() map[string]BotFactory {
	botsM.RLock()
	defer botsM.RUnlock()
	// bots  = make(map[string]BotFactory)
	ret := make(map[string]BotFactory)
	for k, v := range bots {
		ret[k] = v
	}

	return ret
}
