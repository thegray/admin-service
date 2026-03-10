package threats

import (
	"context"

	domain "admin-service/internal/domain/model"

	"github.com/google/uuid"
)

type Repository interface {
	List(ctx context.Context, limit, offset int) ([]*domain.Threat, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Threat, error)
	Create(ctx context.Context, threat *domain.Threat) error
	Update(ctx context.Context, threat *domain.Threat) (bool, error)
	Delete(ctx context.Context, id uuid.UUID) (bool, error)
}
