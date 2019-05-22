package sql

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jinzhu/gorm"

	"github.com/keel-hq/keel/pkg/store"
	"github.com/keel-hq/keel/types"
)

func (s *SQLStore) CreateApproval(approval *types.Approval) (*types.Approval, error) {

	// generating ID
	if approval.ID == "" {
		approval.ID = uuid.New().String()
	}

	tx := s.db.Begin()
	// Note the use of tx as the database handle once you are within a transaction
	if err := tx.Create(approval).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	tx.Commit()

	return approval, nil
}

func (s *SQLStore) UpdateApproval(approval *types.Approval) error {
	if approval.ID == "" {
		return fmt.Errorf("ID not specified")
	}
	return s.db.Save(approval).Error
}

func (s *SQLStore) GetApproval(q *types.GetApprovalQuery) (*types.Approval, error) {
	var result types.Approval
	var err error
	if q.ID == "" {
		err = s.db.Where("identifier = ? AND archived = ?", q.Identifier, q.Archived).First(&result).Error
	} else {
		err = s.db.Where(&types.Approval{
			ID:         q.ID,
			Identifier: q.Identifier,
			Archived:   q.Archived,
			// Rejected:   q.Rejected,
		}).First(&result).Error
	}
	if err == gorm.ErrRecordNotFound {

		return nil, store.ErrRecordNotFound
	}

	return &result, err
}

func (s *SQLStore) ListApprovals(q *types.GetApprovalQuery) ([]*types.Approval, error) {
	var approvals []*types.Approval
	err := s.db.Order("updated_at desc").Where(&types.Approval{
		Identifier: q.Identifier,
		Archived:   q.Archived,
	}).Find(&approvals).Error
	return approvals, err
}

func (s *SQLStore) DeleteApproval(approval *types.Approval) error {
	if approval.ID == "" {
		return fmt.Errorf("ID not specified")
	}
	return s.db.Delete(approval).Error
}
