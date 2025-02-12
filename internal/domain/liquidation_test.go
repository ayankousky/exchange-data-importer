package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLiquidation_Validate(t *testing.T) {
	defaultDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	validLongLiquidation := Order{
		EventAt:    defaultDate,
		Symbol:     "BTCUSDT",
		Side:       OrderSide(LongLiquidation),
		Price:      50000.0,
		Quantity:   1.0,
		TotalPrice: 50000.0,
	}
	validShortLiquidation := Order{
		EventAt:    defaultDate,
		Symbol:     "BTCUSDT",
		Side:       OrderSide(ShortLiquidation),
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
				Order:    validLongLiquidation,
				EventAt:  defaultDate,
				StoredAt: defaultDate.Add(time.Second),
			},
			wantErr: false,
		},
		{
			name: "valid short liquidation",
			liq: Liquidation{
				Order:    validShortLiquidation,
				EventAt:  defaultDate,
				StoredAt: defaultDate.Add(time.Second),
			},
			wantErr: false,
		},
		{
			name: "zero EventAt time",
			liq: Liquidation{
				Order:    validLongLiquidation,
				EventAt:  time.Time{},
				StoredAt: defaultDate,
			},
			wantErr:  true,
			errField: "EventAt",
		},
		{
			name: "zero StoredAt time",
			liq: Liquidation{
				Order:    validLongLiquidation,
				EventAt:  defaultDate,
				StoredAt: time.Time{},
			},
			wantErr:  true,
			errField: "StoredAt",
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
