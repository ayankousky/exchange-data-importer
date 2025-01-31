package importer

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
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
	tickerMutex   sync.Mutex
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

// StartLiquidationsImport starts importing liquidations from the exchange
func (i *Importer) StartLiquidationsImport(ctx context.Context) {
	liqChan, errChan := i.exchange.SubscribeLiquidations(ctx)
	if liqChan == nil || errChan == nil {
		log.Printf("Failed to subscribe to liquidations for exchange %s\n", i.exchange.GetName())
		return
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				fmt.Println("Liquidation import stopped (context canceled).")
				return
			case liq := <-liqChan:
				// Convert the `exchanges.Liquidation` to your domain model
				domainLiq := domain.Liquidation{
					Order: domain.Order{
						Symbol:     domain.TickerName(liq.Symbol),
						EventAt:    liq.EventAt,
						Side:       domain.OrderSide(liq.Side),
						Price:      liq.Price,
						Quantity:   liq.Quantity,
						TotalPrice: liq.TotalPrice,
					},
					EventAt:  liq.EventAt,
					StoredAt: time.Now(),
				}
				// Store it
				err := i.liquidationRepository.Create(context.Background(), domainLiq)
				if err != nil {
					fmt.Printf("Failed storing liquidation: %v\n", err)
				}
			case err := <-errChan:
				fmt.Printf("Error on liquidation stream: %v\n", err)
			}
		}
	}()
}

// StartImportLoop starts a loop that imports data from the exchange periodically.
func (i *Importer) StartImportLoop(ctx context.Context, interval time.Duration) error {
	i.StartLiquidationsImport(ctx)
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
	i.buildTick(ctx, newTick, fetchedTickers)
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

// buildTick calculates indicators and populates domain.Tick.
// This function should never fail; we must always ensure valid data is present.
// Note: For a small history length, concurrent processing is unnecessary.
// We can use a single-thread worker for exchanges where large calculations (such as RSI200) are not required.
func (i *Importer) buildTick(ctx context.Context, tick *domain.Tick, eTickers []exchanges.Ticker) {
	lastTick, _ := i.tickHistory.Last()

	// Set liquidations data
	liquidationsHistory, err := i.liquidationRepository.GetLiquidationsHistory(ctx, tick.StartAt)
	if err != nil {
		log.Printf("Error getting liquidations history: %v", err)
	}
	tick.LL1 = liquidationsHistory.LongLiquidations1s
	tick.LL2 = liquidationsHistory.LongLiquidations2s
	tick.LL5 = liquidationsHistory.LongLiquidations5s
	tick.LL60 = liquidationsHistory.LongLiquidations60s
	tick.SL1 = liquidationsHistory.ShortLiquidations1s
	tick.SL2 = liquidationsHistory.ShortLiquidations2s
	tick.SL10 = liquidationsHistory.ShortLiquidations10s

	// Handle tickers data in parallel
	wg := sync.WaitGroup{}
	numWorkers := runtime.NumCPU()
	taskChannel := make(chan exchanges.Ticker, numWorkers)
	resultChannel := make(chan *domain.Ticker, len(eTickers))
	worker := func(tasks <-chan exchanges.Ticker, results chan<- *domain.Ticker) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Worker panic: %v\n", r)
			}
		}()

		for exchangeTicker := range tasks {
			ticker, err := i.buildTicker(*tick, lastTick, exchangeTicker)
			if err != nil {
				continue
			}
			results <- ticker
		}
	}

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker(taskChannel, resultChannel)
		}()
	}

	for _, eTicker := range eTickers {
		taskChannel <- eTicker
	}
	close(taskChannel)
	go func() {
		wg.Wait()
		close(resultChannel)
	}()
	for processedTicker := range resultChannel {
		tick.SetTicker(processedTicker)
	}

	// Calculate tick averages
	i.addTickHistory(tick)
	tick.CalculateIndicators(i.tickHistory)
}

func (i *Importer) buildTicker(currTick domain.Tick, lastTick *domain.Tick, eTicker exchanges.Ticker) (*domain.Ticker, error) {
	ticker := &domain.Ticker{
		Symbol:    domain.TickerName(eTicker.Symbol),
		Ask:       eTicker.AskPrice,
		Bid:       eTicker.BidPrice,
		EventAt:   eTicker.EventAt,
		CreatedAt: currTick.StartAt,
	}

	if !ticker.IsValid() {
		return nil, fmt.Errorf("invalid ticker data: %v", ticker)
	}

	i.addTickerHistory(ticker)
	ticker.CalculateIndicators(i.getTickerHistory(ticker.Symbol), lastTick)
	return ticker, nil
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
	// protect the map from concurrent reads
	i.tickerMutex.Lock()
	defer i.tickerMutex.Unlock()

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
