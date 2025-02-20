package memory

import (
	"context"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
)

// StatsTickRepository prints tick statistics
type StatsTickRepository struct{}

// Create prints tick statistics
func (r *StatsTickRepository) Create(_ context.Context, _ domain.Tick) error {
	return nil
}

// GetHistorySince returns an empty slice of ticks
func (r *StatsTickRepository) GetHistorySince(_ context.Context, _ time.Time) ([]domain.Tick, error) {
	return []domain.Tick{}, nil
}
