package importer

import (
	"context"
	"fmt"
	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"github.com/ayankousky/exchange-data-importer/pkg/exchanges"
	"time"
)

// MaxTickHistory is the maximum number of tick snapshots to keep in memory
const MaxTickHistory = 25

// RepositoryFactory is a contract for creating repositories
// each exchange must have its own separate repository
type RepositoryFactory interface {
	GetTickRepository(name string) domain.TickRepository
	GetLiquidationRepository(name string) domain.LiquidationRepository
}

// Importer is responsible for importing data from an exchange and storing it in the database
type Importer struct {
	exchange              exchanges.Exchange
	tickRepository        domain.TickRepository
	liquidationRepository domain.LiquidationRepository

	tickerHistory map[domain.TickerName][]*domain.Ticker
	tickHistory   []*domain.Tick
}

// NewImporter creates a new Importer
func NewImporter(exchange exchanges.Exchange, repositoryFactory RepositoryFactory) *Importer {
	return &Importer{
		exchange:              exchange,
		tickRepository:        repositoryFactory.GetTickRepository(exchange.GetName()),
		liquidationRepository: repositoryFactory.GetLiquidationRepository(exchange.GetName()),

		tickerHistory: make(map[domain.TickerName][]*domain.Ticker),
		tickHistory:   make([]*domain.Tick, 0),
	}
}

// StartImport imports tick data from the exchange (temporary method)
func (i *Importer) StartImport() error {
	startAt := time.Now()

	tickers, err := i.exchange.FetchTickers(context.Background())
	if err != nil {
		return err
	}
	fetchedAt := time.Now()

	// Generate ISO timestamp ID
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	tick := &domain.Tick{
		ID:            timestamp,
		StartAt:       startAt,
		FetchedAt:     fetchedAt,
		FetchDuration: fetchedAt.Sub(startAt).Milliseconds(),
		Avg:           &domain.TickAvg{},
	}
	data := make(map[domain.TickerName]*domain.Ticker, 0)
	for _, ticker := range tickers {
		tickerData := &domain.Ticker{
			Symbol: domain.TickerName(ticker.Symbol),
			Ask:    ticker.AskPrice,
			Bid:    ticker.BidPrice,
			Date:   startAt,
		}
		data[tickerData.Symbol] = tickerData
		i.addTickerHistory(tickerData)
		tickerData.CalculateIndicators(i.getTickerHistory(tickerData.Symbol), i.getLastTick())
	}
	tick.Data = data

	i.addTickHistory(tick)
	tick.CalculateIndicators(i.getTickHistory())

	// Store the tick in the database
	tick.CreatedAt = time.Now()
	tick.HandlingDuration = time.Since(fetchedAt).Milliseconds()
	err = i.tickRepository.Create(context.Background(), tick)

	if err != nil {
		return err
	}

	return nil
}

// StartImportEverySecond starts the import process every second
// temporary function to simulate a real-time import process
func (i *Importer) StartImportEverySecond() {
	i.initHistory()
	for {
		// Calculate the duration until the next second
		now := time.Now()
		next := now.Truncate(time.Second).Add(time.Second)
		time.Sleep(time.Until(next))

		// Run StartImport
		err := i.StartImport()
		if err != nil {
			fmt.Printf("Error in StartImport: %v\n", err)
		}
	}
}

func (i *Importer) addTickHistory(tick *domain.Tick) {
	if len(i.tickHistory) >= MaxTickHistory {
		// Remove the oldest item (index 0)
		i.tickHistory = i.tickHistory[1:]
	}

	i.tickHistory = append(i.tickHistory, tick)
}

// history is a map of TickerName to a list of Ticker data for that symbol
// 1 item = 1 minute of data (no need to store for each second)
func (i *Importer) addTickerHistory(ticker *domain.Ticker) {
	if len(i.tickerHistory[ticker.Symbol]) >= MaxTickHistory {
		// Remove the oldest item (index 0)
		i.tickerHistory[ticker.Symbol] = i.tickerHistory[ticker.Symbol][1:]
	}

	// Retrieve the last ticker data for this symbol, if it exists
	var lastTickerData *domain.Ticker
	if len(i.tickerHistory[ticker.Symbol]) > 0 {
		lastTickerData = i.tickerHistory[ticker.Symbol][len(i.tickerHistory[ticker.Symbol])-1]
	}

	// If there is no data for this minute, create a new history item
	if lastTickerData == nil || !lastTickerData.Date.Truncate(time.Minute).Equal(ticker.Date.Truncate(time.Minute)) {
		ticker.Max = ticker.Ask
		ticker.Min = ticker.Ask
		i.tickerHistory[ticker.Symbol] = append(i.tickerHistory[ticker.Symbol], ticker)
	} else {
		// Update the existing lastTickerData directly
		if ticker.Ask > lastTickerData.Max {
			lastTickerData.Max = ticker.Ask
		}
		if ticker.Ask < lastTickerData.Min {
			lastTickerData.Min = ticker.Ask
		}
		lastTickerData.Ask = ticker.Ask
		lastTickerData.Bid = ticker.Bid
		lastTickerData.Date = ticker.Date

		ticker.Max = lastTickerData.Max
		ticker.Min = lastTickerData.Min
	}
}

func (i *Importer) initHistory() {
	history, _ := i.tickRepository.GetHistorySince(context.Background(), time.Now().Add(-MaxTickHistory*time.Minute))
	for _, tick := range history {
		i.addTickHistory(tick)
		for _, ticker := range tick.Data {
			i.addTickerHistory(ticker)
		}
	}
}

func (i *Importer) getTickHistory() []*domain.Tick {
	return i.tickHistory
}
func (i *Importer) getTickerHistory(tickerName domain.TickerName) []*domain.Ticker {
	return i.tickerHistory[tickerName]
}
func (i *Importer) getLastTick() *domain.Tick {
	if len(i.tickHistory) > 0 {
		return i.tickHistory[len(i.tickHistory)-1]
	}
	return nil
}
