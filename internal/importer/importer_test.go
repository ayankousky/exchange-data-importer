package importer

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
	domainMocks "github.com/ayankousky/exchange-data-importer/internal/domain/mocks"
	importerMocks "github.com/ayankousky/exchange-data-importer/internal/importer/mocks"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges"
	exchangeMocks "github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges/mocks"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/notify"
	notifyMock "github.com/ayankousky/exchange-data-importer/internal/infrastructure/notify/mocks"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/telemetry"
	"github.com/ayankousky/exchange-data-importer/internal/notifier"
	"github.com/ayankousky/exchange-data-importer/pkg/utils/mathutils"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type testSuite struct {
	exchange    *exchangeMocks.ExchangeMock
	repoFactory *importerMocks.RepositoryFactoryMock
	tickRepo    *domainMocks.TickRepositoryMock
	liqRepo     *domainMocks.LiquidationRepositoryMock
	importer    *Importer
}

func setupTest() *testSuite {
	exchange := &exchangeMocks.ExchangeMock{
		GetNameFunc: func() string {
			return "mockExchange"
		},
		FetchTickersFunc: func(ctx context.Context) ([]exchanges.Ticker, error) {
			return []exchanges.Ticker{
				{Symbol: "BTCUSDT", AskPrice: 50000, BidPrice: 49900},
				{Symbol: "ETHUSDT", AskPrice: 3000, BidPrice: 2990},
			}, nil
		},
		SubscribeLiquidationsFunc: func(ctx context.Context) (<-chan exchanges.Liquidation, <-chan error) {
			liquidChan := make(chan exchanges.Liquidation)
			errChan := make(chan error)
			return liquidChan, errChan
		},
	}

	tickRepo := &domainMocks.TickRepositoryMock{
		CreateFunc: func(ctx context.Context, ts domain.Tick) error {
			return nil
		},
		GetHistorySinceFunc: func(ctx context.Context, since time.Time) ([]domain.Tick, error) {
			return []domain.Tick{}, nil
		},
	}

	liqRepo := &domainMocks.LiquidationRepositoryMock{
		CreateFunc: func(ctx context.Context, l domain.Liquidation) error {
			return nil
		},
		GetLiquidationsHistoryFunc: func(ctx context.Context, timeAt time.Time) (domain.LiquidationsHistory, error) {
			return domain.LiquidationsHistory{}, nil
		},
	}

	repoFactory := &importerMocks.RepositoryFactoryMock{
		GetTickRepositoryFunc: func(name string) (domain.TickRepository, error) {
			return tickRepo, nil
		},
		GetLiquidationRepositoryFunc: func(name string) (domain.LiquidationRepository, error) {
			return liqRepo, nil
		},
	}

	telemetryProvider := &telemetry.NoopProvider{}

	cfg := &Config{
		Exchange:          exchange,
		RepositoryFactory: repoFactory,
		NotifierService:   notifier.New(zap.NewNop()),
		Telemetry:         telemetryProvider,
		Logger:            zap.NewNop(),
	}

	return &testSuite{
		exchange:    exchange,
		repoFactory: repoFactory,
		tickRepo:    tickRepo,
		liqRepo:     liqRepo,
		importer:    New(cfg),
	}
}

func TestStartImport(t *testing.T) {
	ts := setupTest()
	ctx := context.Background()

	tickers, err := ts.exchange.FetchTickers(ctx)
	assert.Equal(t, 2, len(tickers))
	assert.NoError(t, err)

	err = ts.importer.importTick(ctx)
	assert.NoError(t, err)
}

func TestTickerHistory(t *testing.T) {
	ts := setupTest()
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 1000; i++ {
		ticker := &domain.Ticker{
			Symbol:    "BTCUSDT",
			Ask:       50000 + float64(i),
			Bid:       49950 + float64(i),
			CreatedAt: startDate.Add(time.Second * time.Duration(i)),
		}
		ts.importer.addTickerHistory(ticker)
	}

	tickerHistory := ts.importer.tickerHistory.Get("BTCUSDT")
	lastItem, _ := tickerHistory.Last()

	assert.Equal(t, 17, tickerHistory.Len(), "Only 1 ticker per minute should be stored")
	assert.Equal(t, 39, lastItem.CreatedAt.Second(), "Last inserted ticker should be at the 39th second")
	assert.Equal(t, 59, tickerHistory.At(tickerHistory.Len()-2).CreatedAt.Second(), "Last second inserted ticker should be at the 59th second")
	assert.Equal(t, 59, tickerHistory.At(tickerHistory.Len()-3).CreatedAt.Second(), "Last third inserted ticker should be at the 59th second")

	// Test history limit
	for i := 0; i < (60+10)*domain.MaxTickHistory; i++ {
		ticker := &domain.Ticker{
			Symbol:    "BTCUSDT",
			Ask:       50000 + float64(i),
			Bid:       49950 + float64(i),
			CreatedAt: startDate.Add(time.Second * time.Duration(i)),
		}
		ts.importer.addTickerHistory(ticker)
	}
	assert.Equal(t, domain.MaxTickHistory, ts.importer.tickerHistory.Get("BTCUSDT").Len(), "Ticker history should be limited")
}

func TestCorruptedData(t *testing.T) {
	ts := setupTest()
	startDate := time.Now().Truncate(time.Hour)

	for i := 0; i < 1500; i++ {
		ticker := &domain.Ticker{
			Symbol:    "BTCUSDT",
			Ask:       50000,
			Bid:       49950,
			CreatedAt: startDate.Add(time.Second),
		}
		ts.importer.addTickerHistory(ticker)
	}

	history := ts.importer.tickerHistory.Get("BTCUSDT")
	assert.Equal(t, 1, history.Len(), "Only 1 ticker per minute should be stored")
}

func TestInitHistory(t *testing.T) {
	ts := setupTest()
	ctx := context.Background()

	// Update mock for historical data
	ts.tickRepo.GetHistorySinceFunc = func(ctx context.Context, since time.Time) ([]domain.Tick, error) {
		ticks := make([]domain.Tick, 1000)
		defaultDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		for i := 0; i < 1000; i++ {
			multiplier := 1.0 + float64(i)/200
			ticks[i] = domain.Tick{
				Data: map[domain.TickerName]*domain.Ticker{
					"BTCUSDT": {
						Symbol:    "BTCUSDT",
						Ask:       mathutils.Round(104388.7*multiplier, 6),
						Bid:       mathutils.Round(104388.6*multiplier, 6),
						CreatedAt: defaultDate.Add(time.Second * time.Duration(i)),
					},
				},
			}
		}
		return ticks, nil
	}

	err := ts.importer.initHistory(ctx)
	assert.NoError(t, err)

	assert.Equal(t, domain.MaxTickHistory, ts.importer.tickHistory.Len())
	assert.Equal(t, 17, ts.importer.tickerHistory.Get("BTCUSDT").Len())

	lastTick, exists := ts.importer.tickHistory.Last()
	btcHistory := ts.importer.tickerHistory.Get("BTCUSDT")
	assert.True(t, exists)
	assert.Equal(t, 625810.2565, lastTick.Data["BTCUSDT"].Ask)
	assert.Equal(t, 625810.2565, btcHistory.At(btcHistory.Len()-1).Ask)
	assert.Equal(t, 604932.5165, btcHistory.At(btcHistory.Len()-2).Ask)
	assert.Equal(t, 573615.9065, btcHistory.At(btcHistory.Len()-3).Ask)
	assert.Equal(t, lastTick.Data["BTCUSDT"].Ask, btcHistory.At(btcHistory.Len()-1).Ask)

	// Test error scenario
	ts.tickRepo.GetHistorySinceFunc = func(ctx context.Context, since time.Time) ([]domain.Tick, error) {
		return nil, fmt.Errorf("database error")
	}
	err = ts.importer.initHistory(ctx)
	assert.Error(t, err, "Error in fetching history should return an error")
}

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

func TestNotifyNewTick(t *testing.T) {
	tests := []struct {
		name          string
		tick          *domain.Tick
		notifierCount int
		wantEventType string
		wantCalls     int
	}{
		{
			name: "should notify multiple tickers to single notifier",
			tick: &domain.Tick{
				Data: map[domain.TickerName]*domain.Ticker{
					"BTCUSDT": {
						Symbol: "BTCUSDT",
						Ask:    45000.00,
						Bid:    44990.00,
					},
					"ETHUSDT": {
						Symbol: "ETHUSDT",
						Ask:    3000.00,
						Bid:    2999.00,
					},
				},
			},
			notifierCount: 1,
			wantEventType: string(notifier.MarketDataTopic),
			wantCalls:     2, // One call per ticker
		},
		{
			name: "should notify single ticker to multiple notifiers",
			tick: &domain.Tick{
				Data: map[domain.TickerName]*domain.Ticker{
					"BTCUSDT": {
						Symbol: "BTCUSDT",
						Ask:    45000.00,
						Bid:    44990.00,
					},
				},
			},
			notifierCount: 3,
			wantEventType: string(notifier.MarketDataTopic),
			wantCalls:     3, // One call per notifier
		},
		{
			name: "should handle empty tick data",
			tick: &domain.Tick{
				Data: map[domain.TickerName]*domain.Ticker{},
			},
			notifierCount: 1,
			wantEventType: string(notifier.MarketDataTopic),
			wantCalls:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test suite
			ts := setupTest()

			// Create and configure notifier mocks
			notifiers := make([]*notifyMock.ClientMock, tt.notifierCount)
			for i := 0; i < tt.notifierCount; i++ {
				n := &notifyMock.ClientMock{
					SendFunc: func(ctx context.Context, event notify.Event) error {
						// Verify event properties
						assert.Equal(t, tt.wantEventType, event.EventType)
						assert.NotZero(t, event.Time)
						assert.NotNil(t, event.Data)
						return nil
					},
				}
				notifiers[i] = n

				// Create strategy mock with implementation
				strategy := &notifyMock.StrategyMock{
					FormatFunc: func(data any) []notify.Event {
						tick, ok := data.(*domain.Tick)
						if !ok {
							return nil
						}
						// If tick is empty, return empty events
						if len(tick.Data) == 0 {
							return nil
						}
						// Return one event per ticker
						events := make([]notify.Event, 0, len(tick.Data))
						for _ = range tick.Data {
							events = append(events, notify.Event{
								Time:      time.Now(),
								EventType: tt.wantEventType,
								Data:      data,
							})
						}
						return events
					},
				}

				ts.importer.WithNotifier(n, string(notifier.MarketDataTopic), strategy)
			}

			// Execute the notification
			ts.importer.notifyNewTick(tt.tick)

			// Verify the notifier
			totalCalls := 0
			for _, notifier := range notifiers {
				calls := len(notifier.SendCalls())
				totalCalls += calls

				// For each call, verify the context was passed
				for _, call := range notifier.SendCalls() {
					assert.NotNil(t, call.Ctx)
				}
			}

			assert.Equal(t, tt.wantCalls, totalCalls, "unexpected number of notification calls")
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

func TestInitHistoryWithErrors(t *testing.T) {
	ts := setupTest()
	ctx := context.Background()

	tests := []struct {
		name       string
		setupMocks func()
		wantError  bool
	}{
		{
			name: "should handle repository error",
			setupMocks: func() {
				ts.tickRepo.GetHistorySinceFunc = func(ctx context.Context, since time.Time) ([]domain.Tick, error) {
					return nil, fmt.Errorf("database error")
				}
			},
			wantError: true,
		},
		{
			name: "should handle empty history",
			setupMocks: func() {
				ts.tickRepo.GetHistorySinceFunc = func(ctx context.Context, since time.Time) ([]domain.Tick, error) {
					return []domain.Tick{}, nil
				}
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := ts.importer.initHistory(ctx)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTickerHistoryDataRace(t *testing.T) {
	ts := setupTest()

	const numGoroutines = 10
	const numOperations = 100

	wg := sync.WaitGroup{}
	wg.Add(numGoroutines)

	startTime := time.Now()

	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				ticker := &domain.Ticker{
					Symbol:    "BTCUSDT",
					Ask:       float64(50000 + routineID*j),
					Bid:       float64(49900 + routineID*j),
					CreatedAt: startTime.Add(time.Duration(j) * time.Second),
				}
				ts.importer.addTickerHistory(ticker)
			}
		}(i)
	}

	wg.Wait()

	history := ts.importer.tickerHistory.Get("BTCUSDT")
	assert.LessOrEqual(t, history.Len(), domain.MaxTickHistory)

	// Verify no duplicates for the same minute
	minutes := make(map[time.Time]bool)
	for i := 0; i < history.Len(); i++ {
		ticker := history.At(i)
		minute := ticker.CreatedAt.Truncate(time.Minute)
		assert.False(t, minutes[minute], "Found duplicate minute in history")
		minutes[minute] = true
	}
}

func TestConvertLiquidationToDomain(t *testing.T) {
	// Setup test suite
	ts := setupTest()

	// Fixed test time for consistency
	testTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		input      exchanges.Liquidation
		want       domain.Liquidation
		setupClock func() time.Time
	}{
		{
			name: "should convert long liquidation correctly",
			input: exchanges.Liquidation{
				Symbol:     "BTCUSDT",
				Side:       "SELL",
				Price:      50000.0,
				Quantity:   1.5,
				TotalPrice: 75000.0,
				EventAt:    testTime,
			},
			setupClock: func() time.Time {
				return testTime.Add(time.Second)
			},
			want: domain.Liquidation{
				Order: domain.Order{
					Symbol:     "BTCUSDT",
					EventAt:    testTime,
					Side:       domain.OrderSideSell,
					Price:      50000.0,
					Quantity:   1.5,
					TotalPrice: 75000.0,
				},
				EventAt:  testTime,
				StoredAt: testTime.Add(time.Second),
			},
		},
		{
			name: "should convert short liquidation correctly",
			input: exchanges.Liquidation{
				Symbol:     "ETHUSDT",
				Side:       "BUY",
				Price:      3000.0,
				Quantity:   10.0,
				TotalPrice: 30000.0,
				EventAt:    testTime,
			},
			setupClock: func() time.Time {
				return testTime.Add(time.Second * 2)
			},
			want: domain.Liquidation{
				Order: domain.Order{
					Symbol:     "ETHUSDT",
					EventAt:    testTime,
					Side:       domain.OrderSideBuy,
					Price:      3000.0,
					Quantity:   10.0,
					TotalPrice: 30000.0,
				},
				EventAt:  testTime,
				StoredAt: testTime.Add(time.Second * 2),
			},
		},
		{
			name: "should handle zero values correctly",
			input: exchanges.Liquidation{
				Symbol:     "SOLUSDT",
				Side:       "SELL",
				Price:      0.0,
				Quantity:   0.0,
				TotalPrice: 0.0,
				EventAt:    testTime,
			},
			setupClock: func() time.Time {
				return testTime.Add(time.Second * 3)
			},
			want: domain.Liquidation{
				Order: domain.Order{
					Symbol:     "SOLUSDT",
					EventAt:    testTime,
					Side:       domain.OrderSideSell,
					Price:      0.0,
					Quantity:   0.0,
					TotalPrice: 0.0,
				},
				EventAt:  testTime,
				StoredAt: testTime.Add(time.Second * 3),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ts.importer.convertLiquidationToDomain(tt.input)

			// Verify result matches expected values
			assert.Equal(t, tt.want.Order.Symbol, result.Order.Symbol)
			assert.Equal(t, tt.want.Order.EventAt, result.Order.EventAt)
			assert.Equal(t, tt.want.Order.Side, result.Order.Side)
			assert.Equal(t, tt.want.Order.Price, result.Order.Price)
			assert.Equal(t, tt.want.Order.Quantity, result.Order.Quantity)
			assert.Equal(t, tt.want.Order.TotalPrice, result.Order.TotalPrice)
			assert.Equal(t, tt.want.EventAt, result.EventAt)

			assert.NotZero(t, result.StoredAt)
			assert.True(t, result.StoredAt.After(result.EventAt) || result.StoredAt.Equal(result.EventAt))
		})
	}
}

func TestConvertLiquidationToDomainValidation(t *testing.T) {
	ts := setupTest()

	// Test that converted liquidations pass validation
	testTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		input   exchanges.Liquidation
		wantErr bool
	}{
		{
			name: "valid long liquidation should pass validation",
			input: exchanges.Liquidation{
				Symbol:     "BTCUSDT",
				Side:       "SELL",
				Price:      50000.0,
				Quantity:   1.5,
				TotalPrice: 75000.0,
				EventAt:    testTime,
			},
			wantErr: false,
		},
		{
			name: "valid short liquidation should pass validation",
			input: exchanges.Liquidation{
				Symbol:     "ETHUSDT",
				Side:       "BUY",
				Price:      3000.0,
				Quantity:   10.0,
				TotalPrice: 30000.0,
				EventAt:    testTime,
			},
			wantErr: false,
		},
		{
			name: "zero price should not pass validation",
			input: exchanges.Liquidation{
				Symbol:     "SOLUSDT",
				Side:       "SELL",
				Price:      0.0,
				Quantity:   5.0,
				TotalPrice: 0.0,
				EventAt:    testTime,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domainLiq := ts.importer.convertLiquidationToDomain(tt.input)

			err := domainLiq.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
