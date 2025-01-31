package mock

import (
	"context"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
)

// FactoryMock is a mock factory for creating repositories
type FactoryMock struct {
}

// NewFactoryMock creates a new FactoryMock
func NewFactoryMock() *FactoryMock {
	return &FactoryMock{}
}

// GetTickRepository mock method
func (f *FactoryMock) GetTickRepository(_ string) domain.TickRepository {
	return &TickRepoMock{}
}

// GetLiquidationRepository mock method
func (f *FactoryMock) GetLiquidationRepository(_ string) domain.LiquidationRepository {
	return &LiquidationRepoMock{}
}

// TickRepoMock is a mock repository for storing tick snapshots
type TickRepoMock struct {
	ticks []domain.Tick
}

// Create mock method
func (r *TickRepoMock) Create(_ context.Context, tick domain.Tick) error {
	r.ticks = append(r.ticks, tick)
	return nil
}

// GetHistorySince mock method
func (r *TickRepoMock) GetHistorySince(_ context.Context, _ time.Time) ([]domain.Tick, error) {
	return r.ticks, nil
}

// LiquidationRepoMock is a mock repository for storing liquidation snapshots
type LiquidationRepoMock struct {
	liquidations []domain.Liquidation
}

// Create mock method
func (r *LiquidationRepoMock) Create(_ context.Context, liquidation domain.Liquidation) error {
	r.liquidations = append(r.liquidations, liquidation)
	return nil
}

// GetLiquidationsHistory mock method
func (r *LiquidationRepoMock) GetLiquidationsHistory(_ context.Context, _ time.Time) (domain.LiquidationsHistory, error) {
	return domain.LiquidationsHistory{
		LongLiquidations1s:   10,
		LongLiquidations2s:   20,
		LongLiquidations5s:   50,
		LongLiquidations60s:  600,
		ShortLiquidations1s:  1,
		ShortLiquidations2s:  2,
		ShortLiquidations10s: 10,
	}, nil
}
