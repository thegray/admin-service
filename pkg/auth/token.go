package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	UserID       uuid.UUID `json:"user_id"`
	TokenVersion int64     `json:"token_version"`
	Roles        []string  `json:"roles"`
	Permissions  []string  `json:"permissions"`
	jwt.RegisteredClaims
}

type TokenManager struct {
	secret []byte
	ttl    time.Duration
}

func NewTokenManager(secret string, ttl time.Duration) (*TokenManager, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, fmt.Errorf("token secret cannot be empty")
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("token ttl must be positive")
	}
	return &TokenManager{
		secret: []byte(secret),
		ttl:    ttl,
	}, nil
}

func (m *TokenManager) GenerateAccessToken(ctx context.Context, userID uuid.UUID, tokenVersion int64, roles, permissions []string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:       userID,
		TokenVersion: tokenVersion,
		Roles:        roles,
		Permissions:  permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
			Subject:   userID.String(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *TokenManager) ParseAccessToken(ctx context.Context, tokenStr string) (*Claims, error) {
	if strings.TrimSpace(tokenStr) == "" {
		return nil, fmt.Errorf("token must be provided")
	}
	parser := jwt.NewParser(jwt.WithLeeway(5 * time.Second))
	claims := &Claims{}
	token, err := parser.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
