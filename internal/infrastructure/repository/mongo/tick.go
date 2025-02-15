package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Tick is a repository for storing tick snapshots
type Tick struct {
	db *mongo.Collection
}

// Create method stores a tick snapshot in the database
func (r *Tick) Create(ctx context.Context, tick domain.Tick) error {
	_, err := r.db.InsertOne(ctx, tick)
	if err != nil {
		return fmt.Errorf("error inserting tick snapshot: %w", err)
	}

	return nil
}

// GetHistorySince method returns a list of tick snapshots since the specified time
func (r *Tick) GetHistorySince(ctx context.Context, since time.Time) ([]domain.Tick, error) {
	filter := map[string]interface{}{
		"created_at": map[string]interface{}{
			"$gte": since,
		},
	}
	findOptions := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})

	cursor, err := r.db.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("error finding tick snapshots: %w", err)
	}
	defer cursor.Close(ctx)

	var history []domain.Tick
	for cursor.Next(ctx) {
		var tick domain.Tick
		if err := cursor.Decode(&tick); err != nil {
			return nil, fmt.Errorf("error decoding tick snapshot: %w", err)
		}
		history = append(history, tick)
	}

	return history, nil

}
