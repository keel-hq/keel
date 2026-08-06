package notification

import (
	"context"
	"sync"
	"time"

	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/timeutil"

	log "github.com/sirupsen/logrus"
)

const (
	notifierCheckInterval       = 5 * time.Minute
	notifierMaxBackOff          = 15 * time.Minute
	notifierLockRefreshDuration = time.Minute * 2
	notifierLockDuration        = time.Minute*8 + notifierLockRefreshDuration

	logSenderName = "sender name"
	logNotiName   = "notification name"

	senderQueueSize = 100
)

var (
	sendersM sync.RWMutex
	senders  = make(map[string]Sender)
)

// Config is the configuration for the Notifier service and its registered
// notifiers.
type Config struct {
	Attempts int
	Level    types.Level
	Params   map[string]interface{} `yaml:",inline"`
}

// Sender represents anything that can transmit notifications.
type Sender interface {
	// Configure attempts to initialize the notifier with the provided configuration.
	// It returns whether the notifier is enabled or not.
	Configure(*Config) (bool, error)

	// Send informs the existence of the specified notification.
	Send(event types.EventNotification) error
}

// RegisterSender makes a Sender available by the provided name.
//
// If called twice with the same name, the name is blank, or if the provided
// Sender is nil, this function panics.
func RegisterSender(name string, s Sender) {
	if name == "" {
		panic("notification: could not register a Sender with an empty name")
	}

	if s == nil {
		panic("notification: could not register a nil Sender")
	}

	sendersM.Lock()
	defer sendersM.Unlock()

	if _, dup := senders[name]; dup {
		panic("notification: RegisterSender called twice for " + name)
	}

	log.WithFields(log.Fields{
		"name": name,
	}).Debug("extension.notification: sender registered")

	senders[name] = s
}

// DefaultNotificationSender - default notification sender, manages configuration
type DefaultNotificationSender struct {
	config *Config
	level  types.Level
	ctx    context.Context

	workersM sync.Mutex
	workers  map[string]*notificationWorker
}

type notificationWorker struct {
	queue  chan types.EventNotification
	cancel context.CancelFunc
}

// New - create new sender
func New(ctx context.Context) *DefaultNotificationSender {
	return &DefaultNotificationSender{
		ctx:     ctx,
		workers: make(map[string]*notificationWorker),
	}
}

// Configure - configure is used to register multiple notification senders
func (m *DefaultNotificationSender) Configure(config *Config) (bool, error) {
	m.config = config
	// Configure registered notifiers.
	for senderName, sender := range m.Senders() {
		if configured, err := sender.Configure(config); configured {
			log.WithField(logSenderName, senderName).Info("notificationSender: sender configured")
		} else {
			m.UnregisterSender(senderName)
			if err != nil {
				log.WithError(err).WithField(logSenderName, senderName).Error("could not configure notifier")
			}
		}
	}

	return true, nil
}

// Senders returns the list of the registered Senders.
func (m *DefaultNotificationSender) Senders() map[string]Sender {
	sendersM.RLock()
	defer sendersM.RUnlock()

	ret := make(map[string]Sender)
	for k, v := range senders {
		ret[k] = v
	}

	return ret
}

// Send queues notifications for delivery by each configured sender. Delivery
// happens independently per sender so a slow or broken sender cannot block the
// update pipeline or another sender.
func (m *DefaultNotificationSender) Send(event types.EventNotification) error {
	if event.Level < m.config.Level {
		return nil
	}

	for senderName, sender := range m.Senders() {
		worker := m.worker(senderName, sender)
		select {
		case worker.queue <- event:
		default:
			log.WithFields(log.Fields{
				logNotiName:   event.Name,
				logSenderName: senderName,
			}).Warn("notification queue full, dropping notification")
		}
	}

	return nil
}

func (m *DefaultNotificationSender) worker(senderName string, sender Sender) *notificationWorker {
	m.workersM.Lock()
	defer m.workersM.Unlock()

	if worker, ok := m.workers[senderName]; ok {
		return worker
	}

	ctx, cancel := context.WithCancel(m.ctx)
	worker := &notificationWorker{
		queue:  make(chan types.EventNotification, senderQueueSize),
		cancel: cancel,
	}
	m.workers[senderName] = worker
	go func() {
		for {
			select {
			case event := <-worker.queue:
				m.send(ctx, senderName, sender, event)
			case <-ctx.Done():
				return
			}
		}
	}()

	return worker
}

func (m *DefaultNotificationSender) send(ctx context.Context, senderName string, sender Sender, event types.EventNotification) {
	var attempts int
	var backOff time.Duration
	for {
		if attempts >= m.config.Attempts {
			log.WithFields(log.Fields{
				logNotiName:    event.Name,
				logSenderName:  senderName,
				"max attempts": m.config.Attempts,
			}).Warn("giving up on sending notification: max attempts exceeded")
			return
		}

		if backOff > 0 {
			log.WithFields(log.Fields{
				"duration":     backOff,
				logNotiName:    event.Name,
				logSenderName:  senderName,
				"attempts":     attempts + 1,
				"max attempts": m.config.Attempts,
			}).Info("waiting before retrying to send notification")
			select {
			case <-time.After(backOff):
			case <-ctx.Done():
				return
			}
		}

		if err := sender.Send(event); err != nil {
			log.WithError(err).WithFields(log.Fields{logSenderName: senderName, logNotiName: event.Name}).Error("could not send notification via notifier")
			backOff = timeutil.ExpBackoff(backOff, notifierMaxBackOff)
			attempts++
			continue
		}

		return
	}
}

// UnregisterSender removes a Sender with a particular name from the list.
func (m *DefaultNotificationSender) UnregisterSender(name string) {
	sendersM.Lock()
	delete(senders, name)
	sendersM.Unlock()

	m.workersM.Lock()
	worker := m.workers[name]
	delete(m.workers, name)
	m.workersM.Unlock()
	if worker != nil {
		worker.cancel()
	}
}
