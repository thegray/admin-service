package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client redis.Cmdable
}

func NewRedisStore(client redis.Cmdable) *RedisStore {
	return &RedisStore{client: client}
}

func (r *RedisStore) Save(ctx context.Context, key string, payload []byte, ttl time.Duration) error {
	return r.client.Set(ctx, key, payload, ttl).Err()
}

func (r *RedisStore) Load(ctx context.Context, key string) ([]byte, error) {
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	return data, nil
}

func (r *RedisStore) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}
