package approvals

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/keel-hq/keel/pkg/store"
	"github.com/keel-hq/keel/types"

	log "github.com/sirupsen/logrus"
)

// Manager is used to manage updates
type Manager interface {
	// Subscribe for approval request events, subscriber should provide
	// its name. Indented to be used by extensions that collect
	// approvals
	Subscribe(ctx context.Context) (<-chan *types.Approval, error)

	// SubscribeApproved - is used to get approved events by the manager
	SubscribeApproved(ctx context.Context) (<-chan *types.Approval, error)

	// request approval for deployment/release/etc..
	Create(r *types.Approval) error
	// Update whole approval object
	Update(r *types.Approval) error

	// Increases Approval votes by 1
	Approve(identifier, voter string) (*types.Approval, error)
	// Rejects Approval
	Reject(identifier string) (*types.Approval, error)

	Get(identifier string) (*types.Approval, error)
	List() ([]*types.Approval, error)
	Delete(*types.Approval) error
	Archive(identifier string) error

	StartExpiryService(ctx context.Context) error
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
	// cache cache.Cache

	store store.Store

	// subscriber channels
	channels map[uint32]chan *types.Approval
	index    uint32

	// approved channels
	approvedCh map[uint32]chan *types.Approval

	mu    *sync.Mutex
	subMu *sync.RWMutex
}

type Opts struct {
	Store store.Store
	// Cache cache.Cache
}

// New create new instance of default manager
func New(opts *Opts) *DefaultManager {
	man := &DefaultManager{
		// cache:      opts.Cache,
		store:      opts.Store,
		channels:   make(map[uint32]chan *types.Approval),
		approvedCh: make(map[uint32]chan *types.Approval),
		index:      0,
		mu:         &sync.Mutex{},
		subMu:      &sync.RWMutex{},
	}

	return man
}

// StartExpiryService - starts approval expiry service which deletes approvals
// that already reached their deadline
func (m *DefaultManager) StartExpiryService(ctx context.Context) error {
	ticker := time.NewTicker(60 * time.Minute)
	defer ticker.Stop()
	err := m.expireEntries()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("approvals.StartExpiryService: got error while performing initial expired approvals check")
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			err := m.expireEntries()
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Error("approvals.StartExpiryService: got error while performing routinely expired approvals check")
			}
		}
	}
}

func (m *DefaultManager) expireEntries() error {
	approvals, err := m.store.ListApprovals(&types.GetApprovalQuery{
		Archived: false,
	})
	if err != nil {
		return err
	}

	for _, approval := range approvals {
		if approval.Expired() {
			err = m.Delete(approval)
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
					// "identifier": k,
				}).Error("approvals.expireEntries: failed to delete expired approval")
				continue
			}

			m.addAuditEntry(approval, types.AuditActionApprovalExpired, "")
		}
	}

	return nil
}

// Subscribe - subscribe for approval events
func (m *DefaultManager) Subscribe(ctx context.Context) (<-chan *types.Approval, error) {
	m.subMu.Lock()
	index := atomic.AddUint32(&m.index, 1)
	approvalsCh := make(chan *types.Approval, 10)
	m.channels[index] = approvalsCh
	m.subMu.Unlock()

	go func() {
		for {
			select {
			case <-ctx.Done():
				m.subMu.Lock()

				delete(m.channels, index)

				m.subMu.Unlock()
				return
			}
		}
	}()

	return approvalsCh, nil
}

// SubscribeApproved - subscribe for approved update requests
func (m *DefaultManager) SubscribeApproved(ctx context.Context) (<-chan *types.Approval, error) {
	m.subMu.Lock()
	index := atomic.AddUint32(&m.index, 1)
	approvedCh := make(chan *types.Approval, 10)
	m.approvedCh[index] = approvedCh
	m.subMu.Unlock()

	go func() {
		for {
			select {
			case <-ctx.Done():
				m.subMu.Lock()

				delete(m.approvedCh, index)

				m.subMu.Unlock()
				return
			}
		}
	}()

	return approvedCh, nil
}

func (m *DefaultManager) publishRequest(approval *types.Approval) error {
	m.subMu.RLock()
	defer m.subMu.RUnlock()

	for _, subscriber := range m.channels {
		subscriber <- approval
	}
	return nil
}

func (m *DefaultManager) publishApproved(approval *types.Approval) error {
	m.subMu.RLock()
	defer m.subMu.RUnlock()

	for _, subscriber := range m.approvedCh {
		subscriber <- approval
	}
	return nil
}

// Update - update approval
func (m *DefaultManager) Update(r *types.Approval) error {
	_, err := m.Get(r.Identifier)
	if err != nil {
		return err
	}

	if r.Status() == types.ApprovalStatusApproved {
		err = m.publishApproved(r)
		if err != nil {
			log.WithFields(log.Fields{
				"error":    err,
				"approval": r.Identifier,
				"provider": r.Provider,
			}).Error("approvals.manager: failed to re-submit event after approvals were collected")
		}
	}

	return m.store.UpdateApproval(r)
}

// Approve - increase VotesReceived by 1 and returns updated version
func (m *DefaultManager) Approve(identifier, voter string) (*types.Approval, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, err := m.Get(identifier)
	if err != nil {
		log.WithFields(log.Fields{
			"identifier": identifier,
			"error":      err,
		}).Error("approvals.manager: failed to get")
		return nil, err
	}

	for _, v := range existing.GetVoters() {
		if v == voter {
			// nothing to do, same voter
			return existing, nil
		}
	}

	existing.AddVoter(voter)
	existing.VotesReceived++

	err = m.Update(existing)
	if err != nil {
		log.WithFields(log.Fields{
			"identifier": identifier,
			"error":      err,
		}).Error("approvals.manager: failed to update")
		return nil, err
	}

	m.addAuditEntry(existing, types.AuditActionApprovalApproved, voter)

	log.WithFields(log.Fields{
		"identifier": identifier,
	}).Info("approvals.manager: approved")

	return existing, nil
}

func (m *DefaultManager) addAuditEntry(approval *types.Approval, action string, voter string) {

	entry := &types.AuditLog{
		ID:           uuid.New().String(),
		AccountID:    voter,
		Username:     voter,
		Action:       action,
		ResourceKind: types.AuditResourceKindApproval,
		Identifier:   approval.Identifier,
	}

	entry.SetMetadata(map[string]string{
		"provider":        approval.Provider.String(),
		"approval_id":     approval.ID,
		"new_version":     approval.NewVersion,
		"current_version": approval.CurrentVersion,
		"votes_required":  strconv.Itoa(approval.VotesReceived),
		"votes_received":  strconv.Itoa(approval.VotesReceived),
	})

	_, err := m.store.CreateAuditLog(entry)
	if err != nil {
		log.WithFields(log.Fields{
			"error":  err,
			"module": "approvals",
		}).Error("failed to create audit log")
	}
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

	m.addAuditEntry(existing, types.AuditActionApprovalRejected, "")

	return existing, nil
}

// Get - get specified, not archived approval
func (m *DefaultManager) Get(identifier string) (*types.Approval, error) {

	a, err := m.store.GetApproval(&types.GetApprovalQuery{
		Identifier: identifier,
		Archived:   false,
	})
	if err != nil {
		return nil, err
	}

	// if it's archived, don't display it
	if a.Archived {
		return nil, store.ErrRecordNotFound
	}

	return a, nil
}

// List - list not archived approvals (for expiration service)
func (m *DefaultManager) List() ([]*types.Approval, error) {
	approvals, err := m.store.ListApprovals(&types.GetApprovalQuery{
		Archived: false,
	})
	return approvals, err
}

// Delete - delete specified approval
func (m *DefaultManager) Delete(approval *types.Approval) error {
	existing, err := m.store.GetApproval(&types.GetApprovalQuery{
		ID: approval.ID,
	})
	if err != nil {
		return err
	}

	m.addAuditEntry(existing, types.AuditActionDeleted, "")

	return m.store.DeleteApproval(existing)
}

func (m *DefaultManager) Archive(identifier string) error {
	existing, err := m.Get(identifier)
	if err != nil {
		return fmt.Errorf("approval not found: %s", err)
	}
	existing.Archived = true

	m.addAuditEntry(existing, types.AuditActionApprovalArchived, "")

	return m.store.UpdateApproval(existing)
}

// Create - creates new approval request and publishes to all subscribers
func (m *DefaultManager) Create(r *types.Approval) error {
	_, err := m.Get(r.Identifier)
	if err == nil {
		return ErrApprovalAlreadyExists
	}

	r.CreatedAt = time.Now()
	r.UpdatedAt = time.Now()

	created, err := m.store.CreateApproval(r)
	if err != nil {
		return fmt.Errorf("failed to create approval: %s", err)
	}

	return m.publishRequest(created)
}

func getKey(identifier string) string {
	return ApprovalsPrefix + "/" + identifier
}
