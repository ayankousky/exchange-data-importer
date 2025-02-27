package mongo

import (
	"context"
	"fmt"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"go.mongodb.org/mongo-driver/mongo"
)

// Factory is a factory for creating mongo repositories
type Factory struct {
	client *mongo.Client
}

// NewMongoRepoFactory creates a new Factory
func NewMongoRepoFactory(client *mongo.Client) (*Factory, error) {
	return &Factory{client: client}, nil
}

// GetTickRepository returns a new TickRepository
func (f *Factory) GetTickRepository(name string) (domain.TickRepository, error) {
	db := f.client.Database("exchange").Collection(name + "_tick")

	// create required indexes
	_, err := db.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys: map[string]any{"created_at": 1},
	})
	if err != nil {
		return nil, fmt.Errorf("error creating index for tick repository: %w", err)
	}

	return &Tick{db: db}, nil
}

// GetLiquidationRepository returns a new LiquidationRepository
func (f *Factory) GetLiquidationRepository(name string) (domain.LiquidationRepository, error) {
	repo, err := NewLiquidationRepository(f.client.Database("exchange").Collection(name + "_liquidation"))
	if err != nil {
		return nil, fmt.Errorf("error creating liquidation repository: %w", err)
	}
	return repo, nil
}
