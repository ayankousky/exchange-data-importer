package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateTickerNotification(t *testing.T) {
	tests := []struct {
		name        string
		tick        *Tick
		tickerName  TickerName
		wantTicker  Ticker
		wantTickMap map[TickerName]*Ticker
		expectError bool
	}{
		{
			name: "should create notification with correct data",
			tick: &Tick{
				AvgBuy10: 100.50,
				LL1:      5,
				Avg: TickAvg{
					Change1m:     1.5,
					TickersCount: 2,
				},
				Data: map[TickerName]*Ticker{
					"BTCUSDT": {
						Symbol:   "BTCUSDT",
						Ask:      45000.00,
						Bid:      44990.00,
						Change1m: 0.5,
					},
					"ETHUSDT": {
						Symbol:   "ETHUSDT",
						Ask:      3000.00,
						Bid:      2999.00,
						Change1m: 0.3,
					},
				},
			},
			tickerName: "BTCUSDT",
			wantTicker: Ticker{
				Symbol:   "BTCUSDT",
				Ask:      45000.00,
				Bid:      44990.00,
				Change1m: 0.5,
			},
			wantTickMap: nil,
		},
		{
			name: "should handle empty data map",
			tick: &Tick{
				AvgBuy10: 100.50,
				LL1:      5,
				Avg: TickAvg{
					Change1m:     1.5,
					TickersCount: 0,
				},
				Data: map[TickerName]*Ticker{},
			},
			tickerName:  "BTCUSDT",
			wantTicker:  Ticker{},
			wantTickMap: nil,
			expectError: true,
		},
		{
			name:        "should handle empty tick",
			tick:        nil,
			tickerName:  "BTCUSDT",
			wantTicker:  Ticker{},
			wantTickMap: nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notification, err := NewTickerNotification(tt.tick, tt.tickerName)

			if tt.expectError {
				assert.NotNil(t, err)
				assert.Nil(t, notification)
				return
			}

			// Verify the notification structure
			assert.Equal(t, tt.tick.AvgBuy10, notification.Tick.AvgBuy10)
			assert.Equal(t, tt.tick.LL1, notification.Tick.LL1)
			assert.Equal(t, tt.tick.Avg, notification.Tick.Avg)
			assert.Equal(t, tt.wantTickMap, notification.Tick.Data)

			// Verify the ticker data
			assert.Equal(t, tt.wantTicker, notification.Ticker)

			// Verify that modifying the original tick doesn't affect the notification
			if len(tt.tick.Data) > 0 {
				originalTicker := tt.tick.Data[tt.tickerName]
				originalTicker.Ask = 999999.99
				assert.NotEqual(t, originalTicker.Ask, notification.Ticker.Ask)
			}
		})
	}
}
