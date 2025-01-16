package mongo

import (
	"context"
	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
)

// Liquidation is a repository for storing liquidation snapshots
type Liquidation struct {
	db *mongo.Collection
}

// Create method stores a liquidation in the database
func (r *Liquidation) Create(ctx context.Context, liquidation *domain.Liquidation) error {
	_, err := r.db.InsertOne(ctx, liquidation)
	if err != nil {
		log.Default().Printf("Error inserting tick snapshot: %v", err)

	}

	return nil
}
