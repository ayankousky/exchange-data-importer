package importer

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges"
	"github.com/stretchr/testify/assert"
)

func TestBuildTick(t *testing.T) {
	defaultDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name               string
		tickers            []exchanges.Ticker
		expectedTickersLen int
		expectedLL60       int64
		expectedSL10       int64
	}{
		{
			name: "should build tick with valid tickers",
			tickers: []exchanges.Ticker{
				{Symbol: "BTCUSDT", AskPrice: 50000, BidPrice: 49900, EventAt: defaultDate},
				{Symbol: "ETHUSDT", AskPrice: 3000, BidPrice: 2990, EventAt: defaultDate},
			},
			expectedTickersLen: 2,
			expectedLL60:       600,
			expectedSL10:       10,
		},
		{
			name:               "should handle empty tickers",
			tickers:            []exchanges.Ticker{},
			expectedTickersLen: 0,
			expectedLL60:       600,
			expectedSL10:       10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := setupTest()
			ctx := context.Background()

			// Setup liquidation history mock
			ts.liqRepo.GetLiquidationsHistoryFunc = func(ctx context.Context, timeAt time.Time) (domain.LiquidationsHistory, error) {
				return domain.LiquidationsHistory{
					LongLiquidations60s:  tt.expectedLL60,
					ShortLiquidations10s: tt.expectedSL10,
				}, nil
			}

			tick := &domain.Tick{
				StartAt: time.Now(),
				Data:    make(map[domain.TickerName]*domain.Ticker),
			}

			ts.importer.buildTick(ctx, tick, tt.tickers)

			assert.Len(t, tick.Data, tt.expectedTickersLen)
			assert.Equal(t, tt.expectedLL60, tick.LL60)
			assert.Equal(t, tt.expectedSL10, tick.SL10)
		})
	}
}

func TestBuildTickerWithInvalidData(t *testing.T) {
	ts := setupTest()
	defaultDate := time.Now()

	tests := []struct {
		name      string
		ticker    exchanges.Ticker
		wantError bool
	}{
		{
			name: "should fail with zero ask price",
			ticker: exchanges.Ticker{
				Symbol:   "BTCUSDT",
				BidPrice: 49900,
				EventAt:  defaultDate,
			},
			wantError: true,
		},
		{
			name: "should fail with zero bid price",
			ticker: exchanges.Ticker{
				Symbol:   "BTCUSDT",
				AskPrice: 50000,
				EventAt:  defaultDate,
			},
			wantError: true,
		},
		{
			name: "should fail with empty symbol",
			ticker: exchanges.Ticker{
				AskPrice: 50000,
				BidPrice: 49900,
				EventAt:  defaultDate,
			},
			wantError: true,
		},
		{
			name: "should handle valid data",
			ticker: exchanges.Ticker{
				Symbol:   "BTCUSDT",
				AskPrice: 50000,
				BidPrice: 49900,
				EventAt:  defaultDate,
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tick := domain.Tick{
				StartAt: defaultDate,
				Data:    make(map[domain.TickerName]*domain.Ticker),
			}

			_, err := ts.importer.buildTicker(tick, nil, tt.ticker)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRunTickersImport(t *testing.T) {
	ts := setupTest()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup tick repository mock to record ticks
	var ticksCreated int
	ts.tickRepo.CreateFunc = func(ctx context.Context, tick domain.Tick) error {
		ticksCreated++
		return nil
	}

	// Setup tick repository for history
	ts.tickRepo.GetHistorySinceFunc = func(ctx context.Context, since time.Time) ([]domain.Tick, error) {
		return []domain.Tick{}, nil
	}

	// Start import in a goroutine
	importDone := make(chan error)
	go func() {
		importDone <- ts.importer.runTickersImport(ctx)
	}()

	// Let it process a few ticks
	time.Sleep(2100 * time.Millisecond)

	// Cancel context to stop import
	cancel()

	// Wait for import to finish
	err := <-importDone

	// Verify context cancellation was returned
	assert.ErrorIs(t, err, context.Canceled)

	// Verify ticks were created
	assert.GreaterOrEqual(t, ticksCreated, 1, "Expected at least one tick to be created")
}

func TestImportTick(t *testing.T) {
	ts := setupTest()
	ctx := context.Background()

	// Setup mocks
	ts.exchange.FetchTickersFunc = func(ctx context.Context) ([]exchanges.Ticker, error) {
		// Sleep to simulate delay in fetching ticker data
		time.Sleep(100 * time.Millisecond)
		return []exchanges.Ticker{
			{Symbol: "BTCUSDT", AskPrice: 50000, BidPrice: 49900, EventAt: time.Now(), AskQuantity: 100, BidQuantity: 200},
			{Symbol: "ETHUSDT", AskPrice: 3000, BidPrice: 2990, EventAt: time.Now(), AskQuantity: 100, BidQuantity: 200},
		}, nil
	}

	ts.liqRepo.GetLiquidationsHistoryFunc = func(ctx context.Context, timeAt time.Time) (domain.LiquidationsHistory, error) {
		// Sleep to simulate delay in fetching liquidations data
		time.Sleep(100 * time.Millisecond)
		return domain.LiquidationsHistory{
			LongLiquidations1s:   10,
			LongLiquidations2s:   20,
			LongLiquidations5s:   30,
			LongLiquidations60s:  100,
			ShortLiquidations1s:  5,
			ShortLiquidations2s:  15,
			ShortLiquidations10s: 25,
		}, nil
	}

	var createdTick domain.Tick
	ts.tickRepo.CreateFunc = func(ctx context.Context, tick domain.Tick) error {
		createdTick = tick
		return nil
	}

	// Call importTick
	err := ts.importer.importTick(ctx)

	// Verify no error
	assert.NoError(t, err)

	// Verify tick data
	assert.NotNil(t, createdTick)
	assert.Len(t, createdTick.Data, 2)
	assert.NotZero(t, createdTick.StartAt, "StartAt should be set")
	assert.NotZero(t, createdTick.FetchedAt, "FetchedAt should be set")
	assert.NotZero(t, createdTick.CreatedAt, "CreatedAt should be set")
	assert.NotZero(t, createdTick.FetchDuration, "FetchDuration should be set")
	assert.NotZero(t, createdTick.HandlingDuration, "HandlingDuration should be set")

	// Verify liquidations data
	assert.Equal(t, int64(10), createdTick.LL1)
	assert.Equal(t, int64(20), createdTick.LL2)
	assert.Equal(t, int64(30), createdTick.LL5)
	assert.Equal(t, int64(100), createdTick.LL60)
	assert.Equal(t, int64(5), createdTick.SL1)
	assert.Equal(t, int64(15), createdTick.SL2)
	assert.Equal(t, int64(25), createdTick.SL10)

	// Verify tickers were processed
	assert.Contains(t, createdTick.Data, domain.TickerName("BTCUSDT"))
	assert.Contains(t, createdTick.Data, domain.TickerName("ETHUSDT"))
}

func TestFetchTickers(t *testing.T) {
	ts := setupTest()
	ctx := context.Background()

	tests := []struct {
		name         string
		setupMock    func()
		expectedLen  int
		expectError  bool
		errorMessage string
	}{
		{
			name: "successful fetch",
			setupMock: func() {
				ts.exchange.FetchTickersFunc = func(ctx context.Context) ([]exchanges.Ticker, error) {
					return []exchanges.Ticker{
						{Symbol: "BTCUSDT", AskPrice: 50000, BidPrice: 49900},
						{Symbol: "ETHUSDT", AskPrice: 3000, BidPrice: 2990},
					}, nil
				}
			},
			expectedLen: 2,
			expectError: false,
		},
		{
			name: "empty result",
			setupMock: func() {
				ts.exchange.FetchTickersFunc = func(ctx context.Context) ([]exchanges.Ticker, error) {
					return []exchanges.Ticker{}, nil
				}
			},
			expectedLen: 0,
			expectError: false,
		},
		{
			name: "exchange error",
			setupMock: func() {
				ts.exchange.FetchTickersFunc = func(ctx context.Context) ([]exchanges.Ticker, error) {
					return nil, fmt.Errorf("exchange API error")
				}
			},
			expectedLen:  0,
			expectError:  true,
			errorMessage: "exchange API error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			tickers, err := ts.importer.fetchTickers(ctx)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMessage)
			} else {
				assert.NoError(t, err)
				assert.Len(t, tickers, tt.expectedLen)
			}
		})
	}
}

func TestProcessTickers(t *testing.T) {
	ts := setupTest()

	tests := []struct {
		name          string
		inputTickers  []exchanges.Ticker
		expectedCount int
	}{
		{
			name: "process multiple tickers",
			inputTickers: []exchanges.Ticker{
				{Symbol: "BTCUSDT", AskPrice: 50000, BidPrice: 49900, EventAt: time.Now()},
				{Symbol: "ETHUSDT", AskPrice: 3000, BidPrice: 2990, EventAt: time.Now()},
				{Symbol: "SOLUSDT", AskPrice: 100, BidPrice: 99, EventAt: time.Now()},
			},
			expectedCount: 3,
		},
		{
			name:          "handle empty tickers",
			inputTickers:  []exchanges.Ticker{},
			expectedCount: 0,
		},
		{
			name: "process with invalid tickers",
			inputTickers: []exchanges.Ticker{
				{Symbol: "BTCUSDT", AskPrice: 50000, BidPrice: 49900, EventAt: time.Now()},
				{Symbol: "", AskPrice: 0, BidPrice: 0, EventAt: time.Now()}, // Invalid
				{Symbol: "SOLUSDT", AskPrice: 100, BidPrice: 99, EventAt: time.Now()},
			},
			expectedCount: 2, // Only valid tickers should be processed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new tick
			tick := &domain.Tick{
				StartAt: time.Now(),
				Data:    make(map[domain.TickerName]*domain.Ticker),
			}

			// Process tickers
			ts.importer.processTickers(tick, nil, tt.inputTickers)

			// Verify expected number of tickers processed
			assert.Len(t, tick.Data, tt.expectedCount)
		})
	}
}

func TestPopulateLiquidationsData(t *testing.T) {
	ts := setupTest()
	ctx := context.Background()

	tests := []struct {
		name           string
		liquidations   domain.LiquidationsHistory
		repoError      error
		expectedValues func(tick *domain.Tick) bool
	}{
		{
			name: "populate all liquidation fields",
			liquidations: domain.LiquidationsHistory{
				LongLiquidations1s:   123,
				LongLiquidations2s:   234,
				LongLiquidations5s:   345,
				LongLiquidations60s:  456,
				ShortLiquidations1s:  111,
				ShortLiquidations2s:  222,
				ShortLiquidations10s: 333,
			},
			repoError: nil,
			expectedValues: func(tick *domain.Tick) bool {
				return tick.LL1 == 123 &&
					tick.LL2 == 234 &&
					tick.LL5 == 345 &&
					tick.LL60 == 456 &&
					tick.SL1 == 111 &&
					tick.SL2 == 222 &&
					tick.SL10 == 333
			},
		},
		{
			name:         "handle repository error",
			liquidations: domain.LiquidationsHistory{},
			repoError:    fmt.Errorf("database error"),
			expectedValues: func(tick *domain.Tick) bool {
				return tick.LL1 == 0 &&
					tick.LL2 == 0 &&
					tick.LL5 == 0 &&
					tick.LL60 == 0 &&
					tick.SL1 == 0 &&
					tick.SL2 == 0 &&
					tick.SL10 == 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup repository mock
			ts.liqRepo.GetLiquidationsHistoryFunc = func(ctx context.Context, timeAt time.Time) (domain.LiquidationsHistory, error) {
				return tt.liquidations, tt.repoError
			}

			// Create a new tick
			tick := &domain.Tick{
				StartAt: time.Now(),
				Data:    make(map[domain.TickerName]*domain.Ticker),
			}

			// Populate liquidations data
			ts.importer.populateLiquidationsData(ctx, tick)

			// Verify expected values
			assert.True(t, tt.expectedValues(tick))
		})
	}
}
