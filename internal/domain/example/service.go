package example

import (
	domain "admin-service/internal/domain/model"
	svcerrors "admin-service/pkg/errors"
	"context"

	"go.uber.org/zap"
)

type Service struct {
	repo Repository
	log  *zap.Logger
}

func NewService(repo Repository, log *zap.Logger) *Service {
	if log == nil {
		log = zap.NewNop()
	}
	return &Service{repo: repo, log: log.Named("example-service")}
}

func (s *Service) GetByID(ctx context.Context, id int64) (*domain.Example, error) {
	s.log.Debug("getting example", zap.Int64("id", id))
	example, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.log.Error("repository GetByID failed", zap.Int64("id", id), zap.Error(err))
		return nil, err
	}
	if example == nil {
		s.log.Info("example not found", zap.Int64("id", id))
		return nil, svcerrors.ErrNotFound
	}
	s.log.Debug("example retrieved", zap.Int64("id", id))
	return example, nil
}

func (s *Service) Create(ctx context.Context, message string) (*domain.Example, error) {
	s.log.Debug("creating example", zap.String("message", message))
	ex := &domain.Example{Message: message}
	err := s.repo.Insert(ctx, ex)
	if err != nil {
		s.log.Error("repository Insert failed", zap.Error(err))
		return nil, svcerrors.ErrInternal
	}

	s.log.Info("example created", zap.Int64("id", ex.ID))
	return ex, nil
}
