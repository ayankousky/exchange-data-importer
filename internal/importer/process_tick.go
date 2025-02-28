package importer

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges"
	"go.uber.org/zap"
)

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

	// Set liquidations data
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

	// Handle tickers data in parallel
	wg := sync.WaitGroup{}
	numWorkers := runtime.NumCPU()
	taskChannel := make(chan exchanges.Ticker, numWorkers)
	resultChannel := make(chan *domain.Ticker, len(eTickers))
	worker := func(tasks <-chan exchanges.Ticker, results chan<- *domain.Ticker) {
		defer func() {
			if r := recover(); r != nil {
				i.logger.Error("Worker panic", zap.Any("panic", r))
			}
		}()

		for exchangeTicker := range tasks {
			ticker, err := i.buildTicker(*tick, lastTick, exchangeTicker)
			if err != nil {
				i.logger.Error("Error building ticker", zap.Error(err))
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

	tickersProcessed := 0
	for processedTicker := range resultChannel {
		tick.SetTicker(processedTicker)
		tickersProcessed++
	}

	i.telemetry.Gauge(telemetryTickBuildTickersProcessed, float64(tickersProcessed))

	// Calculate tick indicators
	indicatorsStart := time.Now()
	i.addTickHistory(tick)
	tick.CalculateIndicators(i.tickHistory.buffer)
	i.telemetry.Timing(telemetryTickCalculateIndicators, time.Since(indicatorsStart))
}
