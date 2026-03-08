package example

import (
	"context"

	"admin-service/internal/domain"
)

// interface definition for the repository
type Repository interface {
	GetByID(ctx context.Context, id int64) (*domain.Example, error)
	Insert(ctx context.Context, ex *domain.Example) error
}
