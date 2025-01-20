package mock

import (
	"context"
	"fmt"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges"
	"github.com/ayankousky/exchange-data-importer/pkg/utils/mathutils"
)

// Client is a Binance client to access Binance data
type Client struct {
	name string

	data   [][]exchanges.Ticker // exists only in mock
	cursor int                  // exists only in mock
}

// NewMockClient creates a new mock client
func NewMockClient(name string) *Client {
	client := &Client{name: name}
	client.GenerateData(10)
	return client
}

// GetName mock method
func (bc *Client) GetName() string {
	return bc.name
}

// FetchTickers mock method
func (bc *Client) FetchTickers(_ context.Context) ([]exchanges.Ticker, error) {
	if len(bc.data) <= bc.cursor {
		return nil, fmt.Errorf("no data available")
	}
	tickers := bc.data[bc.cursor]
	bc.cursor++
	return tickers, nil
}

// GenerateData populates the mock data
func (bc *Client) GenerateData(i int) {
	bc.data = make([][]exchanges.Ticker, i)
	for j := 0; j < i; j++ {
		multiplier := float64(1 + float64(j)/200)
		bc.data[j] = []exchanges.Ticker{
			{
				Symbol:      "BTCUSDT",
				BidPrice:    mathutils.Round(104388.6*multiplier, 4),
				BidQuantity: mathutils.Round(0.002*multiplier*10, 8),
				AskPrice:    mathutils.Round(104388.7*multiplier, 4),
				AskQuantity: mathutils.Round(0.002*multiplier*10, 8),
			},
			{
				Symbol:      "ETHUSDT",
				BidPrice:    mathutils.Round(3345.15*multiplier, 4),
				BidQuantity: mathutils.Round(0.02*multiplier*10, 8),
				AskPrice:    mathutils.Round(3345.16*multiplier, 4),
				AskQuantity: mathutils.Round(0.02*multiplier*10, 8),
			},
		}
	}
	bc.cursor = 0
}
