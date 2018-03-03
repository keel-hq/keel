package notification

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/stopper"
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
	}).Info("extension.notification: sender registered")

	senders[name] = s
}

// DefaultNotificationSender - default notification sender, manages configuration
type DefaultNotificationSender struct {
	config  *Config
	stopper *stopper.Stopper
	level   types.Level
}

// New - create new sender
func New(ctx context.Context) *DefaultNotificationSender {
	return &DefaultNotificationSender{
		stopper: stopper.NewStopper(ctx),
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

// Send - send notifications through all configured senders
func (m *DefaultNotificationSender) Send(event types.EventNotification) error {
	if event.Level < m.config.Level {
		return nil
	}

	sendersM.RLock()
	defer sendersM.RUnlock()

	for senderName, sender := range m.Senders() {
		// TODO: move this into goroutine if we have enough senders
		var attempts int
		var backOff time.Duration
		for {
			// Max attempts exceeded.
			if attempts >= m.config.Attempts {
				log.WithFields(log.Fields{
					logNotiName:    event.Name,
					logSenderName:  senderName,
					"max attempts": m.config.Attempts,
				}).Info("giving up on sending notification : max attempts exceeded")
				return fmt.Errorf("failed to send notification, max attempts (%d) reached", m.config.Attempts)
			}

			// Backoff
			if backOff > 0 {
				log.WithFields(log.Fields{
					"duration":     backOff,
					logNotiName:    event.Name,
					logSenderName:  senderName,
					"attempts":     attempts + 1,
					"max attempts": m.config.Attempts,
				}).Info("waiting before retrying to send notification")
				if !m.stopper.Sleep(backOff) {
					return nil
				}
			}

			// Send using the current notifier.
			if err := sender.Send(event); err != nil {
				// Send failed; increase attempts/backoff and retry.
				log.WithError(err).WithFields(log.Fields{logSenderName: senderName, logNotiName: event.Name}).Error("could not send notification via notifier")
				backOff = timeutil.ExpBackoff(backOff, notifierMaxBackOff)
				attempts++
				continue
			}

			// Send has been successful. Go to the next notifier.
			break
		}
	}

	return nil
}

// UnregisterSender removes a Sender with a particular name from the list.
func (m *DefaultNotificationSender) UnregisterSender(name string) {
	sendersM.Lock()
	defer sendersM.Unlock()

	delete(senders, name)
}
