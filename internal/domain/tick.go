package domain

import (
	"context"
	"fmt"
	"time"

	"github.com/ayankousky/exchange-data-importer/pkg/utils"
	"github.com/ayankousky/exchange-data-importer/pkg/utils/mathutils"
)

//go:generate moq --out mocks/tick_repository.go --pkg mocks --with-resets --skip-ensure . TickRepository

const (
	// MaxTickHistory is the maximum number of tick snapshots to keep in memory
	MaxTickHistory = 25
)

// Tick represents a snapshot of market data for multiple tickers at a specific point in time
// It includes average metrics, liquidation counts,and a map of Ticker data keyed by a TickerName
// This item is stored in the database
type Tick struct {
	StartAt   time.Time `db:"start_at" json:"start_at" bson:"start_at"`       // handling start at
	FetchedAt time.Time `db:"fetched_at" json:"fetched_at" bson:"fetched_at"` // fetched from exchange at
	CreatedAt time.Time `db:"created_at" json:"created_at" bson:"created_at"` // ready to be stored at

	FetchDuration    int64 `db:"fetch_duration" json:"fetch_duration" bson:"fetch_duration"`
	HandlingDuration int64 `db:"handling_duration" json:"handling_duration" bson:"handling_duration"`

	AvgBuy10 float64 `db:"tick_avg_buy_open" json:"tick_avg_buy_open" bson:"tick_avg_buy_open"`
	LL1      int64   `db:"ll_1" json:"ll_1" bson:"ll_1"`    // 1s second total long liquidations
	LL2      int64   `db:"ll_2" json:"ll_2" bson:"ll_2"`    // 2s second total long liquidations
	LL5      int64   `db:"ll_5" json:"ll_5" bson:"ll_5"`    // 5s second total long liquidations
	LL60     int64   `db:"ll_60" json:"ll_60" bson:"ll_60"` // 60s second total long liquidations
	SL1      int64   `db:"sl_1" json:"sl_1" bson:"sl_1"`    // 1s second total short liquidations
	SL2      int64   `db:"sl_2" json:"sl_2" bson:"sl_2"`    // 2s second total short liquidations
	SL10     int64   `db:"sl_10" json:"sl_10" bson:"sl_10"` // 10s second total short liquidations

	Avg TickAvg `db:"avg" json:"avg" bson:"avg"`
	// store data as map to be able to query by ticker name or project the data
	Data map[TickerName]*Ticker `db:"data" json:"data" bson:"data"`
}

// TickAvg represents the average of all tickers in a snapshot
type TickAvg struct {
	Change1m     float64 `db:"pd" json:"pd" bson:"pd"`
	Change20m    float64 `db:"pd_20" json:"pd_20" bson:"pd_20"`
	Max10        float64 `db:"max_10" json:"max_10" bson:"max_10"`
	Min10        float64 `db:"min_10" json:"min_10" bson:"min_10"`
	AskChange    float64 `db:"a_pd" json:"a_pd" bson:"a_pd"`
	BidChange    float64 `db:"s_pd" json:"s_pd" bson:"s_pd"`
	TickersCount int16   `db:"tickers_count" json:"tickers_count" bson:"tickers_count"`
}

// TickRepository represents the tick snapshot repository contract
type TickRepository interface {
	Create(ctx context.Context, ts Tick) error
	GetHistorySince(ctx context.Context, since time.Time) ([]Tick, error)
}

// CalculateIndicators calculates the indicators for the current tick based on the history data
func (t *Tick) CalculateIndicators(history *utils.RingBuffer[*Tick]) {
	if history.Len() <= 1 {
		return
	}
	prevTick := history.At(history.Len() - 2)

	// If we have at least 10 ticks, compute an average Buy price for the last 10
	if history.Len() >= 10 {
		var sumTickAvgBuyOpen float64
		for i := history.Len() - 10; i < history.Len(); i++ {
			sumTickAvgBuyOpen += history.At(i).Avg.AskChange
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

		sumPd += tickerCurrData.Change1m
		sumPd20 += tickerCurrData.Change20m

		sumMax10 += mathutils.PercDiff(tickerCurrData.Ask, tickerCurrData.Max10, -1)
		sumMin10 += mathutils.PercDiff(tickerCurrData.Ask, tickerCurrData.Min10, -1)
	}
	if count > 0 {
		t.Avg.BidChange = mathutils.Round(sumSellDiff/count, 4)
		t.Avg.AskChange = mathutils.Round(sumBuyDiff/count, 4)
		t.Avg.Change1m = mathutils.Round(sumPd/count, 2)
		t.Avg.Change20m = mathutils.Round(sumPd20/count, 2)
		t.Avg.Max10 = mathutils.Round(sumMax10/count, 2)
		t.Avg.Min10 = mathutils.Round(sumMin10/count, 2)
		t.Avg.TickersCount = int16(count)
	}
}

// SetTicker sets a ticker in the tick snapshot
func (t *Tick) SetTicker(ticker *Ticker) {
	t.Data[ticker.Symbol] = ticker
}

// Validate performs validation of the Tick
func (t *Tick) Validate() error {
	if t.StartAt.IsZero() {
		return ValidationError{
			Field: "StartAt",
			Err:   fmt.Errorf("start time cannot be zero"),
		}
	}

	if t.FetchedAt.IsZero() {
		return ValidationError{
			Field: "FetchedAt",
			Err:   fmt.Errorf("fetched time cannot be zero"),
		}
	}

	if t.CreatedAt.IsZero() {
		return ValidationError{
			Field: "CreatedAt",
			Err:   fmt.Errorf("created time cannot be zero"),
		}
	}

	return nil
}
