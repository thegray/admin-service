package repository

import (
	"context"
	"time"

	"admin-service/internal/domain/users"

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
		log: log.Named("users-repo"),
	}
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*users.User, error) {
	r.log.Debug("GetByID", zap.Stringer("user_id", id))
	var user users.User
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *PostgresRepository) List(ctx context.Context, limit, offset int) ([]*users.User, error) {
	r.log.Debug("List", zap.Int("limit", limit), zap.Int("offset", offset))

	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var result []*users.User
	err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Find(&result).Error
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *PostgresRepository) Create(ctx context.Context, user *users.User) error {
	r.log.Debug("Create", zap.String("email", user.Email))
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return err
	}
	return nil
}

func (r *PostgresRepository) Update(ctx context.Context, user *users.User) (bool, error) {
	r.log.Debug("Update", zap.Stringer("user_id", user.ID))
	updates := map[string]interface{}{
		"email":     user.Email,
		"password":  user.Password,
		"is_active": user.IsActive,
	}

	tx := r.db.WithContext(ctx).
		Model(&users.User{}).
		Where("id = ? AND deleted_at IS NULL", user.ID).
		Updates(updates)
	if tx.Error != nil {
		return false, tx.Error
	}
	if tx.RowsAffected == 0 {
		return false, nil
	}

	if err := r.db.WithContext(ctx).Where("id = ?", user.ID).First(user).Error; err != nil {
		return true, err
	}
	return true, nil
}

func (r *PostgresRepository) SoftDelete(ctx context.Context, id uuid.UUID) (bool, error) {
	r.log.Debug("SoftDelete", zap.Stringer("user_id", id))
	res := r.db.WithContext(ctx).
		Model(&users.User{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]interface{}{
			"deleted_at": time.Now().UTC(),
			"updated_at": time.Now().UTC(),
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

var _ users.Repository = (*PostgresRepository)(nil)
