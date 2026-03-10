package auth

import (
	"context"
	"encoding/json"
	"time"

	domain "admin-service/internal/domain/model"

	"github.com/google/uuid"
)

const (
	sessionStatusActive = "active"
	sessionStatusBanned = "banned"
)

type SessionState struct {
	UserID       string   `json:"user_id"`
	TokenVersion int64    `json:"token_version"`
	Status       string   `json:"status"`
	Roles        []string `json:"roles"`
	Permissions  []string `json:"permissions"`
}

func (s *SessionState) IsBanned() bool {
	return s.Status == sessionStatusBanned
}

type SessionCache struct {
	store Store
	ttl   time.Duration
}

func NewSessionCache(store Store, ttl time.Duration) *SessionCache {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &SessionCache{
		store: store,
		ttl:   ttl,
	}
}

func (c *SessionCache) Save(ctx context.Context, state *SessionState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return c.store.Save(ctx, domain.SessionKey(state.UserID), data, c.ttl)
}

func (c *SessionCache) Load(ctx context.Context, userID uuid.UUID) (*SessionState, error) {
	data, err := c.store.Load(ctx, domain.SessionKey(userID.String()))
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}
	var state SessionState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (c *SessionCache) Delete(ctx context.Context, userID uuid.UUID) error {
	return c.store.Delete(ctx, domain.SessionKey(userID.String()))
}

func newSessionState(user *domain.User, roles, permissions []string) *SessionState {
	status := sessionStatusBanned
	if user.IsActive {
		status = sessionStatusActive
	}
	return &SessionState{
		UserID:       user.ID.String(),
		TokenVersion: user.TokenVersion,
		Status:       status,
		Roles:        roles,
		Permissions:  permissions,
	}
}
