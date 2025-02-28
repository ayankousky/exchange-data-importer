package domain

import (
	"testing"
	"time"

	"github.com/ayankousky/exchange-data-importer/pkg/utils"
	"github.com/ayankousky/exchange-data-importer/pkg/utils/mathutils"
	"github.com/stretchr/testify/assert"
)

func TestCalculateIndicators(t *testing.T) {
	// Prepare test data
	historySize := 10
	history := utils.NewRingBuffer[*Tick](historySize)
	for i := 0; i < historySize; i++ {
		history.Push(&Tick{
			Data: map[TickerName]*Ticker{
				"BTCUSDT": {
					Symbol:    "BTCUSDT",
					Ask:       mathutils.Round(100+float64(i), 2),
					Bid:       mathutils.Round(99+float64(i), 2),
					Change1m:  mathutils.Round(0.1*float64(i), 2),
					Change20m: mathutils.Round(0.1*float64(i*2), 2),
					Max10:     mathutils.Round(-0.1*float64(i), 2),
					Min10:     mathutils.Round(0.2*float64(i), 2),
				},
				"ETHUSDT": {
					Symbol:    "ETHUSDT",
					Ask:       mathutils.Round(200+float64(i), 2),
					Bid:       mathutils.Round(199+float64(i), 2),
					Change1m:  mathutils.Round(0.2*float64(i), 2),
					Change20m: mathutils.Round(0.2*float64(i*2), 2),
					Max10:     mathutils.Round(-0.2*float64(i), 2),
					Min10:     mathutils.Round(0.1*float64(i), 2),
				},
			},
			Avg: TickAvg{
				Change1m:     mathutils.Round(float64(i), 2),
				Change20m:    mathutils.Round(float64(i*2), 2),
				Max10:        mathutils.Round(-0.1*float64(i), 2),
				Min10:        mathutils.Round(0.1*float64(i), 2),
				BidChange:    mathutils.Round(0.1*float64(i), 2),
				AskChange:    mathutils.Round(0.1*float64(i), 2),
				TickersCount: 2,
			},
		})
	}

	// Execute CalculateIndicators
	currentTick, _ := history.Last()
	currentTick.CalculateIndicators(history)

	// Validate results
	assert.Equal(t, 0.45, currentTick.AvgBuy10, "AvgBuy10 should match expected value")
	assert.Equal(t, 0.705, currentTick.Avg.BidChange, "BidChange should match expected value")
	assert.Equal(t, 0.705, currentTick.Avg.AskChange, "AskChange should match expected value")
	assert.Equal(t, 1.35, currentTick.Avg.Change1m, "Change1m should match expected value")
	assert.Equal(t, 2.7, currentTick.Avg.Change20m, "Change20m should match expected value")
	assert.InDelta(t, -11961.11, currentTick.Avg.Max10, 0.01, "Max10 should match expected value")
	assert.InDelta(t, 14538.89, currentTick.Avg.Min10, 0.01, "Min10 should match expected value")
	assert.Equal(t, int16(2), currentTick.Avg.TickersCount, "TickersCount should match expected value")

	currentTick.Data["BTCUSDT"].Ask *= 10
	currentTick.CalculateIndicators(history)
	assert.Equal(t, 0.74, currentTick.Avg.AskChange, "Cover the case when diff more than 1% BidChange")

	currentTick.Data["BTCUSDT"].Ask /= 100
	currentTick.CalculateIndicators(history)
	assert.Equal(t, -0.26, currentTick.Avg.AskChange, "Cover the case when diff more than 1% BidChange")
}

func TestTick_Validate(t *testing.T) {
	defaultDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	validTicker := &Ticker{
		Symbol:    "BTCUSDT",
		EventAt:   defaultDate,
		CreatedAt: defaultDate,
		Ask:       50000.0,
		Bid:       49900.0,
		RSI20:     60.0,
	}

	tests := []struct {
		name     string
		tick     Tick
		wantErr  bool
		errField string
	}{
		{
			name: "valid tick",
			tick: Tick{
				StartAt:          defaultDate,
				FetchedAt:        defaultDate.Add(time.Millisecond * 100),
				CreatedAt:        defaultDate.Add(time.Millisecond * 200),
				FetchDuration:    100,
				HandlingDuration: 200,
				Data: map[TickerName]*Ticker{
					"BTCUSDT": validTicker,
				},
				Avg: TickAvg{
					TickersCount: 1,
				},
			},
			wantErr: false,
		},
		{
			name: "zero startAt time",
			tick: Tick{
				StartAt:          time.Time{},
				FetchedAt:        defaultDate,
				CreatedAt:        defaultDate,
				FetchDuration:    100,
				HandlingDuration: 200,
				Data: map[TickerName]*Ticker{
					"BTCUSDT": validTicker,
				},
				Avg: TickAvg{
					TickersCount: 1,
				},
			},
			wantErr:  true,
			errField: "StartAt",
		},
		{
			name: "zero fetchedAt time",
			tick: Tick{
				StartAt:          defaultDate,
				FetchedAt:        time.Time{},
				CreatedAt:        defaultDate,
				FetchDuration:    100,
				HandlingDuration: 200,
				Data: map[TickerName]*Ticker{
					"BTCUSDT": validTicker,
				},
				Avg: TickAvg{
					TickersCount: 1,
				},
			},
			wantErr:  true,
			errField: "FetchedAt",
		},
		{
			name: "zero CreatedAt time",
			tick: Tick{
				StartAt:          defaultDate,
				FetchedAt:        defaultDate,
				CreatedAt:        time.Time{},
				FetchDuration:    100,
				HandlingDuration: 200,
				Data: map[TickerName]*Ticker{
					"BTCUSDT": validTicker,
				},
				Avg: TickAvg{
					TickersCount: 1,
				},
			},
			wantErr:  true,
			errField: "CreatedAt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tick.Validate()
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

func TestCalculateIndicators_EdgeCases(t *testing.T) {
	t.Run("empty history", func(t *testing.T) {
		// Create a new empty history
		history := utils.NewRingBuffer[*Tick](MaxTickHistory)

		// Create a new tick with initial values
		tick := &Tick{
			Avg: TickAvg{
				AskChange: 0.5,
				BidChange: 0.5,
			},
			Data: map[TickerName]*Ticker{
				"BTCUSDT": {
					Symbol: "BTCUSDT",
					Ask:    1000,
					Bid:    990,
				},
			},
		}

		// Initial values that should remain unchanged
		initialAvgAskChange := tick.Avg.AskChange
		initialAvgBidChange := tick.Avg.BidChange

		// Call CalculateIndicators with empty history
		tick.CalculateIndicators(history)

		// Values should remain unchanged
		assert.Equal(t, initialAvgAskChange, tick.Avg.AskChange, "AskChange should remain unchanged with empty history")
		assert.Equal(t, initialAvgBidChange, tick.Avg.BidChange, "BidChange should remain unchanged with empty history")
		assert.Equal(t, 0.0, tick.AvgBuy10, "AvgBuy10 should be zero with empty history")
	})

	t.Run("history with only one item", func(t *testing.T) {
		// Create history with just one item
		history := utils.NewRingBuffer[*Tick](MaxTickHistory)
		tick := &Tick{
			Avg: TickAvg{
				AskChange: 0.5,
				BidChange: 0.5,
			},
			Data: map[TickerName]*Ticker{
				"BTCUSDT": {
					Symbol: "BTCUSDT",
					Ask:    1000,
					Bid:    990,
				},
			},
		}
		history.Push(tick)

		// Initial values that should remain unchanged
		initialAvgAskChange := tick.Avg.AskChange
		initialAvgBidChange := tick.Avg.BidChange

		// Call CalculateIndicators with history of length 1
		tick.CalculateIndicators(history)

		// Values should remain unchanged
		assert.Equal(t, initialAvgAskChange, tick.Avg.AskChange, "AskChange should remain unchanged with history length of 1")
		assert.Equal(t, initialAvgBidChange, tick.Avg.BidChange, "BidChange should remain unchanged with history length of 1")
		assert.Equal(t, 0.0, tick.AvgBuy10, "AvgBuy10 should be zero with history length of 1")
	})

	t.Run("history with new ticker not in previous tick", func(t *testing.T) {
		// Create history with two items
		history := utils.NewRingBuffer[*Tick](MaxTickHistory)

		// First tick with only ETHUSDT
		firstTick := &Tick{
			Data: map[TickerName]*Ticker{
				"ETHUSDT": {
					Symbol: "ETHUSDT",
					Ask:    2000,
					Bid:    1990,
				},
			},
			Avg: TickAvg{
				TickersCount: 1,
			},
		}
		history.Push(firstTick)

		// Second tick with both ETHUSDT and BTCUSDT (new ticker)
		secondTick := &Tick{
			Data: map[TickerName]*Ticker{
				"ETHUSDT": {
					Symbol:    "ETHUSDT",
					Ask:       2100,
					Bid:       2090,
					Change1m:  1.5,
					Change20m: 2.5,
					Max10:     2200,
					Min10:     2000,
				},
				"BTCUSDT": { // New ticker not in previous tick
					Symbol:    "BTCUSDT",
					Ask:       30000,
					Bid:       29900,
					Change1m:  0.5,
					Change20m: 1.0,
					Max10:     31000,
					Min10:     29000,
				},
			},
			Avg: TickAvg{
				TickersCount: 2,
			},
		}
		history.Push(secondTick)

		// Call CalculateIndicators
		secondTick.CalculateIndicators(history)

		// Only ETHUSDT should contribute to the averages
		// BTCUSDT should be skipped since it's not in the previous tick
		assert.Equal(t, int16(1), secondTick.Avg.TickersCount, "Only one ticker should be counted in averages")
	})
}
