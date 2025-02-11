package domain

import "errors"

const (
	// EventTypeTicker is the event type for ticker data
	EventTypeTicker = "TICKER"
)

// TickerNotification represents a ticker notification event without excessive data
type TickerNotification struct {
	Tick   Tick   `json:"tick"`
	Ticker Ticker `json:"ticker"`
}

// NewTickerNotification creates a new TickerNotification from a tick and a given ticker name
func NewTickerNotification(tick *Tick, symbol TickerName) (*TickerNotification, error) {
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
