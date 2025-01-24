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
	GetName() string
	FetchTickers(ctx context.Context) ([]Ticker, error)
}
