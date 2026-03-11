package ratelimit

import (
	"context"

	domain "admin-service/internal/domain/model"
)

type Repository interface {
	List(ctx context.Context) ([]*domain.RateLimitPolicy, error)
	Upsert(ctx context.Context, policy *domain.RateLimitPolicy) error
}
