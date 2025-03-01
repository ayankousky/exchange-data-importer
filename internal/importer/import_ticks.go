package importer

import (
	"context"
	"fmt"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges"
	"github.com/ayankousky/exchange-data-importer/pkg/utils/workerpool"
	"go.uber.org/zap"
)

// runTickersImport runs the tickers import process
func (i *Importer) runTickersImport(ctx context.Context) error {
	// Initialize the history data for calculating tick indicators
	if err := i.initHistory(ctx); err != nil {
		return fmt.Errorf("failed to init history: %w", err)
	}

	// Import should be started exactly at the beginning of the next second
	now := time.Now()
	nextSecond := now.Truncate(time.Second).Add(time.Second)
	time.Sleep(time.Until(nextSecond))

	// Start the import loop with the specified interval
	timeTicker := time.NewTicker(defaultTickInterval)
	defer timeTicker.Stop()

	i.logger.Info("Started tickers import")

	for {
		select {
		case <-ctx.Done():
			i.logger.Info("Tickers import stopped (context canceled)")
			return ctx.Err()
		case <-timeTicker.C:
			// Attempt to import a single "tick" of data
			if err := i.importTick(ctx); err != nil {
				i.logger.Error("Error importing tick", zap.Error(err))
			}
		}
	}
}

func (i *Importer) importTick(ctx context.Context) error {
	span, ctx := i.telemetry.StartSpan(ctx, telemetrySpanImportTick)
	defer span.Finish()

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
	newTick.HandlingDuration = time.Since(newTick.FetchedAt).Milliseconds()

	if err := newTick.Validate(); err != nil {
		return fmt.Errorf("tick validation failed: %w", err)
	}

	i.notifyNewTick(newTick)

	// Store the tick in the database
	if err := i.tickRepository.Create(ctx, *newTick); err != nil {
		return fmt.Errorf("failed to store tick in DB: %w", err)
	}

	return nil
}

// fetchTickers is a simple wrapper that calls exchange.FetchTickers
func (i *Importer) fetchTickers(ctx context.Context) ([]exchanges.Ticker, error) {
	span, ctx := i.telemetry.StartSpan(ctx, telemetrySpanFetchTickers)
	defer span.Finish()

	startTime := time.Now()
	tickers, err := i.exchange.FetchTickers(ctx)
	i.telemetry.Timing(telemetryTickFetchDuration, time.Since(startTime))
	i.telemetry.Gauge(telemetryTickFetchTickersCount, float64(len(tickers)))

	if err != nil {
		span.SetTag("error", true)
		span.SetTag("error.message", err.Error())
		i.telemetry.IncrementCounter(telemetryTickFetchErrors, 1)
	} else {
		span.SetTag("tickers.count", len(tickers))
	}

	return tickers, err
}

// buildTick calculates indicators and populates domain.Tick.
// This function should never fail; we must always ensure valid data is present.
// Note: For a small history length, concurrent processing is unnecessary.
// We can use a single-thread worker for exchanges where large calculations (such as RSI200) are not required.
func (i *Importer) buildTick(ctx context.Context, tick *domain.Tick, eTickers []exchanges.Ticker) {
	span, ctx := i.telemetry.StartSpan(ctx, telemetrySpanBuildTick)
	defer span.Finish()

	lastTick, _ := i.getLastTick()

	// Extract this to a separate function
	i.populateLiquidationsData(ctx, tick)

	// Extract ticker processing to a separate function
	i.processTickers(tick, lastTick, eTickers)

	// Calculate tick indicators
	indicatorsStart := time.Now()
	i.addTickHistory(tick)
	tick.CalculateIndicators(i.tickHistory.buffer)
	i.telemetry.Timing(telemetryTickCalculateIndicators, time.Since(indicatorsStart))
}

func (i *Importer) populateLiquidationsData(ctx context.Context, tick *domain.Tick) {
	liqStart := time.Now()
	liquidationsHistory, err := i.liquidationRepository.GetLiquidationsHistory(ctx, tick.StartAt)
	if err != nil {
		i.logger.Error("Error getting liquidations history", zap.Error(err))
	}
	tick.LL1 = liquidationsHistory.LongLiquidations1s
	tick.LL2 = liquidationsHistory.LongLiquidations2s
	tick.LL5 = liquidationsHistory.LongLiquidations5s
	tick.LL60 = liquidationsHistory.LongLiquidations60s
	tick.SL1 = liquidationsHistory.ShortLiquidations1s
	tick.SL2 = liquidationsHistory.ShortLiquidations2s
	tick.SL10 = liquidationsHistory.ShortLiquidations10s

	i.telemetry.Timing(telemetryTickBuildSetLiquidations, time.Since(liqStart))
}

func (i *Importer) processTickers(tick *domain.Tick, lastTick *domain.Tick, eTickers []exchanges.Ticker) {
	if len(eTickers) == 0 {
		return
	}

	numWorkers := workerpool.CalculateOptimalWorkers(len(eTickers))
	pool := workerpool.New[exchanges.Ticker](numWorkers)
	resultChannel := make(chan *domain.Ticker, len(eTickers))

	pool.Start(func(taskCh <-chan exchanges.Ticker) {
		for eTicker := range taskCh {
			ticker, err := i.buildTicker(*tick, lastTick, eTicker)
			if err != nil {
				i.logger.Error("Error building ticker", zap.Error(err))
				continue
			}
			resultChannel <- ticker
		}
	})

	for _, eTicker := range eTickers {
		pool.Submit(eTicker)
	}
	pool.CloseTaskChannel()

	go func() {
		pool.Wait()
		close(resultChannel)
	}()

	tickersProcessed := 0
	for processedTicker := range resultChannel {
		tick.SetTicker(processedTicker)
		tickersProcessed++
	}

	i.telemetry.Gauge(telemetryTickBuildTickersProcessed, float64(tickersProcessed))
}

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
	ticker.CalculateIndicators(i.tickerHistory.Get(ticker.Symbol), lastTick)
	return ticker, nil
}
