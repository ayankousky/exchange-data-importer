package importer

import (
	"context"
	"fmt"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
)

// initHistory loads old data from repositories and populates ring buffers
func (i *Importer) initHistory(ctx context.Context) error {
	history, err := i.tickRepository.GetHistorySince(ctx, time.Now().Add(-domain.MaxTickHistory*time.Minute))
	if err != nil {
		return fmt.Errorf("GetHistorySince failed: %w", err)
	}

	for _, tick := range history {
		i.addTickHistory(&tick)
		for _, ticker := range tick.Data {
			i.addTickerHistory(ticker)
		}
	}

	return nil
}

func (i *Importer) addTickHistory(tick *domain.Tick) {
	lastTick, exists := i.tickHistory.Last()
	if exists && lastTick.StartAt.After(tick.StartAt) {
		return
	}

	i.tickHistory.Push(tick)
}

// addTickerHistory updates the ring buffer for a particular ticker - 1 item per 1 minute
func (i *Importer) addTickerHistory(ticker *domain.Ticker) {
	i.tickerHistory.UpdateTicker(ticker)
}

func (i *Importer) getLastTick() (*domain.Tick, error) {
	lastTick, exists := i.tickHistory.Last()
	if !exists {
		return nil, fmt.Errorf("no tick history found")
	}
	return lastTick, nil
}
