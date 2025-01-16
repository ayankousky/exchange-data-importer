package importer

import (
	"context"
	"fmt"
	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"github.com/ayankousky/exchange-data-importer/pkg/exchanges"
	"time"
)

// MaxTickHistory is the maximum number of tick snapshots to keep in memory
const MaxTickHistory = 200

// RepositoryFactory is a contract for creating repositories
// each exchange must have its own separate repository
type RepositoryFactory interface {
	GetTickRepository(name string) domain.TickSnapshotRepository
	GetLiquidationRepository(name string) domain.LiquidationRepository
}

// Importer is responsible for importing data from an exchange and storing it in the database
type Importer struct {
	Exchange              exchanges.Exchange
	TickRepository        domain.TickSnapshotRepository
	LiquidationRepository domain.LiquidationRepository

	tickerHistory map[domain.TickerName][]*domain.Ticker
	tickHistory   []*domain.Tick
}

// NewImporter creates a new Importer
func NewImporter(exchange exchanges.Exchange, repositoryFactory RepositoryFactory) *Importer {
	return &Importer{
		Exchange:              exchange,
		TickRepository:        repositoryFactory.GetTickRepository(exchange.GetName()),
		LiquidationRepository: repositoryFactory.GetLiquidationRepository(exchange.GetName()),

		tickerHistory: make(map[domain.TickerName][]*domain.Ticker),
		tickHistory:   make([]*domain.Tick, 0),
	}
}

// StartImport imports tick data from the exchange (temporary method)
func (i *Importer) StartImport() error {
	startAt := time.Now()

	tickers, err := i.Exchange.FetchTickers(context.Background())
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
	}
	tick.Data = data

	i.addTickHistory(tick)

	// Store the tick in the database
	tick.CreatedAt = time.Now()
	tick.HandlingDuration = time.Since(fetchedAt)
	err = i.TickRepository.Create(context.Background(), tick)

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

func (i *Importer) addTickerHistory(ticker *domain.Ticker) {
	if len(i.tickerHistory[ticker.Symbol]) >= MaxTickHistory {
		// Remove the oldest item (index 0)
		i.tickerHistory[ticker.Symbol] = i.tickerHistory[ticker.Symbol][1:]
	}

	i.tickerHistory[ticker.Symbol] = append(i.tickerHistory[ticker.Symbol], ticker)
}

func (i *Importer) initHistory() {
	history, _ := i.TickRepository.GetHistorySince(context.Background(), time.Now().Add(-(MaxTickHistory+10)*time.Second))
	for _, tick := range history {
		i.addTickHistory(tick)
		for _, ticker := range tick.Data {
			i.addTickerHistory(ticker)
		}
	}
}
