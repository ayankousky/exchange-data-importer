package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
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

// NewMongoClient creates a new MongoDB client to inject into the other services
func NewMongoClient(ctx context.Context, uri string) (*mongo.Client, error) {
	clientOptions := options.Client().ApplyURI(uri)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("mongo.Connect failed: %w", err)
	}

	// Verify the connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("mongo.Ping failed: %w", err)
	}

	return client, nil
}

// NewLogger creates a new logger to inject into the other services
func NewLogger(env, service string) (*zap.Logger, error) {
	logger, _ := zap.NewProduction(zap.Fields(
		zap.String("env", env),
		zap.String("service", service),
	))

	if env == "" || env == "development" {
		logger, _ = zap.NewDevelopment()
	}

	return logger, nil
}
