package memory

import "github.com/ayankousky/exchange-data-importer/internal/domain"

// InMemoryRepoFactory is a factory for in-memory repositories
type InMemoryRepoFactory struct{}

// NewInMemoryRepoFactory creates a new InMemoryRepoFactory
func NewInMemoryRepoFactory() *InMemoryRepoFactory {
	return &InMemoryRepoFactory{}
}

// GetTickRepository returns a new TickRepository
func (f *InMemoryRepoFactory) GetTickRepository(_ string) (domain.TickRepository, error) {
	return &StatsTickRepository{}, nil
}

// GetLiquidationRepository returns a new LiquidationRepository
func (f *InMemoryRepoFactory) GetLiquidationRepository(_ string) (domain.LiquidationRepository, error) {
	return &InMemoryLiquidationRepository{
		liquidations: make([]domain.Liquidation, 0),
	}, nil
}
