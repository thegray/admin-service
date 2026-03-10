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
}

func NewService(
	repo users.Repository,
	tokenManager *auth.TokenManager,
	sessionCache *SessionCache,
	store Store,
	refreshTTL time.Duration,
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
	}, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (*domain.AuthTokens, error) {
	email = strings.TrimSpace(email)
	if email == "" || strings.TrimSpace(password) == "" {
		return nil, svcerrors.ErrUnauthorized
	}

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		s.log.Error("failed to fetch user by email", zap.Error(err))
		return nil, err
	}
	if user == nil || !user.IsActive {
		return nil, svcerrors.ErrUnauthorized
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, svcerrors.ErrUnauthorized
	}

	session, err := BuildAndCacheSession(ctx, s.repo, s.sessionCache, s.log, user)
	if err != nil {
		return nil, err
	}

	access, err := s.tokenManager.GenerateAccessToken(ctx, user.ID, user.TokenVersion, session.Roles, session.Permissions)
	if err != nil {
		s.log.Error("failed to generate access token", zap.Error(err))
		return nil, err
	}

	refresh, err := s.rotateRefreshToken(ctx, user.ID, user.TokenVersion, "")
	if err != nil {
		s.log.Error("failed to persist refresh token", zap.Error(err))
		return nil, err
	}

	return &domain.AuthTokens{
		AccessToken:  access,
		RefreshToken: refresh,
	}, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*domain.AuthTokens, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil, svcerrors.ErrUnauthorized
	}

	hash := hashRefreshToken(refreshToken)
	record, err := s.loadRefreshRecord(ctx, hash)
	if err != nil {
		s.log.Error("failed to read refresh token", zap.Error(err))
		return nil, err
	}
	if record == nil {
		return nil, svcerrors.ErrUnauthorized
	}

	userID, err := uuid.Parse(record.UserID)
	if err != nil {
		s.log.Error("stored refresh token has invalid user_id", zap.Error(err))
		_ = s.deleteRefreshRecord(ctx, hash)
		return nil, svcerrors.ErrUnauthorized
	}

	session, err := LoadUserSession(ctx, s.repo, s.sessionCache, s.log, userID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, svcerrors.ErrUnauthorized
	}

	if session.TokenVersion != record.TokenVersion {
		_ = s.deleteRefreshRecord(ctx, hash)
		return nil, svcerrors.ErrUnauthorized
	}

	if session.Status == sessionStatusBanned {
		_ = s.deleteRefreshRecord(ctx, hash)
		_ = s.sessionCache.Delete(ctx, userID)
		return nil, svcerrors.ErrUnauthorized
	}

	access, err := s.tokenManager.GenerateAccessToken(ctx, userID, session.TokenVersion, session.Roles, session.Permissions)
	if err != nil {
		s.log.Error("failed to generate access token", zap.Error(err))
		return nil, err
	}

	newRefresh, err := s.rotateRefreshToken(ctx, userID, session.TokenVersion, hash)
	if err != nil {
		s.log.Error("failed to rotate refresh token", zap.Error(err))
		return nil, err
	}

	return &domain.AuthTokens{
		AccessToken:  access,
		RefreshToken: newRefresh,
	}, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil
	}

	hash := hashRefreshToken(refreshToken)
	record, err := s.loadRefreshRecord(ctx, hash)
	if err != nil {
		s.log.Error("failed to load refresh token for logout", zap.Error(err))
		return err
	}

	if record != nil {
		userID, err := uuid.Parse(record.UserID)
		if err == nil {
			if _, err := s.repo.IncrementTokenVersion(ctx, userID); err != nil {
				s.log.Warn("failed to increment token version", zap.Error(err))
			}
			if err := s.sessionCache.Delete(ctx, userID); err != nil {
				s.log.Warn("failed to delete cached session during logout", zap.Error(err))
			}
		}
	}

	if err := s.deleteRefreshRecord(ctx, hash); err != nil {
		s.log.Warn("failed to delete refresh token during logout", zap.Error(err))
	}

	return nil
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
