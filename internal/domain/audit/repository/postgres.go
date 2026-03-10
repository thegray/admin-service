package repository

import (
	"context"

	domain "admin-service/internal/domain/model"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PostgresRepository struct {
	db  *gorm.DB
	log *zap.Logger
}

func NewPostgresRepository(db *gorm.DB, log *zap.Logger) *PostgresRepository {
	if db == nil {
		return nil
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &PostgresRepository{
		db:  db,
		log: log.Named("audit-repo"),
	}
}

func (r *PostgresRepository) Insert(ctx context.Context, entry *domain.AuditLog) error {
	if entry == nil {
		return nil
	}
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(entry).Error
}
