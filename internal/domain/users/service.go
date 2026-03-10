package users

import (
	"context"
	"strings"

	domain "admin-service/internal/domain/model"
	svcerrors "admin-service/pkg/errors"

	audit "admin-service/internal/domain/audit"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo Repository
	log      *zap.Logger
	auditSvc *audit.Service
}

type CreateUserInput struct {
	Email    string
	Password string
	IsActive bool
	RoleID   *uuid.UUID
}

type UpdateUserInput struct {
	Email    *string
	Password *string
	IsActive *bool
}

func NewService(repo Repository, auditSvc *audit.Service, log *zap.Logger) *Service {
	if log == nil {
		log = zap.NewNop()
	}
	return &Service{
		repo:     repo,
		log:      log.Named("users-service"),
		auditSvc: auditSvc,
	}
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
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

func (s *Service) List(ctx context.Context, limit, offset int) ([]*domain.User, error) {
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

func (s *Service) Create(ctx context.Context, actorID *uuid.UUID, in CreateUserInput) (*domain.User, error) {
	if strings.TrimSpace(in.Email) == "" || strings.TrimSpace(in.Password) == "" {
		return nil, svcerrors.ErrInvalidPayload
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		s.log.Error("hashing password failed", zap.Error(err))
		return nil, svcerrors.ErrInternal
	}

	user := &domain.User{
		Email:    in.Email,
		Password: string(hashed),
		IsActive: in.IsActive,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		s.log.Error("repository Create failed", zap.Error(err))
		if isEmailConflict(err) {
			return nil, svcerrors.ErrEmailExists
		}
		return nil, svcerrors.ErrInternal
	}

	if in.RoleID != nil {
		role, err := s.repo.GetRoleByID(ctx, *in.RoleID)
		if err != nil {
			s.log.Error("failed to fetch role", zap.Error(err))
			return nil, svcerrors.ErrInternal
		}
		if role == nil {
			return nil, svcerrors.ErrRoleNotFound
		}
		if err := s.repo.AssignRole(ctx, user.ID, role.ID); err != nil {
			s.log.Error("failed to assign role", zap.Error(err))
			return nil, svcerrors.ErrInternal
		}
	}

	meta := map[string]any{
		"email":     user.Email,
		"is_active": user.IsActive,
	}
	if in.RoleID != nil {
		meta["role_id"] = in.RoleID.String()
	}
	s.recordUserAction(ctx, actorID, audit.ActionCreateUser, &user.ID, meta)

	return user, nil
}

func (s *Service) Update(ctx context.Context, actorID *uuid.UUID, id uuid.UUID, in UpdateUserInput) (*domain.User, error) {
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
	fields := make([]string, 0, 3)
	if in.Email != nil {
		fields = append(fields, "email")
	}
	if in.Password != nil {
		fields = append(fields, "password")
	}
	if in.IsActive != nil {
		fields = append(fields, "is_active")
	}
	s.recordUserAction(ctx, actorID, audit.ActionUpdateUser, &user.ID, map[string]any{
		"fields": fields,
	})

	return user, nil
}

func (s *Service) Delete(ctx context.Context, actorID *uuid.UUID, id uuid.UUID) error {
	changed, err := s.repo.SoftDelete(ctx, id)
	if err != nil {
		s.log.Error("repository SoftDelete failed", zap.Error(err))
		return err
	}
	if !changed {
		return svcerrors.ErrNotFound
	}
	s.recordUserAction(ctx, actorID, audit.ActionDeleteUser, &id, nil)
	return nil
}

func (s *Service) recordUserAction(ctx context.Context, actorID *uuid.UUID, action string, resourceID *uuid.UUID, metadata map[string]any) {
	if s.auditSvc == nil {
		return
	}
	s.auditSvc.Record(ctx, audit.RecordInput{
		ActorID:      actorID,
		Action:       action,
		ResourceType: audit.ResourceTypeUser,
		ResourceID:   resourceID,
		Status:       audit.StatusSuccess,
		Metadata:     metadata,
	})
}

func isEmailConflict(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "duplicate key value") &&
		!strings.Contains(msg, "unique constraint") {
		return false
	}
	return strings.Contains(msg, "users_email_key") || strings.Contains(msg, "users_email_unique")
}
