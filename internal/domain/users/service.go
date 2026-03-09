package users

import (
	"context"
	"strings"

	svcerrors "admin-service/pkg/errors"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo Repository
	log  *zap.Logger
}

type CreateUserInput struct {
	Email    string
	Password string
	IsActive bool
}

type UpdateUserInput struct {
	Email    *string
	Password *string
	IsActive *bool
}

func NewService(repo Repository, log *zap.Logger) *Service {
	if log == nil {
		log = zap.NewNop()
	}
	return &Service{
		repo: repo,
		log:  log.Named("users-service"),
	}
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	s.log.Debug("get user", zap.Stringer("user_id", id))

	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.log.Error("repository GetByID failed", zap.Error(err))
		return nil, err
	}
	if user == nil {
		s.log.Info("user not found", zap.Stringer("user_id", id))
		return nil, svcerrors.ErrNotFound
	}

	s.log.Debug("user retrieved", zap.Stringer("user_id", id))
	return user, nil
}

func (s *Service) List(ctx context.Context, limit, offset int) ([]*User, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	s.log.Debug("listing users", zap.Int("limit", limit), zap.Int("offset", offset))
	users, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		s.log.Error("repository List failed", zap.Error(err))
		return nil, err
	}
	return users, nil
}

func (s *Service) Create(ctx context.Context, in CreateUserInput) (*User, error) {
	if strings.TrimSpace(in.Email) == "" || strings.TrimSpace(in.Password) == "" {
		return nil, svcerrors.ErrInvalidPayload
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		s.log.Error("hashing password failed", zap.Error(err))
		return nil, svcerrors.ErrInternal
	}

	user := &User{
		Email:    in.Email,
		Password: string(hashed),
		IsActive: in.IsActive,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		s.log.Error("repository Create failed", zap.Error(err))
		return nil, svcerrors.ErrInternal
	}

	return user, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, in UpdateUserInput) (*User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.log.Error("repository GetByID failed", zap.Error(err))
		return nil, err
	}
	if user == nil {
		return nil, svcerrors.ErrNotFound
	}

	if in.Email != nil {
		user.Email = *in.Email
	}

	if in.Password != nil {
		if strings.TrimSpace(*in.Password) == "" {
			return nil, svcerrors.ErrInvalidPayload
		}
		hashed, err := bcrypt.GenerateFromPassword([]byte(*in.Password), bcrypt.DefaultCost)
		if err != nil {
			s.log.Error("hashing password failed", zap.Error(err))
			return nil, svcerrors.ErrInternal
		}
		user.Password = string(hashed)
	}

	if in.IsActive != nil {
		user.IsActive = *in.IsActive
	}

	updated, err := s.repo.Update(ctx, user)
	if err != nil {
		s.log.Error("repository Update failed", zap.Error(err))
		return nil, svcerrors.ErrInternal
	}
	if !updated {
		return nil, svcerrors.ErrNotFound
	}

	return user, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	changed, err := s.repo.SoftDelete(ctx, id)
	if err != nil {
		s.log.Error("repository SoftDelete failed", zap.Error(err))
		return err
	}
	if !changed {
		return svcerrors.ErrNotFound
	}
	return nil
}
