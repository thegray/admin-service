package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	model "admin-service/internal/domain/model"
	"admin-service/pkg/auth"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

func TestServiceLoginSuccess(t *testing.T) {
	ctx := context.Background()
	user := newTestUser()
	hashed, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	require.NoError(t, err)
	user.Password = string(hashed)

	repo := &mockUsersRepo{
		getByEmailFn: func(ctx context.Context, email string) (*model.User, error) {
			return user, nil
		},
		getRolesFn: func(ctx context.Context, id uuid.UUID) ([]string, error) {
			return []string{"admin"}, nil
		},
		getPermissionsFn: func(ctx context.Context, id uuid.UUID) ([]string, error) {
			return []string{"users:read"}, nil
		},
	}

	store := newSpyStore()
	cache := NewSessionCache(store, time.Minute)
	tokenManager, err := auth.NewTokenManager("secret", time.Minute)
	require.NoError(t, err)
	service, err := NewService(repo, tokenManager, cache, store, time.Hour, nil, zap.NewNop())
	require.NoError(t, err)

	tokens, actorID, err := service.Login(ctx, user.Email, "secret")
	require.NoError(t, err)
	require.NotNil(t, actorID)
	require.Equal(t, user.ID, *actorID)
	require.NotEmpty(t, tokens.AccessToken)
	require.NotEmpty(t, tokens.RefreshToken)

	sessionData, err := store.Load(ctx, sessionKey(user.ID.String()))
	require.NoError(t, err)
	require.NotNil(t, sessionData)

	found := false
	for _, key := range store.savedKeys {
		if keyHasPrefix(key, refreshPrefix) {
			found = true
			break
		}
	}
	require.True(t, found, "refresh token was saved")
}

func TestServiceRefreshRotatesToken(t *testing.T) {
	ctx := context.Background()
	user := newTestUser()
	repo := &mockUsersRepo{}
	store := newSpyStore()
	cache := NewSessionCache(store, time.Minute)
	tokenManager, _ := auth.NewTokenManager("secret", time.Minute)
	service, _ := NewService(repo, tokenManager, cache, store, time.Hour, nil, zap.NewNop())

	state := newSessionState(user, []string{"admin"}, []string{"users:read"})
	require.NoError(t, cache.Save(ctx, state))

	oldRefresh := "old-refresh-token"
	record := refreshRecord{
		UserID:       user.ID.String(),
		TokenVersion: 0,
		ExpiresAt:    time.Now().Add(time.Hour),
	}
	data, err := json.Marshal(record)
	require.NoError(t, err)
	oldHash := hashToken(oldRefresh)
	require.NoError(t, store.Save(ctx, refreshKey(oldHash), data, time.Hour))

	tokens, actorID, err := service.Refresh(ctx, oldRefresh)
	require.NoError(t, err)
	require.NotNil(t, actorID)
	require.Equal(t, user.ID, *actorID)
	require.NotEmpty(t, tokens.AccessToken)
	require.NotEmpty(t, tokens.RefreshToken)

	require.Contains(t, store.deletedKeys, refreshKey(oldHash))

	newHash := hashToken(tokens.RefreshToken)
	require.Contains(t, store.savedKeys, refreshKey(newHash))
}

func TestServiceLogoutClearsSession(t *testing.T) {
	ctx := context.Background()
	user := newTestUser()
	called := false
	repo := &mockUsersRepo{
		incrementTokenVersionFn: func(ctx context.Context, id uuid.UUID) (int64, error) {
			called = true
			return 1, nil
		},
	}
	store := newSpyStore()
	cache := NewSessionCache(store, time.Minute)
	tokenManager, _ := auth.NewTokenManager("secret", time.Minute)
	service, _ := NewService(repo, tokenManager, cache, store, time.Hour, nil, zap.NewNop())

	state := newSessionState(user, []string{"admin"}, []string{"users:read"})
	require.NoError(t, cache.Save(ctx, state))

	refresh := "logout-token"
	hash := hashToken(refresh)
	record := refreshRecord{UserID: user.ID.String(), TokenVersion: 0, ExpiresAt: time.Now().Add(time.Hour)}
	bytes, err := json.Marshal(record)
	require.NoError(t, err)
	require.NoError(t, store.Save(ctx, refreshKey(hash), bytes, time.Hour))

	userID, err := service.Logout(ctx, refresh)
	require.NoError(t, err)
	require.NotNil(t, userID)
	require.Equal(t, user.ID, *userID)
	require.True(t, called, "token version was incremented")
	require.Contains(t, store.deletedKeys, refreshKey(hash))
	require.Contains(t, store.deletedKeys, sessionKey(user.ID.String()))
}

type spyStore struct {
	data        map[string][]byte
	savedKeys   []string
	deletedKeys []string
}

func newSpyStore() *spyStore {
	return &spyStore{
		data: make(map[string][]byte),
	}
}

func (s *spyStore) Save(ctx context.Context, key string, payload []byte, ttl time.Duration) error {
	s.data[key] = append([]byte(nil), payload...)
	s.savedKeys = append(s.savedKeys, key)
	return nil
}

func (s *spyStore) Load(ctx context.Context, key string) ([]byte, error) {
	data, ok := s.data[key]
	if !ok {
		return nil, nil
	}
	cpy := append([]byte(nil), data...)
	return cpy, nil
}

func (s *spyStore) Delete(ctx context.Context, key string) error {
	delete(s.data, key)
	s.deletedKeys = append(s.deletedKeys, key)
	return nil
}

type mockUsersRepo struct {
	getByEmailFn            func(ctx context.Context, email string) (*model.User, error)
	getRolesFn              func(ctx context.Context, id uuid.UUID) ([]string, error)
	getPermissionsFn        func(ctx context.Context, id uuid.UUID) ([]string, error)
	incrementTokenVersionFn func(ctx context.Context, id uuid.UUID) (int64, error)
}

func (m *mockUsersRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return nil, errors.New("not implemented")
}

func (m *mockUsersRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return nil, nil
}

func (m *mockUsersRepo) List(ctx context.Context, limit, offset int) ([]*model.User, error) {
	return nil, errors.New("not implemented")
}

func (m *mockUsersRepo) Create(ctx context.Context, user *model.User) error {
	return errors.New("not implemented")
}

func (m *mockUsersRepo) Update(ctx context.Context, user *model.User) (bool, error) {
	return false, errors.New("not implemented")
}

func (m *mockUsersRepo) SoftDelete(ctx context.Context, id uuid.UUID) (bool, error) {
	return false, errors.New("not implemented")
}

func (m *mockUsersRepo) GetRoles(ctx context.Context, id uuid.UUID) ([]string, error) {
	if m.getRolesFn != nil {
		return m.getRolesFn(ctx, id)
	}
	return nil, nil
}

func (m *mockUsersRepo) GetPermissions(ctx context.Context, id uuid.UUID) ([]string, error) {
	if m.getPermissionsFn != nil {
		return m.getPermissionsFn(ctx, id)
	}
	return nil, nil
}

func (m *mockUsersRepo) IncrementTokenVersion(ctx context.Context, id uuid.UUID) (int64, error) {
	if m.incrementTokenVersionFn != nil {
		return m.incrementTokenVersionFn(ctx, id)
	}
	return 0, errors.New("not implemented")
}

func (m *mockUsersRepo) GetRoleByID(ctx context.Context, id uuid.UUID) (*model.Role, error) {
	return nil, errors.New("not implemented")
}

func (m *mockUsersRepo) AssignRole(ctx context.Context, userID, roleID uuid.UUID) error {
	return errors.New("not implemented")
}

func newTestUser() *model.User {
	return &model.User{
		ID:        uuid.Must(uuid.NewRandom()),
		Email:     "test@example.com",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func keyHasPrefix(key, prefix string) bool {
	return strings.HasPrefix(key, prefix)
}

func sessionKey(id string) string {
	return "user:" + id
}

const refreshPrefix = "refresh:"

func refreshKey(hash string) string {
	return refreshPrefix + hash
}

type refreshRecord struct {
	UserID       string    `json:"user_id"`
	TokenVersion int64     `json:"token_version"`
	ExpiresAt    time.Time `json:"expires_at"`
}
