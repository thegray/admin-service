package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	svcerrors "admin-service/pkg/errors"

	audit "admin-service/internal/domain/audit"
	domain "admin-service/internal/domain/model"
	"admin-service/internal/domain/users"
	"admin-service/pkg/auth"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

const refreshTokenBytes = 64

type Service struct {
	repo         users.Repository
	tokenManager *auth.TokenManager
	sessionCache *SessionCache
	store        Store
	refreshTTL   time.Duration
	log          *zap.Logger
	auditSvc     *audit.Service
}

func NewService(
	repo users.Repository,
	tokenManager *auth.TokenManager,
	sessionCache *SessionCache,
	store Store,
	refreshTTL time.Duration,
	auditSvc *audit.Service,
	logger *zap.Logger,
) (*Service, error) {
	if repo == nil || tokenManager == nil || sessionCache == nil || store == nil {
		return nil, errors.New("auth service dependencies cannot be nil")
	}
	if refreshTTL <= 0 {
		return nil, errors.New("refresh token TTL must be positive")
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Service{
		repo:         repo,
		tokenManager: tokenManager,
		sessionCache: sessionCache,
		store:        store,
		refreshTTL:   refreshTTL,
		log:          logger.Named("auth-service"),
		auditSvc:     auditSvc,
	}, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (*domain.AuthTokens, *uuid.UUID, error) {
	email = strings.TrimSpace(email)
	if email == "" || strings.TrimSpace(password) == "" {
		s.recordAuth(ctx, nil, audit.ActionLogin, audit.StatusFailure, map[string]any{"stage": "validation"})
		return nil, nil, svcerrors.ErrUnauthorized
	}

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		s.log.Error("failed to fetch user by email", zap.Error(err))
		s.recordAuth(ctx, nil, audit.ActionLogin, audit.StatusFailure, map[string]any{"stage": "fetch_user"})
		return nil, nil, err
	}
	if user == nil || !user.IsActive {
		s.recordAuth(ctx, nil, audit.ActionLogin, audit.StatusFailure, map[string]any{"stage": "authorize"})
		return nil, nil, svcerrors.ErrUnauthorized
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		s.recordAuth(ctx, &user.ID, audit.ActionLogin, audit.StatusFailure, map[string]any{"stage": "credentials"})
		return nil, nil, svcerrors.ErrUnauthorized
	}

	session, err := BuildAndCacheSession(ctx, s.repo, s.sessionCache, s.log, user)
	if err != nil {
		s.recordAuth(ctx, &user.ID, audit.ActionLogin, audit.StatusFailure, map[string]any{"stage": "session"})
		return nil, nil, err
	}

	access, err := s.tokenManager.GenerateAccessToken(ctx, user.ID, user.TokenVersion, session.Roles, session.Permissions)
	if err != nil {
		s.log.Error("failed to generate access token", zap.Error(err))
		s.recordAuth(ctx, &user.ID, audit.ActionLogin, audit.StatusFailure, map[string]any{"stage": "access_token"})
		return nil, nil, err
	}

	refresh, err := s.rotateRefreshToken(ctx, user.ID, user.TokenVersion, "")
	if err != nil {
		s.log.Error("failed to persist refresh token", zap.Error(err))
		s.recordAuth(ctx, &user.ID, audit.ActionLogin, audit.StatusFailure, map[string]any{"stage": "refresh_token"})
		return nil, nil, err
	}

	userID := user.ID
	s.recordAuth(ctx, &userID, audit.ActionLogin, audit.StatusSuccess, map[string]any{"email": email})

	return &domain.AuthTokens{
		AccessToken:  access,
		RefreshToken: refresh,
	}, &userID, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*domain.AuthTokens, *uuid.UUID, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		s.recordAuth(ctx, nil, audit.ActionRefreshToken, audit.StatusFailure, map[string]any{"stage": "validation"})
		return nil, nil, svcerrors.ErrUnauthorized
	}

	hash := hashRefreshToken(refreshToken)
	record, err := s.loadRefreshRecord(ctx, hash)
	if err != nil {
		s.log.Error("failed to read refresh token", zap.Error(err))
		s.recordAuth(ctx, nil, audit.ActionRefreshToken, audit.StatusFailure, map[string]any{"stage": "load_token"})
		return nil, nil, err
	}
	if record == nil {
		s.recordAuth(ctx, nil, audit.ActionRefreshToken, audit.StatusFailure, map[string]any{"stage": "token_missing"})
		return nil, nil, svcerrors.ErrUnauthorized
	}

	userID, err := uuid.Parse(record.UserID)
	if err != nil {
		s.log.Error("stored refresh token has invalid user_id", zap.Error(err))
		_ = s.deleteRefreshRecord(ctx, hash)
		s.recordAuth(ctx, nil, audit.ActionRefreshToken, audit.StatusFailure, map[string]any{"stage": "parse_token"})
		return nil, nil, svcerrors.ErrUnauthorized
	}

	session, err := LoadUserSession(ctx, s.repo, s.sessionCache, s.log, userID)
	if err != nil {
		s.recordAuth(ctx, &userID, audit.ActionRefreshToken, audit.StatusFailure, map[string]any{"stage": "load_session"})
		return nil, nil, err
	}
	if session == nil {
		s.recordAuth(ctx, &userID, audit.ActionRefreshToken, audit.StatusFailure, map[string]any{"stage": "session_missing"})
		return nil, nil, svcerrors.ErrUnauthorized
	}

	if session.TokenVersion != record.TokenVersion {
		_ = s.deleteRefreshRecord(ctx, hash)
		s.recordAuth(ctx, &userID, audit.ActionRefreshToken, audit.StatusFailure, map[string]any{"stage": "token_version"})
		return nil, nil, svcerrors.ErrUnauthorized
	}

	if session.Status == sessionStatusBanned {
		_ = s.deleteRefreshRecord(ctx, hash)
		_ = s.sessionCache.Delete(ctx, userID)
		s.recordAuth(ctx, &userID, audit.ActionRefreshToken, audit.StatusFailure, map[string]any{"stage": "session_banned"})
		return nil, nil, svcerrors.ErrUnauthorized
	}

	access, err := s.tokenManager.GenerateAccessToken(ctx, userID, session.TokenVersion, session.Roles, session.Permissions)
	if err != nil {
		s.log.Error("failed to generate access token", zap.Error(err))
		s.recordAuth(ctx, &userID, audit.ActionRefreshToken, audit.StatusFailure, map[string]any{"stage": "access_token"})
		return nil, nil, err
	}

	newRefresh, err := s.rotateRefreshToken(ctx, userID, session.TokenVersion, hash)
	if err != nil {
		s.log.Error("failed to rotate refresh token", zap.Error(err))
		s.recordAuth(ctx, &userID, audit.ActionRefreshToken, audit.StatusFailure, map[string]any{"stage": "refresh_token"})
		return nil, nil, err
	}

	s.recordAuth(ctx, &userID, audit.ActionRefreshToken, audit.StatusSuccess, nil)

	return &domain.AuthTokens{
		AccessToken:  access,
		RefreshToken: newRefresh,
	}, &userID, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) (*uuid.UUID, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil, nil
	}

	hash := hashRefreshToken(refreshToken)
	record, err := s.loadRefreshRecord(ctx, hash)
	if err != nil {
		s.log.Error("failed to load refresh token for logout", zap.Error(err))
		s.recordAuth(ctx, nil, audit.ActionLogout, audit.StatusFailure, map[string]any{"stage": "load_token"})
		return nil, err
	}

	if record == nil {
		s.recordAuth(ctx, nil, audit.ActionLogout, audit.StatusFailure, map[string]any{"stage": "token_missing"})
		return nil, nil
	}

	var actorID *uuid.UUID
	userID, err := uuid.Parse(record.UserID)
	if err != nil {
		s.log.Error("stored refresh token has invalid user_id during logout", zap.Error(err))
		s.recordAuth(ctx, nil, audit.ActionLogout, audit.StatusFailure, map[string]any{"stage": "parse_token"})
		return nil, err
	}

	actorID = &userID
	if _, err := s.repo.IncrementTokenVersion(ctx, userID); err != nil {
		s.log.Warn("failed to increment token version", zap.Error(err))
	}
	if err := s.sessionCache.Delete(ctx, userID); err != nil {
		s.log.Warn("failed to delete cached session during logout", zap.Error(err))
	}

	if err := s.deleteRefreshRecord(ctx, hash); err != nil {
		s.log.Warn("failed to delete refresh token during logout", zap.Error(err))
	}

	s.recordAuth(ctx, actorID, audit.ActionLogout, audit.StatusSuccess, nil)
	return actorID, nil
}

func (s *Service) recordAuth(ctx context.Context, actorID *uuid.UUID, action, status string, metadata map[string]any) {
	if s.auditSvc == nil {
		return
	}
	m := make(map[string]any, len(metadata))
	for k, v := range metadata {
		m[k] = v
	}
	s.auditSvc.Record(ctx, audit.RecordInput{
		ActorID:      actorID,
		Action:       action,
		ResourceType: audit.ResourceTypeAuth,
		Status:       status,
		Metadata:     m,
	})
}

func (s *Service) rotateRefreshToken(ctx context.Context, userID uuid.UUID, tokenVersion int64, previousHash string) (string, error) {
	token, err := newRefreshToken()
	if err != nil {
		return "", err
	}

	newHash := hashRefreshToken(token)
	record := domain.RefreshTokenRecord{
		UserID:       userID.String(),
		TokenVersion: tokenVersion,
		ExpiresAt:    time.Now().Add(s.refreshTTL),
	}

	if err := s.saveRefreshRecord(ctx, newHash, &record); err != nil {
		return "", err
	}

	if previousHash != "" {
		if err := s.deleteRefreshRecord(ctx, previousHash); err != nil {
			s.log.Warn("failed to delete previous refresh token", zap.Error(err))
		}
	}

	return token, nil
}

func (s *Service) saveRefreshRecord(ctx context.Context, key string, record *domain.RefreshTokenRecord) error {
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return s.store.Save(ctx, domain.RefreshKey(key), data, s.refreshTTL)
}

func (s *Service) loadRefreshRecord(ctx context.Context, key string) (*domain.RefreshTokenRecord, error) {
	data, err := s.store.Load(ctx, domain.RefreshKey(key))
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}
	var record domain.RefreshTokenRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func (s *Service) deleteRefreshRecord(ctx context.Context, key string) error {
	return s.store.Delete(ctx, domain.RefreshKey(key))
}

func newRefreshToken() (string, error) {
	buff := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(buff); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buff), nil
}

func hashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
