package mongo

import (
	"context"
	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
)

// Tick is a repository for storing tick snapshots
type Tick struct {
	db *mongo.Collection
}

var _ domain.TickSnapshotRepository = &Tick{}

// Create method stores a tick snapshot in the database
func (r *Tick) Create(ctx context.Context, tick *domain.TickSnapshot) error {
	_, err := r.db.InsertOne(ctx, tick)
	if err != nil {
		log.Default().Printf("Error inserting tick snapshot: %v", err)

	}

	return nil
}
