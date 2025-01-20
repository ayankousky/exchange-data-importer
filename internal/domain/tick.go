package domain

import (
	"context"
	"github.com/ayankousky/exchange-data-importer/pkg/utils"
	"github.com/ayankousky/exchange-data-importer/pkg/utils/mathutils"
	"time"
)

// Tick represents a snapshot of market data for multiple tickers at a specific point in time
// It includes average metrics, liquidation counts,and a map of Ticker data keyed by a TickerName
// This item is stored in the database
type Tick struct {
	StartAt   time.Time `db:"start_at" json:"start_at" bson:"start_at"`       // handling start at
	FetchedAt time.Time `db:"fetched_at" json:"fetched_at" bson:"fetched_at"` // fetched from exchange at
	CreatedAt time.Time `db:"created_at" json:"created_at" bson:"created_at"` // ready to be stored at

	FetchDuration    int64         `db:"fetch_duration" json:"fetch_duration" bson:"fetch_duration"`
	HandlingDuration time.Duration `db:"handling_duration" json:"handling_duration" bson:"handling_duration"`

	AvgBuy10 float64 `db:"tick_avg_buy_open" json:"tick_avg_buy_open" bson:"tick_avg_buy_open"`
	Tl1      int16   `db:"tl_1" json:"tl_1" bson:"tl_1"`       // 1s second total long liquidations
	Tl2      int16   `db:"tl_2" json:"tl_2" bson:"tl_2"`       // 2s second total long liquidations
	Tl5      int16   `db:"tl_5" json:"tl_5" bson:"tl_5"`       // 5s second total long liquidations
	Tsl1     int16   `db:"tsl_1" json:"tsl_1" bson:"tsl_1"`    // 1s second total short liquidations
	Tsl2     int16   `db:"tsl_2" json:"tsl_2" bson:"tsl_2"`    // 2s second total short liquidations
	Tsl10    int16   `db:"tsl_10" json:"tsl_10" bson:"tsl_10"` // 10s second total short liquidations
	Btsl     int16   `db:"btsl" json:"btsl" bson:"btsl"`       // 1s bitcoin total short liquidations
	Lmltc    int32   `db:"lmltc" json:"lmltc" bson:"lmltc"`    // last minute total long liquidations count

	Avg TickAvg `db:"avg" json:"avg" bson:"avg"`
	// store data as map to be able to query by ticker name or project the data
	Data map[TickerName]*Ticker `db:"data" json:"data" bson:"data"`
}

// TickAvg represents the average of all tickers in a snapshot
type TickAvg struct {
	PD           float64 `db:"pd" json:"pd" bson:"pd"`
	PD20         float64 `db:"pd_20" json:"pd_20" bson:"pd_20"`
	Max10        float64 `db:"max_10" json:"max_10" bson:"max_10"`
	Min10        float64 `db:"min_10" json:"min_10" bson:"min_10"`
	SellDiff     float64 `db:"sell_diff" json:"sell_diff" bson:"sell_diff"`
	BuyDiff      float64 `db:"buy_diff" json:"buy_diff" bson:"buy_diff"`
	TickersCount int16   `db:"tickers_count" json:"tickers_count" bson:"tickers_count"`
}

// TickRepository represents the tick snapshot repository contract
type TickRepository interface {
	Create(ctx context.Context, ts Tick) error
	GetHistorySince(ctx context.Context, since time.Time) ([]Tick, error)
}

// CalculateIndicators calculates the indicators for the current tick
func (t *Tick) CalculateIndicators(history *utils.RingBuffer[*Tick]) {
	if history.Len() <= 1 {
		return
	}
	prevTick := history.At(history.Len() - 2)

	// If we have at least 10 ticks, compute an average Buy price for the last 10
	if history.Len() >= 10 {
		var sumTickAvgBuyOpen float64
		for i := history.Len() - 10; i < history.Len(); i++ {
			sumTickAvgBuyOpen += history.At(i).Avg.BuyDiff
		}
		t.AvgBuy10 = mathutils.Round(sumTickAvgBuyOpen/10, 6)
	}

	// Calculate the averages for the current tick
	var sumSellDiff, sumBuyDiff, sumPd, sumPd20, sumMax10, sumMin10, count float64
	for _, tickerCurrData := range t.Data {
		tickerPrevData, ok := prevTick.Data[tickerCurrData.Symbol]
		if !ok {
			continue
		}
		count++

		buyDiff := mathutils.Clamp(mathutils.PercDiff(tickerCurrData.Ask, tickerPrevData.Ask, 2), -1, 1)
		sellDiff := mathutils.Clamp(mathutils.PercDiff(tickerCurrData.Bid, tickerPrevData.Bid, 2), -1, 1)
		sumBuyDiff += buyDiff
		sumSellDiff += sellDiff

		sumPd += tickerCurrData.Pd
		sumPd20 += tickerCurrData.Pd20

		sumMax10 += mathutils.PercDiff(tickerCurrData.Ask, tickerCurrData.Max10, -1)
		sumMin10 += mathutils.PercDiff(tickerCurrData.Ask, tickerCurrData.Min10, -1)
	}
	if count > 0 {
		t.Avg.SellDiff = mathutils.Round(sumSellDiff/count, 4)
		t.Avg.BuyDiff = mathutils.Round(sumBuyDiff/count, 4)
		t.Avg.PD = mathutils.Round(sumPd/count, 2)
		t.Avg.PD20 = mathutils.Round(sumPd20/count, 2)
		t.Avg.Max10 = mathutils.Round(sumMax10/count, 2)
		t.Avg.Min10 = mathutils.Round(sumMin10/count, 2)
		t.Avg.TickersCount = int16(count)
	}
}

// SetTicker sets a ticker in the tick snapshot
func (t *Tick) SetTicker(ticker *Ticker) {
	t.Data[ticker.Symbol] = ticker
}
