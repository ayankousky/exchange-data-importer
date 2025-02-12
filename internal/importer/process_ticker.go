package importer

import (
	"fmt"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges"
)

func (i *Importer) buildTicker(currTick domain.Tick, lastTick *domain.Tick, eTicker exchanges.Ticker) (*domain.Ticker, error) {
	ticker := &domain.Ticker{
		Symbol:    domain.TickerName(eTicker.Symbol),
		Ask:       eTicker.AskPrice,
		Bid:       eTicker.BidPrice,
		EventAt:   eTicker.EventAt,
		CreatedAt: currTick.StartAt,
	}

	if err := ticker.Validate(); err != nil {
		return nil, fmt.Errorf("invalid ticker data: %v", err)
	}

	i.addTickerHistory(ticker)
	ticker.CalculateIndicators(i.getTickerHistory(ticker.Symbol), lastTick)
	return ticker, nil
}
