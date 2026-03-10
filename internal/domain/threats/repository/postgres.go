package repository

import (
	"context"

	domain "admin-service/internal/domain/model"
	"admin-service/internal/domain/threats"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
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
		log: log.Named("threats-repo"),
	}
}

func (r *PostgresRepository) List(ctx context.Context, limit, offset int) ([]*domain.Threat, error) {
	r.log.Debug("List", zap.Int("limit", limit), zap.Int("offset", offset))

	var result []*domain.Threat
	err := r.db.WithContext(ctx).
		Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Find(&result).Error
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Threat, error) {
	r.log.Debug("GetByID", zap.Stringer("threat_id", id))

	var threat domain.Threat
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&threat).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &threat, nil
}

func (r *PostgresRepository) Create(ctx context.Context, threat *domain.Threat) error {
	r.log.Debug("Create", zap.String("title", threat.Title))
	if threat.ID == uuid.Nil {
		threat.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(threat).Error
}

func (r *PostgresRepository) Update(ctx context.Context, threat *domain.Threat) (bool, error) {
	r.log.Debug("Update", zap.Stringer("threat_id", threat.ID))

	tx := r.db.WithContext(ctx).
		Model(&domain.Threat{}).
		Where("id = ?", threat.ID).
		Updates(map[string]interface{}{
			"title":       threat.Title,
			"type":        threat.Type,
			"severity":    threat.Severity,
			"indicator":   threat.Indicator,
			"description": threat.Description,
		})
	if tx.Error != nil {
		return false, tx.Error
	}
	return tx.RowsAffected > 0, nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) (bool, error) {
	r.log.Debug("Delete", zap.Stringer("threat_id", id))

	tx := r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&domain.Threat{})
	if tx.Error != nil {
		return false, tx.Error
	}
	return tx.RowsAffected > 0, nil
}

var _ threats.Repository = (*PostgresRepository)(nil)
