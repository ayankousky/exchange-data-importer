package domain

import (
	"context"
	"time"
)

// Tick represents a snapshot of market data for multiple tickers at a specific point in time
// It includes average metrics, liquidation counts,and a map of Ticker data keyed by a TickerName
// This item is stored in the database
type Tick struct {
	ID        string    `db:"_id" json:"_id" bson:"_id"`
	StartAt   time.Time `db:"start_at" json:"start_at" bson:"start_at"`       // handling start at
	FetchedAt time.Time `db:"fetched_at" json:"fetched_at" bson:"fetched_at"` // fetched from exchange at
	CreatedAt time.Time `db:"created_at" json:"created_at" bson:"created_at"` // ready to be stored at

	FetchDuration    int16 `db:"fetch_duration" json:"fetch_duration" bson:"fetch_duration"`
	HandlingDuration int16 `db:"handling_duration" json:"handling_duration" bson:"handling_duration"`

	TickAvgBuyOpen float64 `db:"tick_avg_buy_open" json:"tick_avg_buy_open" bson:"tick_avg_buy_open"`
	Tl1            int16   `db:"tl_1" json:"tl_1" bson:"tl_1"`       // 1s second total long liquidations
	Tl2            int16   `db:"tl_2" json:"tl_2" bson:"tl_2"`       // 2s second total long liquidations
	Tl5            int16   `db:"tl_5" json:"tl_5" bson:"tl_5"`       // 5s second total long liquidations
	Tsl1           int16   `db:"tsl_1" json:"tsl_1" bson:"tsl_1"`    // 1s second total short liquidations
	Tsl2           int16   `db:"tsl_2" json:"tsl_2" bson:"tsl_2"`    // 2s second total short liquidations
	Tsl10          int16   `db:"tsl_10" json:"tsl_10" bson:"tsl_10"` // 10s second total short liquidations
	Btsl           int16   `db:"btsl" json:"btsl" bson:"btsl"`       // 1s bitcoin total short liquidations
	Lmltc          int32   `db:"lmltc" json:"lmltc" bson:"lmltc"`    // last minute total long liquidations count

	Avg *TickSnapshotAvg `db:"avg" json:"avg" bson:"avg"`
	// store data as map to be able to query by ticker name or project the data
	Data map[TickerName]*Ticker `db:"data" json:"data" bson:"data"`
}

// TickSnapshotAvg represents the average of all tickers in a snapshot
type TickSnapshotAvg struct {
	PD           float64 `db:"pd" json:"pd" bson:"pd"`
	PD20         float64 `db:"pd_20" json:"pd_20" bson:"pd_20"`
	Max10        float64 `db:"max_10" json:"max_10" bson:"max_10"`
	Min10        float64 `db:"min_10" json:"min_10" bson:"min_10"`
	SellDiff     float64 `db:"sell_diff" json:"sell_diff" bson:"sell_diff"`
	BuyDiff      float64 `db:"buy_diff" json:"buy_diff" bson:"buy_diff"`
	TickersCount int16   `db:"tickers_count" json:"tickers_count" bson:"tickers_count"`
}

// TickSnapshotRepository represents the tick snapshot repository contract
type TickSnapshotRepository interface {
	Create(ctx context.Context, ts *Tick) error
}
