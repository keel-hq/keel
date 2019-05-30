package sql

import (
	"context"
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/keel-hq/keel/types"

	// importing sqlite driver
	_ "github.com/jinzhu/gorm/dialects/sqlite"

	log "github.com/sirupsen/logrus"
)

type SQLStore struct {
	db *gorm.DB
}

type Opts struct {
	DatabaseType string // sqlite3 / postgres
	URI          string // path or conn string
}

func New(opts Opts) (*SQLStore, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()
	db, err := connect(ctx, opts)
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(
		&types.Approval{},
		&types.AuditLog{},
	).Error
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("database migration failed ")
		return nil, err
	}

	return &SQLStore{
		db: db,
	}, nil
}

// Close - closes database connection
func (s *SQLStore) Close() error {
	s.db.Close()
	return nil
}

func (s *SQLStore) OK() bool {
	err := s.db.DB().Ping()
	return err == nil
}

func connect(ctx context.Context, opts Opts) (*gorm.DB, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("sql store startup deadline exceeded")
		default:
			db, err := gorm.Open(opts.DatabaseType, opts.URI)
			if err != nil {
				time.Sleep(1 * time.Second)
				log.WithFields(log.Fields{
					"error": err,
					"uri":   opts.URI,
				}).Warn("sql store connector: can't reach DB, waiting")
				continue
			}

			db.DB().SetMaxOpenConns(40)

			// success
			return db, nil

		}
	}
}
