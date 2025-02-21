package strategies

import (
	"errors"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/notify"
	"github.com/ayankousky/exchange-data-importer/internal/notifier"
)

// TickerNotification represents a ticker notification event without excessive data
type TickerNotification struct {
	Tick   domain.Tick   `json:"tick"`
	Ticker domain.Ticker `json:"ticker"`
}

// newTickerNotification creates a new TickerNotification from a tick and a given ticker name
func newTickerNotification(tick *domain.Tick, symbol domain.TickerName) (*TickerNotification, error) {
	if tick == nil {
		return nil, errors.New("tick cannot be nil")
	}

	tickCopy := *tick
	tickCopy.Data = nil

	if _, exists := tick.Data[symbol]; !exists {
		return nil, errors.New("ticker not found in tick data")
	}

	notification := &TickerNotification{
		Tick:   tickCopy,
		Ticker: *tick.Data[symbol],
	}

	return notification, nil
}

// MarketDataStrategy sends updates about every ticker for every subscribed service
type MarketDataStrategy struct{}

// Format formats the tick data into a human-readable format
func (s *MarketDataStrategy) Format(data any) []notify.Event {
	tick, ok := data.(*domain.Tick)
	if !ok {
		return nil
	}

	if tick == nil {
		return nil
	}

	events := make([]notify.Event, 0, len(tick.Data))
	for symbol := range tick.Data {
		notification, err := newTickerNotification(tick, symbol)
		if err != nil {
			continue
		}

		events = append(events, notify.Event{
			Time:      time.Now(),
			EventType: string(notifier.MarketDataTopic),
			Data:      notification,
		})
	}

	return events
}
