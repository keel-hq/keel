package approvals

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rusenask/keel/cache"
	"github.com/rusenask/keel/provider"
	"github.com/rusenask/keel/types"
	"github.com/rusenask/keel/util/codecs"

	log "github.com/Sirupsen/logrus"
)

// Manager is used to manage updates
type Manager interface {
	// Subscribe for approval request events, subscriber should provide
	// its name
	Subscribe(ctx context.Context) (<-chan *types.Approval, error)

	// request approval for deployment/release/etc..
	Create(r *types.Approval) error
	// Update whole approval object
	Update(r *types.Approval) error

	// Increases Approval votes by 1
	Approve(identifier string) (*types.Approval, error)
	// Rejects Approval
	Reject(identifier string) (*types.Approval, error)

	Get(identifier string) (*types.Approval, error)
	List() ([]*types.Approval, error)
	Delete(identifier string) error
}

// Approvals related errors
var (
	ErrApprovalAlreadyExists = errors.New("approval already exists")
)

// Approvals cache prefix
const (
	ApprovalsPrefix = "approvals"
)

// DefaultManager - default manager implementation
type DefaultManager struct {
	// cache is used to store approvals, key example:
	// approvals/<provider name>/<identifier>
	cache      cache.Cache
	serializer codecs.Serializer

	// providers are used to re-submit event
	// when all approvals are collected
	providers provider.Providers

	// subscriber channels
	channels map[uint64]chan *types.Approval
	index    uint64

	mu *sync.Mutex
}

// New create new instance of default manager
func New(cache cache.Cache, serializer codecs.Serializer, providers provider.Providers) *DefaultManager {
	man := &DefaultManager{
		cache:      cache,
		serializer: serializer,
		providers:  providers,
		channels:   make(map[uint64]chan *types.Approval),
		index:      0,
		mu:         &sync.Mutex{},
	}

	return man
}

// Subscribe - subscribe for approval events
func (m *DefaultManager) Subscribe(ctx context.Context) (<-chan *types.Approval, error) {
	m.mu.Lock()
	m.mu.Unlock()

	index := atomic.AddUint64(&m.index, 1)
	approvalsCh := make(chan *types.Approval, 10)
	m.channels[index] = approvalsCh
	m.mu.Unlock()

	go func() {
		for {
			select {
			case <-ctx.Done():
				m.mu.Lock()

				delete(m.channels, index)

				m.mu.Unlock()
				return
			}
		}
	}()

	return approvalsCh, nil
}

func (m *DefaultManager) publish(approval *types.Approval) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, subscriber := range m.channels {
		subscriber <- approval
	}
	return nil
}

// Create - creates new approval request and publishes to all subscribers
func (m *DefaultManager) Create(r *types.Approval) error {
	_, err := m.Get(r.Identifier)
	if err == nil {
		return ErrApprovalAlreadyExists
	}

	bts, err := m.serializer.Encode(r)
	if err != nil {
		return err
	}

	ctx := cache.SetContextExpiration(context.Background(), r.Deadline)

	err = m.cache.Put(ctx, getKey(r.Identifier), bts)
	if err != nil {
		return err
	}

	return m.publish(r)

}

func (m *DefaultManager) Update(r *types.Approval) error {
	existing, err := m.Get(r.Identifier)
	if err != nil {
		return err
	}

	r.CreatedAt = existing.CreatedAt
	r.UpdatedAt = time.Now()

	bts, err := m.serializer.Encode(r)
	if err != nil {
		return err
	}

	if r.Status() == types.ApprovalStatusApproved {
		err = m.providers.Submit(*r.Event)
		if err != nil {
			log.WithFields(log.Fields{
				"error":    err,
				"approval": r.Identifier,
				"provider": r.Provider,
			}).Error("approvals.manager: failed to re-submit event after approvals were collected")
		}
	}

	ctx := cache.SetContextExpiration(context.Background(), r.Deadline)
	return m.cache.Put(ctx, getKey(r.Identifier), bts)
}

// Approve - increase VotesReceived by 1 and returns updated version
func (m *DefaultManager) Approve(identifier string) (*types.Approval, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, err := m.Get(identifier)
	if err != nil {
		return nil, err
	}

	existing.VotesReceived++

	err = m.Update(existing)
	if err != nil {
		return nil, err
	}

	return existing, nil
}

// Reject - rejects approval (marks rejected=true), approval will not be valid even if it
// collects required votes
func (m *DefaultManager) Reject(identifier string) (*types.Approval, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, err := m.Get(identifier)
	if err != nil {
		return nil, err
	}

	existing.Rejected = true

	err = m.Update(existing)
	if err != nil {
		return nil, err
	}

	return existing, nil
}

func (m *DefaultManager) Get(identifier string) (*types.Approval, error) {
	bts, err := m.cache.Get(context.Background(), getKey(identifier))
	if err != nil {
		return nil, err
	}

	var approval types.Approval
	err = m.serializer.Decode(bts, &approval)
	return &approval, err
}

func (m *DefaultManager) List() ([]*types.Approval, error) {
	bts, err := m.cache.List(ApprovalsPrefix)
	if err != nil {
		return nil, err
	}

	var approvals []*types.Approval
	for _, v := range bts {
		var approval types.Approval
		err = m.serializer.Decode(v, &approval)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("approvals.manager: failed to decode payload")
			continue
		}
		approvals = append(approvals, &approval)
	}
	return approvals, nil

}
func (m *DefaultManager) Delete(identifier string) error {
	return m.cache.Delete(context.Background(), getKey(identifier))
}

func getKey(identifier string) string {
	return ApprovalsPrefix + "/" + identifier
}
