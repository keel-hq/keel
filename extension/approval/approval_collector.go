package approval

import (
	"sync"

	"github.com/keel-hq/keel/approvals"

	log "github.com/sirupsen/logrus"
)

var (
	collectorsM sync.RWMutex
	collectors  = make(map[string]Collector)
)

// Collector - generic interface for implementing approval mechanisms
type Collector interface {
	Configure(approvalsManager approvals.Manager) (bool, error)
}

func RegisterCollector(name string, c Collector) {
	if name == "" {
		panic("approval collector:: could not register a Sender with an empty name")
	}

	if c == nil {
		panic("approval collector:: could not register a nil Sender")
	}

	collectorsM.Lock()
	defer collectorsM.Unlock()

	if _, dup := collectors[name]; dup {
		panic("approval collector: RegisterCollector called twice for " + name)
	}

	log.WithFields(log.Fields{
		"name": name,
	}).Info("approval.RegisterCollector: collector registered")

	collectors[name] = c
}

// MainCollector holds all registered collectors
type MainCollector struct {
	approvalsManager approvals.Manager
}

// New - create new sender
func New() *MainCollector {
	return &MainCollector{}
}

// Configure - configure is used to register multiple notification senders
func (m *MainCollector) Configure(approvalsManager approvals.Manager) (bool, error) {
	m.approvalsManager = approvalsManager
	// Configure registered notifiers.
	for collectorName, collector := range m.Collectors() {
		if configured, err := collector.Configure(approvalsManager); configured {
			log.WithFields(log.Fields{
				"name": collectorName,
			}).Info("extension.approval.Configure: collector configured")
		} else {
			m.UnregisterCollector(collectorName)
			if err != nil {
				log.WithFields(log.Fields{
					"name":  collectorName,
					"error": err,
				}).Error("extension.approval.Configure: could not configure collector")
			}
		}
	}

	return true, nil
}

// Collectors returns the list of the registered Collectors.
func (m *MainCollector) Collectors() map[string]Collector {
	collectorsM.RLock()
	defer collectorsM.RUnlock()

	ret := make(map[string]Collector)
	for k, v := range collectors {
		ret[k] = v
	}

	return ret
}

// UnregisterCollector removes a Collector with a particular name from the list.
func (m *MainCollector) UnregisterCollector(name string) {
	collectorsM.Lock()
	defer collectorsM.Unlock()

	delete(collectors, name)
}
