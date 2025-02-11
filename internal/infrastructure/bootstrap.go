package infrastructure

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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
func NewMongoClient(uri string) (*mongo.Client, error) {
	clientOptions := options.Client().ApplyURI(uri)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

	log.Println("Connected to MongoDB")
	return client, nil
}
