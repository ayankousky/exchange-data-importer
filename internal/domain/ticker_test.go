package domain

import (
	"testing"
	"time"

	"github.com/ayankousky/exchange-data-importer/pkg/utils"
	"github.com/ayankousky/exchange-data-importer/pkg/utils/mathutils"
	"github.com/stretchr/testify/assert"
)

func TestTicker_CalculateIndicators(t *testing.T) {
	// Prepare test data
	historySize := 32
	history := utils.NewRingBuffer[*Ticker](historySize)
	for i := 0; i < historySize; i++ {
		history.Push(&Ticker{
			Symbol:    "BTCUSDT",
			Ask:       mathutils.Round(100*float64(i), 2),
			Bid:       mathutils.Round(99*float64(i), 2),
			Change1m:  mathutils.Round(0.1*float64(i), 2),
			Change20m: mathutils.Round(0.2*float64(i), 2),
			Max10:     mathutils.Round(101*float64(i), 2),
			Min10:     mathutils.Round(98*float64(i), 2),
		})
	}

	// Execute CalculateIndicators
	ticker, _ := history.Last()
	prevTick := &Tick{Data: map[TickerName]*Ticker{
		"BTCUSDT": {
			Symbol:    "BTCUSDT",
			Ask:       mathutils.Round(ticker.Ask*0.99, 4),
			Bid:       mathutils.Round(ticker.Bid*0.99, 4),
			Change1m:  mathutils.Round(ticker.Change1m*0.99, 4),
			Change20m: mathutils.Round(ticker.Change20m*0.99, 4),
			Max10:     mathutils.Round(ticker.Max10*0.99, 4),
			Min10:     mathutils.Round(ticker.Min10*0.99, 4),
		}}}
	ticker.CalculateIndicators(history, prevTick)

	// Validate results
	assert.Equal(t, 3069.0, ticker.Bid, "Bid should remain unchanged")
	assert.Equal(t, 3100.0, ticker.Ask, "Ask should remain unchanged")
	assert.Equal(t, 3.33, ticker.Change1m, "Change1m should match expected value")
	assert.Equal(t, 181.82, ticker.Change20m, "Change20m should match expected value")
	assert.Equal(t, 3100.0, ticker.Max10, "Max10 should match expected value")
	assert.Equal(t, 2200.0, ticker.Min10, "Min10 should match expected value")
	assert.Equal(t, 0.0, ticker.Max10Diff, "Max10Diff should match expected value")
	assert.Equal(t, 40.91, ticker.Min10Diff, "Min10Diff should match expected value")

	ticker.Ask = ticker.Ask * 0.9
	ticker.CalculateIndicators(history, prevTick)
	assert.Equal(t, -7.0, ticker.Max10Diff, "Max10Diff should increase negative if ask reduced")
	assert.Equal(t, 26.82, ticker.Min10Diff, "Min10Diff should reduce if ask reduced")
}

func TestTicker_Validate(t *testing.T) {
	defaultDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		ticker   Ticker
		wantErr  bool
		errField string
	}{
		{
			name: "valid ticker",
			ticker: Ticker{
				Symbol:    "BTCUSDT",
				EventAt:   defaultDate,
				CreatedAt: defaultDate,
				Ask:       50000.0,
				Bid:       49900.0,
				RSI20:     60.0,
			},
			wantErr: false,
		},
		{
			name: "empty symbol",
			ticker: Ticker{
				Symbol:    "",
				EventAt:   defaultDate,
				CreatedAt: defaultDate,
				Ask:       50000.0,
				Bid:       49900.0,
			},
			wantErr:  true,
			errField: "Symbol",
		},
		{
			name: "zero event time",
			ticker: Ticker{
				Symbol:    "BTCUSDT",
				EventAt:   time.Time{},
				CreatedAt: defaultDate,
				Ask:       50000.0,
				Bid:       49900.0,
			},
			wantErr:  true,
			errField: "EventAt",
		},
		{
			name: "zero CreatedAt time",
			ticker: Ticker{
				Symbol:    "BTCUSDT",
				EventAt:   defaultDate,
				CreatedAt: time.Time{},
				Ask:       50000.0,
				Bid:       49900.0,
			},
			wantErr:  true,
			errField: "CreatedAt",
		},
		{
			name: "negative ask price",
			ticker: Ticker{
				Symbol:    "BTCUSDT",
				EventAt:   defaultDate,
				CreatedAt: defaultDate,
				Ask:       -50000.0,
				Bid:       49900.0,
			},
			wantErr:  true,
			errField: "Ask",
		},
		{
			name: "negative bid price",
			ticker: Ticker{
				Symbol:    "BTCUSDT",
				EventAt:   defaultDate,
				CreatedAt: defaultDate,
				Ask:       50000.0,
				Bid:       -49900.0,
			},
			wantErr:  true,
			errField: "Bid",
		},
		{
			name: "bid greater than ask",
			ticker: Ticker{
				Symbol:    "BTCUSDT",
				EventAt:   defaultDate,
				CreatedAt: defaultDate,
				Ask:       50000.0,
				Bid:       50100.0,
			},
			wantErr:  true,
			errField: "Bid/Ask",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ticker.Validate()
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

func TestTicker_CalculateIndicators_EdgeCases(t *testing.T) {
	// Test for nil safety checks
	t.Run("nil ticker", func(t *testing.T) {
		var ticker *Ticker
		history := utils.NewRingBuffer[*Ticker](10)
		lastTick := &Tick{Data: map[TickerName]*Ticker{"BTCUSDT": {Symbol: "BTCUSDT"}}}

		// This should not panic
		ticker.CalculateIndicators(history, lastTick)
	})

	t.Run("nil lastTick", func(t *testing.T) {
		ticker := &Ticker{Symbol: "BTCUSDT", Ask: 100, Bid: 99}
		history := utils.NewRingBuffer[*Ticker](10)

		// This should not panic
		ticker.CalculateIndicators(history, nil)

		// Values should remain unchanged
		assert.Equal(t, TickerName("BTCUSDT"), ticker.Symbol)
		assert.Equal(t, 100.0, ticker.Ask)
		assert.Equal(t, 99.0, ticker.Bid)
		assert.Equal(t, 0.0, ticker.Change1m)
	})

	t.Run("symbol not in lastTick", func(t *testing.T) {
		ticker := &Ticker{Symbol: "BTCUSDT", Ask: 100, Bid: 99}
		history := utils.NewRingBuffer[*Ticker](10)
		lastTick := &Tick{Data: map[TickerName]*Ticker{"ETHUSDT": {Symbol: "ETHUSDT"}}}

		// This should not panic
		ticker.CalculateIndicators(history, lastTick)

		// Values should remain unchanged
		assert.Equal(t, TickerName("BTCUSDT"), ticker.Symbol)
		assert.Equal(t, 100.0, ticker.Ask)
		assert.Equal(t, 99.0, ticker.Bid)
		assert.Equal(t, 0.0, ticker.Change1m)
	})

	t.Run("nil lastTick.Data", func(t *testing.T) {
		ticker := &Ticker{Symbol: "BTCUSDT", Ask: 100, Bid: 99}
		history := utils.NewRingBuffer[*Ticker](10)
		lastTick := &Tick{} // Data is nil

		// This should not panic
		ticker.CalculateIndicators(history, lastTick)

		// Values should remain unchanged
		assert.Equal(t, TickerName("BTCUSDT"), ticker.Symbol)
		assert.Equal(t, 100.0, ticker.Ask)
		assert.Equal(t, 99.0, ticker.Bid)
		assert.Equal(t, 0.0, ticker.Change1m)
	})

	t.Run("history length < 2", func(t *testing.T) {
		ticker := &Ticker{Symbol: "BTCUSDT", Ask: 100, Bid: 99}
		history := utils.NewRingBuffer[*Ticker](10)

		// Add just one item to history
		history.Push(&Ticker{Symbol: "BTCUSDT", Ask: 95, Bid: 94})

		lastTick := &Tick{Data: map[TickerName]*Ticker{
			"BTCUSDT": {Symbol: "BTCUSDT", Ask: 95, Bid: 94},
		}}

		// This should not compute anything
		ticker.CalculateIndicators(history, lastTick)

		// Values should remain unchanged
		assert.Equal(t, 0.0, ticker.Change1m)
		assert.Equal(t, 0.0, ticker.Change20m)
		assert.Equal(t, 0.0, ticker.Max10)
		assert.Equal(t, 0.0, ticker.Min10)
	})

	t.Run("history length between 2 and 21", func(t *testing.T) {
		ticker := &Ticker{Symbol: "BTCUSDT", Ask: 100, Bid: 99}
		history := utils.NewRingBuffer[*Ticker](10)

		// Add 5 items to history
		for i := 0; i < 5; i++ {
			history.Push(&Ticker{
				Symbol: "BTCUSDT",
				Ask:    95.0 + float64(i),
				Bid:    94.0 + float64(i),
			})
		}

		lastTick := &Tick{Data: map[TickerName]*Ticker{
			"BTCUSDT": {Symbol: "BTCUSDT", Ask: 95, Bid: 94},
		}}

		// Execute
		ticker.CalculateIndicators(history, lastTick)

		// Should calculate 1m change but not 20m change
		assert.NotEqual(t, 0.0, ticker.Change1m)
		assert.Equal(t, 0.0, ticker.Change20m)
		assert.Equal(t, 0.0, ticker.RSI20)

		// Max10 and Min10 should be calculated
		assert.NotEqual(t, 0.0, ticker.Max10)
		assert.NotEqual(t, 0.0, ticker.Min10)
	})

	t.Run("history length >= 21", func(t *testing.T) {
		ticker := &Ticker{Symbol: "BTCUSDT", Ask: 100, Bid: 99}
		history := utils.NewRingBuffer[*Ticker](30)

		// Add 25 items to history with increasing values
		for i := 0; i < 25; i++ {
			history.Push(&Ticker{
				Symbol: "BTCUSDT",
				Ask:    95.0 + float64(i),
				Bid:    94.0 + float64(i),
			})
		}

		lastTick := &Tick{Data: map[TickerName]*Ticker{
			"BTCUSDT": {Symbol: "BTCUSDT", Ask: 99, Bid: 98},
		}}

		// Execute
		ticker.CalculateIndicators(history, lastTick)

		// Should calculate all indicators
		assert.NotEqual(t, 0.0, ticker.Change1m)
		assert.NotEqual(t, 0.0, ticker.Change20m)
		assert.NotEqual(t, 0.0, ticker.RSI20)
		assert.NotEqual(t, 0.0, ticker.Max10)
		assert.NotEqual(t, 0.0, ticker.Min10)

		// RSI for continuously increasing values should be near 100
		assert.InDelta(t, 100.0, ticker.RSI20, 5.0)
	})
}
