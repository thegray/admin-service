package example

import (
	"context"

	domain "admin-service/internal/domain/model"
)

// interface definition for the repository
type Repository interface {
	GetByID(ctx context.Context, id int64) (*domain.Example, error)
	Insert(ctx context.Context, ex *domain.Example) error
}
