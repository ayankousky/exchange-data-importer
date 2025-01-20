package mock

import (
	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"github.com/ayankousky/exchange-data-importer/pkg/utils/mathutils"
	"time"
)

// GenerateTick creates a mock tick where the values are multiplied by the given i%
func GenerateTick(i int) domain.Tick {
	multiplier := 1.0 + float64(i)/200
	defaultDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	return domain.Tick{
		StartAt:          defaultDate.Add(time.Duration(i) * time.Second),
		FetchedAt:        defaultDate.Add(time.Duration(i)*time.Second + 100*time.Millisecond),
		CreatedAt:        defaultDate.Add(time.Duration(i)*time.Second + 200*time.Millisecond),
		FetchDuration:    5000,
		HandlingDuration: 2000,
		AvgBuy10:         3345.16,
		Tl1:              10,
		Tl2:              20,
		Tl5:              50,
		Tsl1:             15,
		Tsl2:             30,
		Tsl10:            100,
		Btsl:             5,
		Lmltc:            150,
		Avg: domain.TickAvg{
			PD:           mathutils.Round(-0.07*multiplier, 6),
			PD20:         mathutils.Round(1.25*multiplier, 6),
			Max10:        mathutils.Round(3381.8*multiplier, 6),
			Min10:        mathutils.Round(3337.2*multiplier, 6),
			SellDiff:     mathutils.Round(-0.01*multiplier, 6),
			BuyDiff:      mathutils.Round(0.04*multiplier, 6),
			TickersCount: 2,
		},
		Data: map[domain.TickerName]*domain.Ticker{
			"ETHUSDT": {
				Symbol:    "ETHUSDT",
				Date:      defaultDate.Add(time.Duration(i) * time.Second),
				Ask:       mathutils.Round(3345.16*multiplier, 6),
				Bid:       mathutils.Round(3345.15*multiplier, 6),
				Rsi20:     mathutils.Round(59.2*multiplier, 6),
				BuyPd:     mathutils.Round(-0.03*multiplier, 6),
				SellPd:    mathutils.Round(-0.03*multiplier, 6),
				Pd:        mathutils.Round(-0.07*multiplier, 6),
				Pd20:      mathutils.Round(1.25*multiplier, 6),
				Max:       mathutils.Round(3354.28*multiplier, 6),
				Min:       mathutils.Round(3344.83*multiplier, 6),
				Max10:     mathutils.Round(3381.8*multiplier, 6),
				Min10:     mathutils.Round(3337.2*multiplier, 6),
				Max10Diff: mathutils.Round(-1.08*multiplier, 6),
				Min10Diff: mathutils.Round(0.24*multiplier, 6),
			},
			"BTCUSDT": {
				Symbol:    "BTCUSDT",
				Date:      defaultDate.Add(time.Duration(i) * time.Second),
				Ask:       mathutils.Round(104388.7*multiplier, 6),
				Bid:       mathutils.Round(104388.6*multiplier, 6),
				Rsi20:     mathutils.Round(47.4*multiplier, 6),
				BuyPd:     mathutils.Round(0.01*multiplier, 6),
				SellPd:    mathutils.Round(0.01*multiplier, 6),
				Pd:        mathutils.Round(0.01*multiplier, 6),
				Pd20:      mathutils.Round(0.02*multiplier, 6),
				Max:       mathutils.Round(104393.0*multiplier, 6),
				Min:       mathutils.Round(104368.6*multiplier, 6),
				Max10:     mathutils.Round(104403.1*multiplier, 6),
				Min10:     mathutils.Round(104350.0*multiplier, 6),
				Max10Diff: mathutils.Round(-0.01*multiplier, 6),
				Min10Diff: mathutils.Round(0.04*multiplier, 6),
			},
		},
	}
}

// GenerateTicks creates a slice of mock ticks
func GenerateTicks(i int) []domain.Tick {
	ticks := make([]domain.Tick, i)
	for j := 0; j < i; j++ {
		ticks[j] = GenerateTick(j)
	}
	return ticks
}
