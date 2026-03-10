package redisclient

import (
	"context"
	"time"

	"admin-service/pkg/config"

	"github.com/redis/go-redis/v9"
)

func New(cfg config.Config) *redis.Client {
	options := &redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	}
	client := redis.NewClient(options)

	// warm up connection once
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = client.Ping(ctx).Err()

	return client
}
