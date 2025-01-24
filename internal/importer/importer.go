package importer

import (
	"context"
	"fmt"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges"
	"github.com/ayankousky/exchange-data-importer/pkg/utils"
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

	tickerHistory map[domain.TickerName]*utils.RingBuffer[*domain.Ticker]
	tickHistory   *utils.RingBuffer[*domain.Tick]
}

// NewImporter creates a new Importer
func NewImporter(exchange exchanges.Exchange, repositoryFactory RepositoryFactory) *Importer {
	return &Importer{
		exchange:              exchange,
		tickRepository:        repositoryFactory.GetTickRepository(exchange.GetName()),
		liquidationRepository: repositoryFactory.GetLiquidationRepository(exchange.GetName()),

		tickerHistory: make(map[domain.TickerName]*utils.RingBuffer[*domain.Ticker]),
		tickHistory:   utils.NewRingBuffer[*domain.Tick](MaxTickHistory),
	}
}

func (i *Importer) importTickers() error {
	startAt := time.Now()
	// Fetch tickers from the exchange
	fetchedTickers, err := i.fetchTickers()
	if err != nil {
		fmt.Printf("Error fetching tickers: %v\n", err)
		return err
	}
	fetchedAt := time.Now()

	// Create a new tick
	newTick := &domain.Tick{
		StartAt:       startAt,
		FetchedAt:     fetchedAt,
		FetchDuration: fetchedAt.Sub(startAt).Milliseconds(),
		Avg:           domain.TickAvg{},
		Data:          make(map[domain.TickerName]*domain.Ticker, 0),
	}
	err = i.buildTick(newTick, fetchedTickers)
	if err != nil {
		fmt.Printf("Error building tick: %v\n", err)
		return err
	}
	newTick.CreatedAt = time.Now()
	newTick.HandlingDuration = time.Since(newTick.FetchedAt)

	// Store the tick in the database
	err = i.tickRepository.Create(context.Background(), *newTick)
	if err != nil {
		fmt.Printf("Error storing tick: %v\n", err)
	}
	return nil
}
func (i *Importer) fetchTickers() ([]exchanges.Ticker, error) {
	return i.exchange.FetchTickers(context.Background())
}
func (i *Importer) buildTick(tick *domain.Tick, eTickers []exchanges.Ticker) error {
	lastTick, _ := i.tickHistory.Last()
	// calculate tickers indicators
	for _, eTicker := range eTickers {
		ticker := &domain.Ticker{
			Symbol:    domain.TickerName(eTicker.Symbol),
			Ask:       eTicker.AskPrice,
			Bid:       eTicker.BidPrice,
			EventAt:   eTicker.EventAt,
			CreatedAt: tick.StartAt,
		}

		if !ticker.IsValid() {
			continue
		}

		i.addTickerHistory(ticker)
		if lastTick != nil {
			ticker.CalculateIndicators(i.getTickerHistory(ticker.Symbol), lastTick)
		}
		tick.SetTicker(ticker)
	}

	// calculate the tick indicators
	i.addTickHistory(tick)
	tick.CalculateIndicators(i.tickHistory)

	return nil
}

// StartImportEverySecond starts a loop that imports data from the exchange every second
// This simulates a real-time import process and stores the results in the database.
func (i *Importer) StartImportEverySecond() {
	i.initHistory()
	for {
		// continue the loop every second
		now := time.Now()
		next := now.Truncate(time.Second).Add(time.Second)
		time.Sleep(time.Until(next))

		err := i.importTickers()
		if err != nil {
			fmt.Println(now)
		}
	}
}

func (i *Importer) addTickHistory(tick *domain.Tick) {
	i.tickHistory.Push(tick)
}

// history is a map of TickerName to a list of Ticker data for that symbol
// 1 item = 1 minute of data (no need to store for each second)
func (i *Importer) addTickerHistory(ticker *domain.Ticker) {

	history := i.getTickerHistory(ticker.Symbol)

	// Retrieve the last ticker data for this symbol, if it exists
	lastTickerData, err := i.getLastTicker(ticker.Symbol)
	// If there is no data for this minute, create a new history item
	if err != nil || !lastTickerData.CreatedAt.Truncate(time.Minute).Equal(ticker.CreatedAt.Truncate(time.Minute)) {
		ticker.Max = ticker.Ask
		ticker.Min = ticker.Ask
		history.Push(ticker)
		return
	}

	// Update the existing lastTickerData directly
	if ticker.Ask > lastTickerData.Max {
		lastTickerData.Max = ticker.Ask
	}
	if ticker.Ask < lastTickerData.Min {
		lastTickerData.Min = ticker.Ask
	}
	lastTickerData.Ask = ticker.Ask
	lastTickerData.Bid = ticker.Bid
	lastTickerData.CreatedAt = ticker.CreatedAt

	ticker.Max = lastTickerData.Max
	ticker.Min = lastTickerData.Min
}

func (i *Importer) initHistory() {
	history, _ := i.tickRepository.GetHistorySince(context.Background(), time.Now().Add(-MaxTickHistory*time.Minute))
	for _, tick := range history {
		i.addTickHistory(&tick)
		for _, ticker := range tick.Data {
			i.addTickerHistory(ticker)
		}
	}
}

func (i *Importer) getTickerHistory(tickerName domain.TickerName) *utils.RingBuffer[*domain.Ticker] {
	history, ok := i.tickerHistory[tickerName]
	if !ok {
		history = utils.NewRingBuffer[*domain.Ticker](MaxTickHistory)
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
