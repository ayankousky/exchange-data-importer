package okx

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
				InstID:      "BTC-USDT-SWAP",
				LastPrice:   "50000.50",
				BidPrice:    "50000.25",
				BidQuantity: "1.5",
				AskPrice:    "50000.75",
				AskQuantity: "2.5",
				Timestamp:   "1635739200000",
			},
			want: exchanges.Ticker{
				Symbol:      "BTC-USDT-SWAP",
				BidPrice:    50000.25,
				BidQuantity: 1.5,
				AskPrice:    50000.75,
				AskQuantity: 2.5,
				EventAt:     time.Unix(0, 1635739200000*int64(time.Millisecond)),
			},
			wantErr: false,
		},
		{
			name: "invalid ask price",
			dto: TickerDTO{
				InstID:      "BTC-USDT-SWAP",
				BidPrice:    "50000.50",
				BidQuantity: "1.5",
				AskPrice:    "invalid",
				AskQuantity: "2.5",
				Timestamp:   "1635739200000",
			},
			want:    exchanges.Ticker{},
			wantErr: true,
		},
		{
			name: "invalid bid price",
			dto: TickerDTO{
				InstID:      "BTC-USDT-SWAP",
				BidPrice:    "invalid",
				BidQuantity: "1.5",
				AskPrice:    "50000.75",
				AskQuantity: "2.5",
				Timestamp:   "1635739200000",
			},
			want:    exchanges.Ticker{},
			wantErr: true,
		},
		{
			name: "invalid ask quantity",
			dto: TickerDTO{
				InstID:      "BTC-USDT-SWAP",
				BidPrice:    "40000.0",
				BidQuantity: "1.0",
				AskPrice:    "40010.0",
				AskQuantity: "not-a-number",
				Timestamp:   "2000",
			},
			want:    exchanges.Ticker{},
			wantErr: true,
		},
		{
			name: "invalid bid quantity",
			dto: TickerDTO{
				InstID:      "BTC-USDT-SWAP",
				BidPrice:    "40000.0",
				BidQuantity: "not-a-number",
				AskPrice:    "40010.0",
				AskQuantity: "1.0",
				Timestamp:   "2000",
			},
			want:    exchanges.Ticker{},
			wantErr: true,
		},
		{
			name: "invalid timestamp",
			dto: TickerDTO{
				InstID:      "BTC-USDT-SWAP",
				BidPrice:    "40000.0",
				BidQuantity: "1.0",
				AskPrice:    "40010.0",
				AskQuantity: "1.0",
				Timestamp:   "invalid",
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
			name: "valid conversion - sell",
			dto: LiquidationDTO{
				InstID: "BTC-USDT-SWAP",
				Details: []struct {
					Side      string `json:"side"`
					Quantity  string `json:"sz"`
					Timestamp string `json:"ts"`
					Price     string `json:"bkPx"`
				}{
					{
						Side:      "sell",
						Quantity:  "0.001",
						Price:     "50000.50",
						Timestamp: "1635739200000",
					},
				},
			},
			want: exchanges.Liquidation{
				Symbol:     "BTC-USDT-SWAP",
				Side:       "SELL",
				Price:      50000.50,
				Quantity:   0.001,
				EventAt:    time.Unix(0, 1635739200000*int64(time.Millisecond)),
				TotalPrice: 50.0005,
			},
			wantErr: false,
		},
		{
			name: "valid conversion - buy",
			dto: LiquidationDTO{
				InstID: "BTC-USDT-SWAP",
				Details: []struct {
					Side      string `json:"side"`
					Quantity  string `json:"sz"`
					Timestamp string `json:"ts"`
					Price     string `json:"bkPx"`
				}{
					{
						Side:      "buy",
						Quantity:  "0.001",
						Price:     "50000.50",
						Timestamp: "1635739200000",
					},
				},
			},
			want: exchanges.Liquidation{
				Symbol:     "BTC-USDT-SWAP",
				Side:       "BUY",
				Price:      50000.50,
				Quantity:   0.001,
				EventAt:    time.Unix(0, 1635739200000*int64(time.Millisecond)),
				TotalPrice: 50.0005,
			},
			wantErr: false,
		},
		{
			name: "invalid price",
			dto: LiquidationDTO{
				InstID: "BTC-USDT-SWAP",
				Details: []struct {
					Side      string `json:"side"`
					Quantity  string `json:"sz"`
					Timestamp string `json:"ts"`
					Price     string `json:"bkPx"`
				}{
					{
						Side:      "buy",
						Quantity:  "0.001",
						Price:     "invalid",
						Timestamp: "1635739200000",
					},
				},
			},
			want:    exchanges.Liquidation{},
			wantErr: true,
		},
		{
			name: "invalid quantity",
			dto: LiquidationDTO{
				InstID: "BTC-USDT-SWAP",
				Details: []struct {
					Side      string `json:"side"`
					Quantity  string `json:"sz"`
					Timestamp string `json:"ts"`
					Price     string `json:"bkPx"`
				}{
					{
						Side:      "sell",
						Quantity:  "invalid",
						Price:     "50000.50",
						Timestamp: "1635739200000",
					},
				},
			},
			want:    exchanges.Liquidation{},
			wantErr: true,
		},
		{
			name: "invalid timestamp",
			dto: LiquidationDTO{
				InstID: "BTC-USDT-SWAP",
				Details: []struct {
					Side      string `json:"side"`
					Quantity  string `json:"sz"`
					Timestamp string `json:"ts"`
					Price     string `json:"bkPx"`
				}{
					{
						Side:      "sell",
						Quantity:  "0.001",
						Price:     "50000.50",
						Timestamp: "invalid",
					},
				},
			},
			want:    exchanges.Liquidation{},
			wantErr: true,
		},
		{
			name: "invalid side",
			dto: LiquidationDTO{
				InstID: "BTC-USDT-SWAP",
				Details: []struct {
					Side      string `json:"side"`
					Quantity  string `json:"sz"`
					Timestamp string `json:"ts"`
					Price     string `json:"bkPx"`
				}{
					{
						Side:      "invalid",
						Quantity:  "0.001",
						Price:     "50000.50",
						Timestamp: "1635739200000",
					},
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
