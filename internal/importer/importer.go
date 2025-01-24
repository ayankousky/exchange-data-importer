package importer

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges"
	"github.com/ayankousky/exchange-data-importer/pkg/utils"
)

// RepositoryFactory is a contract for creating repositories
type RepositoryFactory interface {
	GetTickRepository(name string) domain.TickRepository
	GetLiquidationRepository(name string) domain.LiquidationRepository
}

// Importer is responsible for importing data from an exchange and storing it in the database
type Importer struct {
	exchange              exchanges.Exchange
	tickRepository        domain.TickRepository
	liquidationRepository domain.LiquidationRepository

	tickHistory   *utils.RingBuffer[*domain.Tick]
	tickerHistory map[domain.TickerName]*utils.RingBuffer[*domain.Ticker]
}

// NewImporter creates a new Importer
func NewImporter(exchange exchanges.Exchange, repositoryFactory RepositoryFactory) *Importer {
	return &Importer{
		exchange:              exchange,
		tickRepository:        repositoryFactory.GetTickRepository(exchange.GetName()),
		liquidationRepository: repositoryFactory.GetLiquidationRepository(exchange.GetName()),

		tickerHistory: make(map[domain.TickerName]*utils.RingBuffer[*domain.Ticker]),
		tickHistory:   utils.NewRingBuffer[*domain.Tick](domain.MaxTickHistory),
	}
}

// StartImportLoop starts a loop that imports data from the exchange periodically.
func (i *Importer) StartImportLoop(ctx context.Context, interval time.Duration) error {
	// Initialize the history data for calculating tick indicators
	if err := i.initHistory(ctx); err != nil {
		return fmt.Errorf("failed to init history: %w", err)
	}

	// Import should be started exactly at the beginning of the next second
	now := time.Now()
	nextSecond := now.Truncate(time.Second).Add(time.Second)
	time.Sleep(time.Until(nextSecond))

	// Start the import loop with the specified interval
	timeTicker := time.NewTicker(interval)
	defer timeTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Println("Context canceled, stopping import loop...")
			return ctx.Err()
		case <-timeTicker.C:
			// Attempt to import a single "tick" of data
			if err := i.importTick(ctx); err != nil {
				log.Printf("Error importing tick (continuing loop): %v", err)
			}
		}
	}
}

// initHistory loads old data from repositories and populates ring buffers
func (i *Importer) initHistory(ctx context.Context) error {
	history, err := i.tickRepository.GetHistorySince(ctx, time.Now().Add(-domain.MaxTickHistory*time.Minute))
	if err != nil {
		return fmt.Errorf("GetHistorySince failed: %w", err)
	}

	for _, tick := range history {
		i.addTickHistory(&tick)
		for _, ticker := range tick.Data {
			i.addTickerHistory(ticker)
		}
	}

	return nil
}

func (i *Importer) importTick(ctx context.Context) error {
	startAt := time.Now()

	// Fetch tickers from the exchange
	fetchedTickers, err := i.fetchTickers(ctx)
	if err != nil {
		return fmt.Errorf("fetchTickers failed: %w", err)
	}
	fetchedAt := time.Now()

	// Create a new tick
	newTick := &domain.Tick{
		StartAt:       startAt,
		FetchedAt:     fetchedAt,
		FetchDuration: fetchedAt.Sub(startAt).Milliseconds(),
		Avg:           domain.TickAvg{},
		Data:          make(map[domain.TickerName]*domain.Ticker),
	}

	// Build the tick using the fetched data
	i.buildTick(newTick, fetchedTickers)
	newTick.CreatedAt = time.Now()
	newTick.HandlingDuration = time.Since(newTick.FetchedAt)

	// Store the tick in the database
	if err := i.tickRepository.Create(ctx, *newTick); err != nil {
		return fmt.Errorf("failed to store tick in DB: %w", err)
	}

	return nil
}

// fetchTickers is a simple wrapper that calls exchange.FetchTickers
func (i *Importer) fetchTickers(ctx context.Context) ([]exchanges.Ticker, error) {
	return i.exchange.FetchTickers(ctx)
}

// buildTick calculates any indicators and populates domain.Tick
// this should never fail, we must always have valid data
func (i *Importer) buildTick(tick *domain.Tick, eTickers []exchanges.Ticker) {
	lastTick, _ := i.tickHistory.Last()

	// Calculate tickers indicators
	for _, eTicker := range eTickers {
		ticker := &domain.Ticker{
			Symbol:    domain.TickerName(eTicker.Symbol),
			Ask:       eTicker.AskPrice,
			Bid:       eTicker.BidPrice,
			EventAt:   eTicker.EventAt,
			CreatedAt: tick.StartAt,
		}

		if !ticker.IsValid() {
			// Skipping invalid ticker. Not necessarily an error.
			continue
		}

		i.addTickerHistory(ticker)
		ticker.CalculateIndicators(i.getTickerHistory(ticker.Symbol), lastTick)
		tick.SetTicker(ticker)
	}

	// Calculate the tick indicators
	i.addTickHistory(tick)
	tick.CalculateIndicators(i.tickHistory)
}

func (i *Importer) addTickHistory(tick *domain.Tick) {
	i.tickHistory.Push(tick)
}

// addTickerHistory updates the ring buffer for a particular ticker - 1 item per 1 minute
func (i *Importer) addTickerHistory(ticker *domain.Ticker) {
	history := i.getTickerHistory(ticker.Symbol)

	lastTickerData, err := i.getLastTicker(ticker.Symbol)
	// If there’s no data for this minute, push a new item
	if err != nil || !lastTickerData.CreatedAt.Truncate(time.Minute).Equal(ticker.CreatedAt.Truncate(time.Minute)) {
		ticker.Max = ticker.Ask
		ticker.Min = ticker.Ask
		history.Push(ticker)
		return
	}

	// Update the existing lastTickerData
	if ticker.Ask > lastTickerData.Max {
		lastTickerData.Max = ticker.Ask
	}
	if ticker.Ask < lastTickerData.Min {
		lastTickerData.Min = ticker.Ask
	}
	lastTickerData.Ask = ticker.Ask
	lastTickerData.Bid = ticker.Bid
	lastTickerData.CreatedAt = ticker.CreatedAt

	// mirror these changes in the newly pushed ticker object
	ticker.Max = lastTickerData.Max
	ticker.Min = lastTickerData.Min
}

func (i *Importer) getTickerHistory(tickerName domain.TickerName) *utils.RingBuffer[*domain.Ticker] {
	history, ok := i.tickerHistory[tickerName]
	if !ok {
		history = utils.NewRingBuffer[*domain.Ticker](domain.MaxTickHistory)
		i.tickerHistory[tickerName] = history
	}
	return history
}

func (i *Importer) getLastTicker(tickerName domain.TickerName) (*domain.Ticker, error) {
	history := i.getTickerHistory(tickerName)
	lastTicker, exists := history.Last()
	if !exists {
		return nil, fmt.Errorf("no ticker history found for %s", tickerName)
	}
	return lastTicker, nil
}
