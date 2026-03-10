package auth

import (
	"context"

	domain "admin-service/internal/domain/model"
	"admin-service/internal/domain/users"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func LoadUserSession(ctx context.Context, repo users.Repository, cache *SessionCache,
	logger *zap.Logger, userID uuid.UUID) (*SessionState, error) {
	if cache != nil {
		state, err := cache.Load(ctx, userID)
		if err != nil && logger != nil {
			logger.Warn("failed to read session cache", zap.Error(err))
		}
		if state != nil {
			return state, nil
		}
	}

	user, err := repo.GetByID(ctx, userID)
	if err != nil {
		if logger != nil {
			logger.Warn("failed to fetch user for session", zap.Error(err))
		}
		return nil, err
	}
	if user == nil {
		return nil, nil
	}

	return BuildAndCacheSession(ctx, repo, cache, logger, user)
}

func BuildAndCacheSession(ctx context.Context, repo users.Repository, cache *SessionCache,
	logger *zap.Logger, user *domain.User) (*SessionState, error) {
	roles, err := repo.GetRoles(ctx, user.ID)
	if err != nil {
		if logger != nil {
			logger.Warn("failed to load roles for user session", zap.Error(err))
		}
		return nil, err
	}

	perms, err := repo.GetPermissions(ctx, user.ID)
	if err != nil {
		if logger != nil {
			logger.Warn("failed to load permissions for user session", zap.Error(err))
		}
		return nil, err
	}

	state := newSessionState(user, roles, perms)
	if cache != nil {
		if err := cache.Save(ctx, state); err != nil && logger != nil {
			logger.Warn("failed to cache session state", zap.Error(err))
		}
	}
	return state, nil
}
