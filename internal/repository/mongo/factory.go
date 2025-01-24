package mongo

import (
	"context"
	"log"

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
func (f *Factory) GetTickRepository(name string) domain.TickRepository {
	db := f.client.Database("exchange").Collection(name + "_tick")

	// create required indexes
	_, err := db.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys: map[string]interface{}{"created_at": 1},
	})
	if err != nil {
		log.Printf("Error creating index: %v", err)
	}

	return &Tick{db: db}
}

// GetLiquidationRepository returns a new LiquidationRepository
func (f *Factory) GetLiquidationRepository(name string) domain.LiquidationRepository {
	return &Liquidation{db: f.client.Database("exchange").Collection(name + "_liquidation")}
}
