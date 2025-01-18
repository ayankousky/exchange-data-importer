package domain

import (
	"github.com/ayankousky/exchange-data-importer/pkg/utils/mathutils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCalculateIndicators(t *testing.T) {
	// Prepare test data
	historySize := 10
	history := make([]*Tick, historySize)
	for i := 0; i < historySize; i++ {
		history[i] = &Tick{
			Data: map[TickerName]*Ticker{
				"BTC": {
					Symbol: "BTC",
					Ask:    mathutils.Round(100+float64(i), 2),
					Bid:    mathutils.Round(99+float64(i), 2),
					Pd:     mathutils.Round(0.1*float64(i), 2),
					Pd20:   mathutils.Round(0.1*float64(i*2), 2),
					Max10:  mathutils.Round(-0.1*float64(i), 2),
					Min10:  mathutils.Round(0.2*float64(i), 2),
				},
				"ETH": {
					Symbol: "ETH",
					Ask:    mathutils.Round(200+float64(i), 2),
					Bid:    mathutils.Round(199+float64(i), 2),
					Pd:     mathutils.Round(0.2*float64(i), 2),
					Pd20:   mathutils.Round(0.2*float64(i*2), 2),
					Max10:  mathutils.Round(-0.2*float64(i), 2),
					Min10:  mathutils.Round(0.1*float64(i), 2),
				},
			},
			Avg: &TickAvg{
				PD:           mathutils.Round(float64(i), 2),
				PD20:         mathutils.Round(float64(i*2), 2),
				Max10:        mathutils.Round(-0.1*float64(i), 2),
				Min10:        mathutils.Round(0.1*float64(i), 2),
				SellDiff:     mathutils.Round(0.1*float64(i), 2),
				BuyDiff:      mathutils.Round(0.1*float64(i), 2),
				TickersCount: 2,
			},
		}
	}

	// Execute CalculateIndicators
	currentTick := history[historySize-1]
	currentTick.CalculateIndicators(history)

	// Validate results
	assert.Equal(t, 0.45, currentTick.AvgBuy10, "AvgBuy10 should match expected value")
	assert.Equal(t, 0.705, currentTick.Avg.SellDiff, "SellDiff should match expected value")
	assert.Equal(t, 0.705, currentTick.Avg.BuyDiff, "BuyDiff should match expected value")
	assert.Equal(t, 1.35, currentTick.Avg.PD, "PD should match expected value")
	assert.Equal(t, 2.7, currentTick.Avg.PD20, "PD20 should match expected value")
	assert.InDelta(t, -11961.11, currentTick.Avg.Max10, 0.01, "Max10 should match expected value")
	assert.InDelta(t, 14538.89, currentTick.Avg.Min10, 0.01, "Min10 should match expected value")
	assert.Equal(t, int16(2), currentTick.Avg.TickersCount, "TickersCount should match expected value")

	currentTick.Data["BTC"].Ask *= 10
	assert.Equal(t, 0.705, currentTick.Avg.BuyDiff, "Cover the case when diff more than 1% SellDiff")
	currentTick.Data["BTC"].Ask /= 100
	currentTick.CalculateIndicators(history)
	assert.Equal(t, -0.26, currentTick.Avg.BuyDiff, "Cover the case when diff more than 1% SellDiff")
}
