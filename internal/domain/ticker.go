package domain

import (
	"github.com/ayankousky/exchange-data-importer/pkg/utils/mathutils"
	"github.com/ayankousky/exchange-data-importer/pkg/utils/tradeutils"
	"math"
	"time"
)

// TickerName represents a market symbol
type TickerName string

// Ticker represents a market symbol's snapshot at a given time
// Use short names to save space in the database
type Ticker struct {
	Symbol  TickerName `db:"s" json:"s" bson:"s"` // symbol
	Date    time.Time  `db:"date" json:"date" bson:"date"`
	Ask     float64    `db:"ask" json:"ask" bson:"ask"`
	Bid     float64    `db:"bid" json:"bid" bson:"bid"`
	Rsi20   float64    `db:"rsi_20" json:"rsi_20" bson:"rsi_20"`
	BuyPd   float64    `db:"tb_pd" json:"tb_pd" bson:"tb_pd"`             // % diff: prev vs curr ask
	SellPd  float64    `db:"ts_pd" json:"ts_pd" bson:"ts_pd"`             // % diff: prev vs curr bid
	TPdDiff float64    `db:"t_pd_diff" json:"t_pd_diff" bson:"t_pd_diff"` // SellPd - BuyPd
	Pd      float64    `db:"pd" json:"pd" bson:"pd"`                      // % change since last minute
	Pd20    float64    `db:"pd_20" json:"pd_20" bson:"pd_20"`             // % change since last 20 minutes

	// Max / Min => 1-minute rolling extremes
	// Max10 / Min10 => 10-minute rolling extremes
	Max       float64 `db:"max"       json:"max"       bson:"max"`
	Min       float64 `db:"min"       json:"min"       bson:"min"`
	Max10     float64 `db:"max_10"    json:"max_10"    bson:"max_10"`
	Min10     float64 `db:"min_10"    json:"min_10"    bson:"min_10"`
	Max10Diff float64 `db:"max_10_diff" json:"max_10_diff" bson:"max_10_diff"` // (Ask - Max10) / Max10 * 100
	Min10Diff float64 `db:"min_10_diff" json:"min_10_diff" bson:"min_10_diff"` // (Ask - Min10) / Min10 * 100
}

// CalculateIndicators calculates the indicators for current moment
// each history item is a minute of data
func (t *Ticker) CalculateIndicators(history []*Ticker, prevTick *Tick) {
	// Safety checks
	if t == nil || prevTick == nil || prevTick.Data == nil {
		return
	}
	prevTicker, ok := prevTick.Data[t.Symbol]
	if !ok {
		return
	}
	historyLength := len(history)
	if historyLength < 2 {
		return
	}

	if historyLength > 10 {
		t.Pd = mathutils.PercDiff(t.Bid, history[historyLength-2].Bid, 2)

		// Evaluate the last 10 Tickers for max/min
		min10, max10 := math.MaxFloat64, -1*math.MaxFloat64
		recent10 := history[historyLength-10:]
		for _, h := range recent10 {
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

		t.BuyPd = mathutils.PercDiff(t.Ask, prevTicker.Ask, 2)
		t.SellPd = mathutils.PercDiff(t.Bid, prevTicker.Bid, 2)
		t.TPdDiff = mathutils.Round(t.SellPd-t.BuyPd, 4)
	}
	if historyLength > 21 {
		t.Pd20 = mathutils.PercDiff(t.Bid, history[historyLength-21].Bid, 2)

		// calculate RSI
		bidHistory := make([]float64, 20)
		for i := 0; i < 20; i++ {
			bidHistory[i] = history[historyLength-20+i].Bid
		}
		t.Rsi20 = mathutils.Round(tradeutils.CalculateRSI(bidHistory, 20), 1)
	}
}
