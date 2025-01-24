package domain

import (
	"testing"

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
