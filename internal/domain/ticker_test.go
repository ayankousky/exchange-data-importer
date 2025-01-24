package domain

import (
	"github.com/ayankousky/exchange-data-importer/pkg/utils"
	"github.com/ayankousky/exchange-data-importer/pkg/utils/mathutils"
	"github.com/stretchr/testify/assert"
	"testing"
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
