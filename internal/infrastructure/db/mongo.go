package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// NewMongoClient creates a new MongoDB client
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
