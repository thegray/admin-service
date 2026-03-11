package ratelimit

import (
	"context"

	domain "admin-service/internal/domain/model"
	ratelimit "admin-service/internal/domain/rate_limit"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PostgresRepository struct {
	db  *gorm.DB
	log *zap.Logger
}

func NewPostgresRepository(db *gorm.DB, log *zap.Logger) *PostgresRepository {
	if log == nil {
		log = zap.NewNop()
	}
	return &PostgresRepository{
		db:  db,
		log: log.Named("rate-limit-repo"),
	}
}

func (r *PostgresRepository) List(ctx context.Context) ([]*domain.RateLimitPolicy, error) {
	var policies []*domain.RateLimitPolicy
	if err := r.db.WithContext(ctx).
		Preload("Role").
		Find(&policies).Error; err != nil {
		return nil, err
	}
	return policies, nil
}

func (r *PostgresRepository) Upsert(ctx context.Context, policy *domain.RateLimitPolicy) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "scope"},
				{Name: "role_id"},
				{Name: "resource"},
			},
			DoUpdates: clause.AssignmentColumns([]string{"requests_per_minute", "burst"}),
		}).
		Create(policy).Error
}

var _ ratelimit.Repository = (*PostgresRepository)(nil)
