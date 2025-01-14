package domain

import "time"

// TickerName represents a market symbol
type TickerName string

// Ticker represents a market symbol's snapshot at a given time
// Use short names to save space in the database
type Ticker struct {
	MID       string    `db:"mid" json:"mid" bson:"mid"` // minute id
	Date      time.Time `db:"date" json:"date" bson:"date"`
	BuyOpen   float64   `db:"buy_open" json:"buy_open" bson:"buy_open"`
	SellOpen  float64   `db:"sell_open" json:"sell_open" bson:"sell_open"`
	Rsi20     float64   `db:"rsi_20" json:"rsi_20" bson:"rsi_20"`
	BuyPd     float64   `db:"tb_pd" json:"tb_pd" bson:"tb_pd"`                   // % diff: prev vs curr buy
	SellPd    float64   `db:"ts_pd" json:"ts_pd" bson:"ts_pd"`                   // % diff: prev vs curr sell
	TPdDiff   float64   `db:"t_pd_diff" json:"t_pd_diff" bson:"t_pd_diff"`       // SellPd - BuyPd
	Pd        float64   `db:"pd" json:"pd" bson:"pd"`                            // % change since last minute
	Pd20      float64   `db:"pd_20" json:"pd_20" bson:"pd_20"`                   // % change since last 20 minutes
	Max       float64   `db:"max" json:"max" bson:"max"`                         // max price in the last minute
	Min       float64   `db:"min" json:"min" bson:"min"`                         // min price in the last minute
	Max10     float64   `db:"max_10" json:"max_10" bson:"max_10"`                // max price in the last 10 minutes
	Min10     float64   `db:"min_10" json:"min_10" bson:"min_10"`                // min price in the last 10 minutes
	Max10Diff float64   `db:"max_10_diff" json:"max_10_diff" bson:"max_10_diff"` // Max10 - BuyOpen / Max10
}
