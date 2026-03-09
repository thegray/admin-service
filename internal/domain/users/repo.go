package users

import (
	"admin-service/internal/domain"
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	List(ctx context.Context, limit, offset int) ([]*domain.User, error)
	Create(ctx context.Context, user *domain.User) error
	Update(ctx context.Context, user *domain.User) (bool, error)
	SoftDelete(ctx context.Context, id uuid.UUID) (bool, error)
	GetRoles(ctx context.Context, userID uuid.UUID) ([]string, error)
	GetPermissions(ctx context.Context, userID uuid.UUID) ([]string, error)
	IncrementTokenVersion(ctx context.Context, id uuid.UUID) (int64, error)
}
