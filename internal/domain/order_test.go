package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOrder_Validate(t *testing.T) {
	defaultDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		order    Order
		wantErr  bool
		errField string
	}{
		{
			name: "valid order",
			order: Order{
				EventAt:    defaultDate,
				Symbol:     "BTCUSDT",
				Side:       OrderSideBuy,
				Price:      50000.0,
				Quantity:   1.0,
				TotalPrice: 50000.0,
			},
			wantErr: false,
		},
		{
			name: "zero event time",
			order: Order{
				EventAt:    time.Time{},
				Symbol:     "BTCUSDT",
				Side:       OrderSideBuy,
				Price:      50000.0,
				Quantity:   1.0,
				TotalPrice: 50000.0,
			},
			wantErr:  true,
			errField: "EventAt",
		},
		{
			name: "empty symbol",
			order: Order{
				EventAt:    defaultDate,
				Symbol:     "",
				Side:       OrderSideBuy,
				Price:      50000.0,
				Quantity:   1.0,
				TotalPrice: 50000.0,
			},
			wantErr:  true,
			errField: "Symbol",
		},
		{
			name: "invalid side",
			order: Order{
				EventAt:    defaultDate,
				Symbol:     "BTCUSDT",
				Side:       "INVALID",
				Price:      50000.0,
				Quantity:   1.0,
				TotalPrice: 50000.0,
			},
			wantErr:  true,
			errField: "Side",
		},
		{
			name: "invalid price",
			order: Order{
				EventAt:    defaultDate,
				Symbol:     "BTCUSDT",
				Side:       OrderSideBuy,
				Price:      0,
				Quantity:   1.0,
				TotalPrice: 40000.0,
			},
			wantErr:  true,
			errField: "Price",
		},
		{
			name: "invalid quantity",
			order: Order{
				EventAt:    defaultDate,
				Symbol:     "BTCUSDT",
				Side:       OrderSideBuy,
				Price:      50000.0,
				Quantity:   0,
				TotalPrice: 40000.0,
			},
			wantErr:  true,
			errField: "Quantity",
		},
		{
			name: "invalid total price",
			order: Order{
				EventAt:    defaultDate,
				Symbol:     "BTCUSDT",
				Side:       OrderSideBuy,
				Price:      50000.0,
				Quantity:   1.0,
				TotalPrice: 40000.0,
			},
			wantErr:  true,
			errField: "TotalPrice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.order.Validate()
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
