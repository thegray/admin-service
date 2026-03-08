package repository

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"admin-service/internal/domain"
	"admin-service/internal/domain/example"

	"go.uber.org/zap"
)

type InMemoryRepository struct {
	log            *zap.Logger
	exampleStorage sync.Map // type map[int64]*domain.Example
}

var idIncrement atomic.Int64

func NewInMemoryRepository(log *zap.Logger) *InMemoryRepository {
	if log == nil {
		log = zap.NewNop()
	}

	repo := &InMemoryRepository{
		log: log.Named("example-repo"),
	}

	now := time.Now().UTC()
	idIncrement.Store(2)

	repo.exampleStorage.Store(int64(1), &domain.Example{
		ID: 1, Message: "Example 1", CreatedAt: now.Add(-72 * time.Hour),
	})
	repo.exampleStorage.Store(int64(2), &domain.Example{
		ID: 2, Message: "Example 2", CreatedAt: now.Add(-48 * time.Hour),
	})

	return repo
}

func (r *InMemoryRepository) GetByID(ctx context.Context, id int64) (*domain.Example, error) {
	r.log.Debug("repository GetByID", zap.Int64("id", id))
	if val, ok := r.exampleStorage.Load(id); ok {
		r.log.Debug("example loaded from cache", zap.Int64("id", id))
		return val.(*domain.Example), nil
	}
	r.log.Info("example cache miss", zap.Int64("id", id))
	return nil, nil
}

func (r *InMemoryRepository) Insert(ctx context.Context, ex *domain.Example) error {
	r.log.Debug("inserting new example", zap.String("message", ex.Message))

	idIncrement.Add(1)
	ex.ID = idIncrement.Load()
	ex.CreatedAt = time.Now().UTC()
	r.exampleStorage.Store(idIncrement.Load(), ex)

	r.log.Info("example persisted", zap.Int64("id", ex.ID))
	return nil
}

var _ example.Repository = (*InMemoryRepository)(nil)
