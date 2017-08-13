package approvals

import (
	"context"
	"errors"
	"time"

	"github.com/rusenask/keel/cache"
	"github.com/rusenask/keel/types"
	"github.com/rusenask/keel/util/codecs"

	log "github.com/Sirupsen/logrus"
)

// Manager is used to manage updates
type Manager interface {
	// request approval for deployment/release/etc..
	Create(r *types.Approval) error
	Update(r *types.Approval) error
	Get(provider types.ProviderType, identifier string) (*types.Approval, error)
	List(provider types.ProviderType) ([]*types.Approval, error)
	Delete(provider types.ProviderType, identifier string) error
}

var (
	ErrApprovalAlreadyExists = errors.New("approval already exists")
)

const (
	ApprovalsPrefix = "approvals"
)

// DefaultManager - default manager implementation
type DefaultManager struct {
	// cache is used to store approvals, key example:
	// approvals/<provider name>/<identifier>
	cache      cache.Cache
	serializer codecs.Serializer
}

// New create new instance of default manager
func New(cache cache.Cache, serializer codecs.Serializer) *DefaultManager {
	return &DefaultManager{
		cache:      cache,
		serializer: serializer,
	}
}

func (m *DefaultManager) Create(r *types.Approval) error {
	_, err := m.Get(r.Provider, r.Identifier)
	if err == nil {
		return ErrApprovalAlreadyExists
	}

	bts, err := m.serializer.Encode(r)
	if err != nil {
		return err
	}

	ctx := cache.SetContextExpiration(context.Background(), r.Deadline)

	return m.cache.Put(ctx, getKey(r.Provider, r.Identifier), bts)
}

func (m *DefaultManager) Update(r *types.Approval) error {
	existing, err := m.Get(r.Provider, r.Identifier)
	if err != nil {
		return err
	}

	r.CreatedAt = existing.CreatedAt
	r.UpdatedAt = time.Now()

	bts, err := m.serializer.Encode(r)
	if err != nil {
		return err
	}

	ctx := cache.SetContextExpiration(context.Background(), r.Deadline)

	return m.cache.Put(ctx, getKey(r.Provider, r.Identifier), bts)
}

func (m *DefaultManager) Get(provider types.ProviderType, identifier string) (*types.Approval, error) {
	bts, err := m.cache.Get(context.Background(), getKey(provider, identifier))
	if err != nil {
		return nil, err
	}

	var approval types.Approval
	err = m.serializer.Decode(bts, &approval)
	return &approval, err
}

func (m *DefaultManager) List(provider types.ProviderType) ([]*types.Approval, error) {
	prefix := ""
	if provider != types.ProviderTypeUnknown {
		prefix = provider.String()
	}
	bts, err := m.cache.List(prefix)
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
func (m *DefaultManager) Delete(provider types.ProviderType, identifier string) error {
	return m.cache.Delete(context.Background(), getKey(provider, identifier))
}

func getKey(provider types.ProviderType, identifier string) string {
	return ApprovalsPrefix + "/" + provider.String() + "/" + identifier
}
