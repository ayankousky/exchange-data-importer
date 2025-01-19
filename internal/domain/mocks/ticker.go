package mocks

import (
	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"time"
)

// ValidETHUSDT is a valid ticker for ETHUSDT
var ValidETHUSDT = domain.Ticker{
	Symbol:    "ETHUSDT",
	Date:      time.Date(2025, 1, 19, 14, 27, 26, 3000000, time.UTC),
	Ask:       3345.16,
	Bid:       3345.15,
	Rsi20:     59.2,
	BuyPd:     -0.03,
	SellPd:    -0.03,
	TPdDiff:   0.0,
	Pd:        -0.07,
	Pd20:      1.25,
	Max:       3354.28,
	Min:       3344.83,
	Max10:     3381.8,
	Min10:     3337.2,
	Max10Diff: -1.08,
	Min10Diff: 0.24,
}

// ValidBTCUSDT is a valid ticker for BTCUSDT
var ValidBTCUSDT = domain.Ticker{
	Symbol:    "BTCUSDT",
	Date:      time.Date(2025, 1, 19, 14, 27, 26, 3000000, time.UTC),
	Ask:       104388.7,
	Bid:       104388.6,
	Rsi20:     47.4,
	BuyPd:     0.0,
	SellPd:    0.0,
	TPdDiff:   0.0,
	Pd:        0.01,
	Pd20:      0.02,
	Max:       104393.0,
	Min:       104368.6,
	Max10:     104403.1,
	Min10:     104350.0,
	Max10Diff: -0.01,
	Min10Diff: 0.04,
}

// Invalid1BTCUSDT is an invalid ticker for BTCUSDT with wrong Ask value
var Invalid1BTCUSDT = domain.Ticker{
	Symbol:    "BTCUSDT",
	Date:      time.Date(2025, 1, 19, 14, 27, 26, 3000000, time.UTC),
	Ask:       0,
	Bid:       104388.6,
	Rsi20:     47.4,
	BuyPd:     0.0,
	SellPd:    0.0,
	TPdDiff:   0.0,
	Pd:        0.01,
	Pd20:      0.02,
	Max:       104393.0,
	Min:       104368.6,
	Max10:     104403.1,
	Min10:     104350.0,
	Max10Diff: -0.01,
	Min10Diff: 0.04,
}

// Invalid2BTCUSDT is an invalid ticker for BTCUSDT with wrong Bid value
var Invalid2BTCUSDT = domain.Ticker{
	Symbol:    "BTCUSDT",
	Date:      time.Date(2025, 1, 19, 14, 27, 26, 3000000, time.UTC),
	Ask:       104388.7,
	Bid:       0,
	Rsi20:     47.4,
	BuyPd:     0.0,
	SellPd:    0.0,
	TPdDiff:   0.0,
	Pd:        0.01,
	Pd20:      0.02,
	Max:       104393.0,
	Min:       104368.6,
	Max10:     104403.1,
	Min10:     104350.0,
	Max10Diff: -0.01,
	Min10Diff: 0.04,
}
