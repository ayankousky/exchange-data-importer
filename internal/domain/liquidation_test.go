package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLiquidation_Validate(t *testing.T) {
	defaultDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Create valid orders for testing
	validLongLiquidationOrder := Order{
		EventAt:    defaultDate,
		Symbol:     "BTCUSDT",
		Side:       OrderSideSell, // Long liquidation uses SELL side
		Price:      50000.0,
		Quantity:   1.0,
		TotalPrice: 50000.0,
	}

	validShortLiquidationOrder := Order{
		EventAt:    defaultDate,
		Symbol:     "BTCUSDT",
		Side:       OrderSideBuy, // Short liquidation uses BUY side
		Price:      50000.0,
		Quantity:   1.0,
		TotalPrice: 50000.0,
	}

	tests := []struct {
		name     string
		liq      Liquidation
		wantErr  bool
		errField string
	}{
		{
			name: "valid long liquidation",
			liq: Liquidation{
				Order:    validLongLiquidationOrder,
				EventAt:  defaultDate,
				StoredAt: defaultDate.Add(time.Second),
			},
			wantErr: false,
		},
		{
			name: "valid short liquidation",
			liq: Liquidation{
				Order:    validShortLiquidationOrder,
				EventAt:  defaultDate,
				StoredAt: defaultDate.Add(time.Second),
			},
			wantErr: false,
		},
		{
			name: "zero EventAt time",
			liq: Liquidation{
				Order:    validLongLiquidationOrder,
				EventAt:  time.Time{},
				StoredAt: defaultDate,
			},
			wantErr:  true,
			errField: "EventAt",
		},
		{
			name: "zero StoredAt time",
			liq: Liquidation{
				Order:    validLongLiquidationOrder,
				EventAt:  defaultDate,
				StoredAt: time.Time{},
			},
			wantErr:  true,
			errField: "StoredAt",
		},
		{
			name: "invalid order (missing symbol)",
			liq: Liquidation{
				Order: Order{
					EventAt:    defaultDate,
					Symbol:     "", // Invalid - missing symbol
					Side:       OrderSideSell,
					Price:      50000.0,
					Quantity:   1.0,
					TotalPrice: 50000.0,
				},
				EventAt:  defaultDate,
				StoredAt: defaultDate.Add(time.Second),
			},
			wantErr:  true,
			errField: "Order",
		},
		{
			name: "invalid order side",
			liq: Liquidation{
				Order: Order{
					EventAt:    defaultDate,
					Symbol:     "BTCUSDT",
					Side:       "INVALID", // Neither BUY nor SELL
					Price:      50000.0,
					Quantity:   1.0,
					TotalPrice: 50000.0,
				},
				EventAt:  defaultDate,
				StoredAt: defaultDate.Add(time.Second),
			},
			wantErr:  true,
			errField: "Order",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.liq.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				valErr, ok := err.(ValidationError)
				assert.True(t, ok)
				assert.Equal(t, tt.errField, valErr.Field)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
