package domain

import (
	"math"
	"time"

	"github.com/ayankousky/exchange-data-importer/pkg/utils"
	"github.com/ayankousky/exchange-data-importer/pkg/utils/mathutils"
	"github.com/ayankousky/exchange-data-importer/pkg/utils/tradeutils"
)

// TickerName represents a market symbol
type TickerName string

// Ticker represents a market symbol's snapshot at a given time
// Use short names to save space in the database but readable enough for human
type Ticker struct {
	Symbol    TickerName `db:"s" json:"s" bson:"s"`    // symbol
	EventAt   time.Time  `db:"et" json:"et" bson:"et"` // date when event happened on the exchange
	CreatedAt time.Time  `db:"ct" json:"ct" bson:"ct"` // date when data was created in the system
	Ask       float64    `db:"ask" json:"ask" bson:"ask"`
	Bid       float64    `db:"bid" json:"bid" bson:"bid"`
	RSI20     float64    `db:"rsi_20" json:"rsi_20" bson:"rsi_20"`
	AskChange float64    `db:"a_pd" json:"a_pd" bson:"a_pd"` // % diff: prev vs curr ask
	BidChange float64    `db:"b_pd" json:"b_pd" bson:"b_pd"` // % diff: prev vs curr bid

	// % change since last minute, last 20 minutes
	Change1m  float64 `db:"pd" json:"pd" bson:"pd"`
	Change20m float64 `db:"pd_20" json:"pd_20" bson:"pd_20"`

	// Max / Min => 1-minute rolling extremes
	// Max10 / Min10 => 10-minute rolling extremes
	Max       float64 `db:"max"       json:"max"       bson:"max"`
	Min       float64 `db:"min"       json:"min"       bson:"min"`
	Max10     float64 `db:"max_10"    json:"max_10"    bson:"max_10"`
	Min10     float64 `db:"min_10"    json:"min_10"    bson:"min_10"`
	Max10Diff float64 `db:"max_10_diff" json:"max_10_diff" bson:"max_10_diff"` // (Ask - Max10) / Max10 * 100
	Min10Diff float64 `db:"min_10_diff" json:"min_10_diff" bson:"min_10_diff"` // (Ask - Min10) / Min10 * 100
}

// CalculateIndicators calculates the indicators for current moment based on the history data
// each history item is a minute of data
func (t *Ticker) CalculateIndicators(history *utils.RingBuffer[*Ticker], lastTick *Tick) {
	// Safety checks
	if t == nil || lastTick == nil || lastTick.Data == nil {
		return
	}
	prevTicker, ok := lastTick.Data[t.Symbol]
	if !ok {
		return
	}
	historyLength := history.Len()

	if historyLength < 2 {
		return
	}

	if historyLength > 10 {
		t.Change1m = mathutils.PercDiff(t.Bid, history.At(historyLength-2).Bid, 2)

		// Evaluate the last 10 Tickers for max/min
		min10, max10 := math.MaxFloat64, -1*math.MaxFloat64
		for i := historyLength - 10; i < historyLength; i++ {
			h := history.At(i)
			if h.Ask > max10 {
				max10 = h.Ask
			}
			if h.Ask < min10 {
				min10 = h.Ask
			}
		}
		t.Max10 = max10
		t.Min10 = min10
		t.Max10Diff = mathutils.PercDiff(t.Ask, t.Max10, 2)
		t.Min10Diff = mathutils.PercDiff(t.Ask, t.Min10, 2)

		t.AskChange = mathutils.PercDiff(t.Ask, prevTicker.Ask, 2)
		t.BidChange = mathutils.PercDiff(t.Bid, prevTicker.Bid, 2)
	}
	if historyLength > 21 {
		t.Change20m = mathutils.PercDiff(t.Bid, history.At(historyLength-21).Bid, 2)

		// calculate RSI
		bidHistory := make([]float64, 20)
		for i := 0; i < 20; i++ {
			bidHistory[i] = history.At(historyLength - 20 + i).Bid
		}
		t.RSI20 = mathutils.Round(tradeutils.CalculateRSI(bidHistory, 20), 1)
	}
}

// IsValid checks if the ticker has valid data
func (t *Ticker) IsValid() bool {
	if t == nil {
		return false
	}
	if t.Ask == 0 || t.Bid == 0 {
		return false
	}
	return true
}
