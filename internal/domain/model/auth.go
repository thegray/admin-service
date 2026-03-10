package domain

import "time"

type AuthTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type RefreshTokenRecord struct {
	UserID       string    `json:"user_id"`
	TokenVersion int64     `json:"token_version"`
	ExpiresAt    time.Time `json:"expires_at"`
}

func SessionKey(userID string) string {
	return "user:" + userID
}

func RefreshKey(hash string) string {
	return "refresh:" + hash
}
