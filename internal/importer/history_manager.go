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
	i.tickHistory.Push(tick)
}

// addTickerHistory updates the ring buffer for a particular ticker - 1 item per 1 minute
func (i *Importer) addTickerHistory(ticker *domain.Ticker) {
	history := i.tickerHistory.Get(ticker.Symbol)

	lastTickerData, err := i.getLastTicker(ticker.Symbol)
	// If there’s no data for this minute, push a new item
	if err != nil || !lastTickerData.CreatedAt.Truncate(time.Minute).Equal(ticker.CreatedAt.Truncate(time.Minute)) {
		ticker.Max = ticker.Ask
		ticker.Min = ticker.Ask
		history.Push(ticker)
		return
	}

	// Update the existing lastTickerData
	if ticker.Ask > lastTickerData.Max {
		lastTickerData.Max = ticker.Ask
	}
	if ticker.Ask < lastTickerData.Min {
		lastTickerData.Min = ticker.Ask
	}
	lastTickerData.Ask = ticker.Ask
	lastTickerData.Bid = ticker.Bid
	lastTickerData.CreatedAt = ticker.CreatedAt

	// mirror these changes in the newly pushed ticker object
	ticker.Max = lastTickerData.Max
	ticker.Min = lastTickerData.Min
}

func (i *Importer) getLastTicker(tickerName domain.TickerName) (*domain.Ticker, error) {
	history := i.tickerHistory.Get(tickerName)
	lastTicker, exists := history.Last()
	if !exists {
		return nil, fmt.Errorf("no ticker history found for %s", tickerName)
	}
	return lastTicker, nil
}

func (i *Importer) getLastTick() (*domain.Tick, error) {
	lastTick, exists := i.tickHistory.Last()
	if !exists {
		return nil, fmt.Errorf("no tick history found")
	}
	return lastTick, nil
}
