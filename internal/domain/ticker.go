package domain

import (
	"math"
	"time"
)

// TickerName represents a market symbol
type TickerName string

// Ticker represents a market symbol's snapshot at a given time
// Use short names to save space in the database
type Ticker struct {
	Symbol    TickerName `db:"s" json:"s" bson:"s"` // symbol
	Date      time.Time  `db:"date" json:"date" bson:"date"`
	Ask       float64    `db:"ask" json:"ask" bson:"ask"`
	Bid       float64    `db:"bid" json:"bid" bson:"bid"`
	Rsi20     float64    `db:"rsi_20" json:"rsi_20" bson:"rsi_20"`
	BuyPd     float64    `db:"tb_pd" json:"tb_pd" bson:"tb_pd"`                   // % diff: prev vs curr ask
	SellPd    float64    `db:"ts_pd" json:"ts_pd" bson:"ts_pd"`                   // % diff: prev vs curr bid
	TPdDiff   float64    `db:"t_pd_diff" json:"t_pd_diff" bson:"t_pd_diff"`       // SellPd - BuyPd
	Pd        float64    `db:"pd" json:"pd" bson:"pd"`                            // % change since last minute
	Pd20      float64    `db:"pd_20" json:"pd_20" bson:"pd_20"`                   // % change since last 20 minutes
	Max       float64    `db:"max" json:"max" bson:"max"`                         // max ask in the last minute
	Min       float64    `db:"min" json:"min" bson:"min"`                         // min ask in the last minute
	Max10     float64    `db:"max_10" json:"max_10" bson:"max_10"`                // max ask in the last 10 minutes
	Min10     float64    `db:"min_10" json:"min_10" bson:"min_10"`                // min ask in the last 10 minutes
	Max10Diff float64    `db:"max_10_diff" json:"max_10_diff" bson:"max_10_diff"` // Max10 - ask / Max10
}

// CalculateIndicators calculates the indicators for current moment
// history includes current moment data (e.g. history[len(history)-1] == t)
func (t *Ticker) CalculateIndicators(history []*Ticker, prevTick *Tick) {
	prevTicker := prevTick.Data[t.Symbol]
	if len(history) > 21 {
		// calculate pd and pd20
		ticker20MinutesAgo := history[len(history)-21]
		ticker1MinutesAgo := history[len(history)-2]
		t.Pd20 = math.Round((t.Bid-ticker20MinutesAgo.Bid)/ticker20MinutesAgo.Bid*100*100) / 100
		t.Pd = math.Round((t.Bid-ticker1MinutesAgo.Bid)/ticker1MinutesAgo.Bid*100*100) / 100

		rsi := CalculateRSI(history, 20, "bid")
		t.Rsi20 = math.Round(rsi*10) / 10
	}

	if len(history) > 10 {
		min10, max10 := math.MaxFloat64, -1*math.MaxFloat64
		for _, h := range history[len(history)-10:] {
			if h.Ask > max10 {
				max10 = h.Ask
			}
			if h.Ask < min10 {
				min10 = h.Ask
			}
		}
		t.Max10 = max10
		t.Min10 = min10
		t.Max10Diff = math.Round((t.Max10-t.Ask)/t.Max10*100*100) / 100

		t.BuyPd = math.Round((t.Ask-prevTicker.Ask)/prevTicker.Ask*100*10000) / 10000
		t.SellPd = math.Round((t.Bid-prevTicker.Bid)/prevTicker.Bid*100*10000) / 10000
		t.TPdDiff = math.Round(t.SellPd-t.BuyPd*10000) / 10000
	}
}

// CalculateRSI replicates the Python-style RSI logic for a given period.
// field should be either "ask" or "bid" to indicate which Ticker field to use.
func CalculateRSI(history []*Ticker, period int, field string) float64 {
	// Require at least 2 data points. If fewer, just return 0 or 50—your call.
	if len(history) < 2 {
		return 0
	}

	// We'll take the last `period` items in history
	if len(history) < period {
		// Not enough for the full period, so fallback or just use all
		period = len(history)
	}
	slice := history[len(history)-period:]

	var up float64
	var down float64

	// Accumulate up/down moves
	for i := 1; i < len(slice); i++ {
		var current, previous float64
		switch field {
		case "ask":
			current = slice[i].Ask
			previous = slice[i-1].Ask
		case "bid":
			current = slice[i].Bid
			previous = slice[i-1].Bid
		default:
			// fallback to Bid or handle error
			current = slice[i].Bid
			previous = slice[i-1].Bid
		}

		if current > previous {
			up += current - previous
		} else {
			down += previous - current
		}
	}

	// Handle edge cases
	if up == 0 && down == 0 {
		// Flat line => RSI is 50
		return 50
	}
	if up == 0 {
		// Pure downward movement
		return 0
	}
	if down == 0 {
		// Pure upward movement
		return 100
	}

	// Standard RSI formula
	return 100.0 - (100.0 / (1.0 + (up / down)))
}
