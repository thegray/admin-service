package auth

import (
	"context"
	"time"
)

type Store interface {
	Save(ctx context.Context, key string, payload []byte, ttl time.Duration) error
	Load(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
}
