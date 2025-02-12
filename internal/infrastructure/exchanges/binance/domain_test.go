package binance

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
				Time:        1635739200000,
			},
			want: exchanges.Ticker{
				Symbol:      "BTCUSDT",
				BidPrice:    50000.50,
				BidQuantity: 1.5,
				AskPrice:    50000.75,
				AskQuantity: 2.5,
				EventAt:     time.UnixMilli(1635739200000),
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
				Time:        2000,
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
				Time:        2000,
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
				EventType: "forceOrder",
				EventTime: 1635739200000,
				OrderData: struct {
					Symbol       string `json:"s"`
					Side         string `json:"S"`
					OrderType    string `json:"o"`
					TimeInForce  string `json:"f"`
					OrigQuantity string `json:"q"`
					Price        string `json:"p"`
					AveragePrice string `json:"ap"`
					OrderStatus  string `json:"X"`
					LastQuantity string `json:"l"`
					Time         int64  `json:"T"`
				}{
					Symbol:       "BTCUSDT",
					Side:         "SELL",
					OrigQuantity: "0.001",
					Price:        "50000.50",
				},
			},
			want: exchanges.Liquidation{
				Symbol:     "BTCUSDT",
				Side:       "SELL",
				Price:      50000.50,
				Quantity:   0.001,
				EventAt:    time.UnixMilli(1635739200000),
				TotalPrice: 50.0005,
			},
			wantErr: false,
		},
		{
			name: "invalid price",
			dto: LiquidationDTO{
				EventTime: 1635739200000,
				OrderData: struct {
					Symbol       string `json:"s"`
					Side         string `json:"S"`
					OrderType    string `json:"o"`
					TimeInForce  string `json:"f"`
					OrigQuantity string `json:"q"`
					Price        string `json:"p"`
					AveragePrice string `json:"ap"`
					OrderStatus  string `json:"X"`
					LastQuantity string `json:"l"`
					Time         int64  `json:"T"`
				}{
					Symbol:       "BTCUSDT",
					Side:         "SELL",
					OrigQuantity: "0.001",
					Price:        "invalid",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid quantity",
			dto: LiquidationDTO{
				EventTime: 3000,
				OrderData: struct {
					Symbol       string `json:"s"`
					Side         string `json:"S"`
					OrderType    string `json:"o"`
					TimeInForce  string `json:"f"`
					OrigQuantity string `json:"q"`
					Price        string `json:"p"`
					AveragePrice string `json:"ap"`
					OrderStatus  string `json:"X"`
					LastQuantity string `json:"l"`
					Time         int64  `json:"T"`
				}{
					Symbol:       "BTCUSDT",
					Side:         "BUY",
					Price:        "40000.0",
					OrigQuantity: "invalid",
				},
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
