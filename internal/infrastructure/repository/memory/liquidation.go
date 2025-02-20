package memory

import (
	"context"
	"sync"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
)

// InMemoryLiquidationRepository stores liquidations in memory
type InMemoryLiquidationRepository struct {
	liquidations []domain.Liquidation
	mu           sync.RWMutex
}

// Create a new InMemoryLiquidationRepository
func (r *InMemoryLiquidationRepository) Create(_ context.Context, l domain.Liquidation) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Store the liquidation
	r.liquidations = append(r.liquidations, l)
	return nil
}

// GetLiquidationsHistory returns liquidations history for the given time
func (r *InMemoryLiquidationRepository) GetLiquidationsHistory(_ context.Context, timeAt time.Time) (domain.LiquidationsHistory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	history := domain.LiquidationsHistory{}

	oneSecondAgo := timeAt.Add(-1 * time.Second)
	twoSecondsAgo := timeAt.Add(-2 * time.Second)
	fiveSecondsAgo := timeAt.Add(-5 * time.Second)
	tenSecondsAgo := timeAt.Add(-10 * time.Second)
	sixtySecondsAgo := timeAt.Add(-60 * time.Second)

	for _, l := range r.liquidations {
		if l.EventAt.Before(sixtySecondsAgo) {
			continue
		}

		if l.Order.Side == domain.OrderSideSell {
			if l.EventAt.After(oneSecondAgo) {
				history.LongLiquidations1s++
			}
			if l.EventAt.After(twoSecondsAgo) {
				history.LongLiquidations2s++
			}
			if l.EventAt.After(fiveSecondsAgo) {
				history.LongLiquidations5s++
			}
			if l.EventAt.After(sixtySecondsAgo) {
				history.LongLiquidations60s++
			}
		} else {
			if l.EventAt.After(oneSecondAgo) {
				history.ShortLiquidations1s++
			}
			if l.EventAt.After(twoSecondsAgo) {
				history.ShortLiquidations2s++
			}
			if l.EventAt.After(tenSecondsAgo) {
				history.ShortLiquidations10s++
			}
		}
	}

	// Clean up old liquidations (older than 60 seconds)
	r.cleanup(sixtySecondsAgo)

	return history, nil
}

// cleanup removes liquidations older than the given time
func (r *InMemoryLiquidationRepository) cleanup(before time.Time) {
	newLiquidations := make([]domain.Liquidation, 0)
	for _, l := range r.liquidations {
		if l.EventAt.After(before) {
			newLiquidations = append(newLiquidations, l)
		}
	}
	r.liquidations = newLiquidations
}
