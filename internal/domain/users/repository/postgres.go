package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	domain "admin-service/internal/domain/model"
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

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	r.log.Debug("GetByID", zap.Stringer("user_id", id))
	var user domain.User
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

func (r *PostgresRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return nil, nil
	}

	r.log.Debug("GetByEmail", zap.String("email", email))
	var user domain.User
	err := r.db.WithContext(ctx).
		Where("email = ? AND deleted_at IS NULL", email).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *PostgresRepository) List(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	r.log.Debug("List", zap.Int("limit", limit), zap.Int("offset", offset))

	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var result []*domain.User
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

func (r *PostgresRepository) Create(ctx context.Context, user *domain.User) error {
	r.log.Debug("Create", zap.String("email", user.Email))
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return err
	}
	return nil
}

func (r *PostgresRepository) Update(ctx context.Context, user *domain.User) (bool, error) {
	r.log.Debug("Update", zap.Stringer("user_id", user.ID))
	updates := map[string]interface{}{
		"email":         user.Email,
		"password":      user.Password,
		"is_active":     user.IsActive,
		"token_version": user.TokenVersion,
	}

	tx := r.db.WithContext(ctx).
		Model(&domain.User{}).
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
		Model(&domain.User{}).
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

func (r *PostgresRepository) GetRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	r.log.Debug("GetRoles", zap.Stringer("user_id", userID))

	type roleRow struct {
		Name string
	}

	var rows []roleRow
	err := r.db.WithContext(ctx).
		Table("user_roles").
		Select("roles.name").
		Joins("JOIN roles ON roles.id = user_roles.role_id").
		Where("user_roles.user_id = ?", userID).
		Order("roles.name").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	out := make([]string, 0, len(rows))
	for _, row := range rows {
		if trimmed := strings.TrimSpace(row.Name); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out, nil
}

func (r *PostgresRepository) GetPermissions(ctx context.Context, userID uuid.UUID) ([]string, error) {
	r.log.Debug("GetPermissions", zap.Stringer("user_id", userID))

	type permRow struct {
		Name string
	}

	var rows []permRow
	err := r.db.WithContext(ctx).
		Table("permissions").
		Distinct("permissions.name").
		Select("permissions.name").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN user_roles ON user_roles.role_id = role_permissions.role_id").
		Where("user_roles.user_id = ?", userID).
		Order("permissions.name").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	out := make([]string, 0, len(rows))
	for _, row := range rows {
		if trimmed := strings.TrimSpace(row.Name); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out, nil
}

func (r *PostgresRepository) IncrementTokenVersion(ctx context.Context, id uuid.UUID) (int64, error) {
	r.log.Debug("IncrementTokenVersion", zap.Stringer("user_id", id))

	res := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("id = ? AND deleted_at IS NULL", id).
		UpdateColumn("token_version", gorm.Expr("token_version + ?", 1))
	if res.Error != nil {
		return 0, res.Error
	}
	if res.RowsAffected == 0 {
		return 0, gorm.ErrRecordNotFound
	}

	var user domain.User
	if err := r.db.WithContext(ctx).
		Select("token_version").
		Where("id = ?", id).
		First(&user).Error; err != nil {
		return 0, err
	}
	return user.TokenVersion, nil
}

var _ users.Repository = (*PostgresRepository)(nil)
