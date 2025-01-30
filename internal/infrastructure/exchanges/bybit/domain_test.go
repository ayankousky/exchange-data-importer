package bybit

import (
	"testing"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTickerDTO_ToTicker(t *testing.T) {
	tests := []struct {
		name    string
		dto     TickerDTO
		want    exchanges.Ticker
		wantErr bool
	}{
		{
			name: "valid conversion",
			dto: TickerDTO{
				Symbol:      "BTCUSDT",
				BidPrice:    "50000.50",
				BidQuantity: "1.5",
				AskPrice:    "50000.75",
				AskQuantity: "2.5",
				LastPrice:   "50000.60",
			},
			want: exchanges.Ticker{
				Symbol:      "BTCUSDT",
				BidPrice:    50000.50,
				BidQuantity: 1.5,
				AskPrice:    50000.75,
				AskQuantity: 2.5,
			},
			wantErr: false,
		},
		{
			name: "invalid ask price",
			dto: TickerDTO{
				Symbol:      "BTCUSDT",
				BidPrice:    "50000.50",
				BidQuantity: "1.5",
				AskPrice:    "invalid",
				AskQuantity: "2.5",
			},
			want:    exchanges.Ticker{},
			wantErr: true,
		},
		{
			name: "invalid bid price",
			dto: TickerDTO{
				Symbol:      "BTCUSDT",
				BidPrice:    "invalid",
				BidQuantity: "1.5",
				AskPrice:    "50000.75",
				AskQuantity: "2.5",
			},
			want:    exchanges.Ticker{},
			wantErr: true,
		},
		{
			name: "invalid ask quantity",
			dto: TickerDTO{
				Symbol:      "BTCUSDT",
				BidPrice:    "40000.0",
				BidQuantity: "1.0",
				AskPrice:    "40010.0",
				AskQuantity: "not-a-number",
			},
			want:    exchanges.Ticker{},
			wantErr: true,
		},
		{
			name: "invalid bid quantity",
			dto: TickerDTO{
				Symbol:      "BTCUSDT",
				BidPrice:    "40000.0",
				BidQuantity: "not-a-number",
				AskPrice:    "40010.0",
				AskQuantity: "1.0",
			},
			want:    exchanges.Ticker{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.dto.toTicker()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLiquidationDTO_ToLiquidation(t *testing.T) {
	tests := []struct {
		name    string
		dto     LiquidationDTO
		want    exchanges.Liquidation
		wantErr bool
	}{
		{
			name: "valid conversion",
			dto: LiquidationDTO{
				Symbol:      "BTCUSDT",
				Side:        "Sell",
				Price:       "50000.50",
				Quantity:    "0.001",
				UpdatedTime: 1635739200000,
			},
			want: exchanges.Liquidation{
				Symbol:   "BTCUSDT",
				Side:     "SELL",
				Price:    50000.50,
				Quantity: 0.001,
				EventAt:  time.UnixMilli(1635739200000),
			},
			wantErr: false,
		},
		{
			name: "invalid price",
			dto: LiquidationDTO{
				Symbol:      "BTCUSDT",
				Side:        "Sell",
				Price:       "invalid",
				Quantity:    "0.001",
				UpdatedTime: 1635739200000,
			},
			wantErr: true,
		},
		{
			name: "invalid quantity",
			dto: LiquidationDTO{
				Symbol:      "BTCUSDT",
				Side:        "Buy",
				Price:       "40000.0",
				Quantity:    "invalid",
				UpdatedTime: 1635739200000,
			},
			want:    exchanges.Liquidation{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.dto.toLiquidation()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
