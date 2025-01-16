package mongo

import (
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
	return &Tick{db: f.client.Database("exchange").Collection(name + "_tick")}
}

// GetLiquidationRepository returns a new LiquidationRepository
func (f *Factory) GetLiquidationRepository(name string) domain.LiquidationRepository {
	return &Liquidation{db: f.client.Database("exchange").Collection(name + "_liquidation")}
}
