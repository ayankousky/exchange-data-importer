package infrastructure

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient creates a new Redis client to inject into the other services
func NewRedisClient(ctx context.Context, url string, maxConns int) (*redis.Client, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	opt.PoolSize = maxConns

	client := redis.NewClient(opt)
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return client, nil
}
