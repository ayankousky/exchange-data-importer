package exchanges

import (
	"context"
	"time"
)

// Ticker represents a ticker data imported from an exchange
type Ticker struct {
	Symbol      string
	AskPrice    float64
	BidPrice    float64
	AskQuantity float64
	BidQuantity float64
	EventAt     time.Time
}

// Liquidation represents a liquidation data imported from an exchange
type Liquidation struct {
	Symbol     string
	Side       string
	Price      float64
	Quantity   float64
	TotalPrice float64
	EventAt    time.Time
}

// Exchange represents an exchange that can be queried for data
type Exchange interface {
	// GetName returns the name of the exchange
	// Required to create corresponding collections/tables etc
	GetName() string

	// FetchTickers fetches the latest tickers from the exchange
	FetchTickers(ctx context.Context) ([]Ticker, error)

	// SubscribeLiquidations subscribes to liquidation events from the exchange
	SubscribeLiquidations(ctx context.Context) (<-chan Liquidation, <-chan error)
}
