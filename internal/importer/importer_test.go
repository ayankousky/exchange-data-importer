package importer

import (
	"context"
	"fmt"
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

func TestNew(t *testing.T) {
	// Test successful creation
	exchange := &exchangeMocks.ExchangeMock{
		GetNameFunc: func() string {
			return "mockExchange"
		},
	}

	tickRepo := &domainMocks.TickRepositoryMock{}
	liqRepo := &domainMocks.LiquidationRepositoryMock{}

	repoFactory := &importerMocks.RepositoryFactoryMock{
		GetTickRepositoryFunc: func(name string) (domain.TickRepository, error) {
			assert.Equal(t, "mockExchange", name)
			return tickRepo, nil
		},
		GetLiquidationRepositoryFunc: func(name string) (domain.LiquidationRepository, error) {
			assert.Equal(t, "mockExchange", name)
			return liqRepo, nil
		},
	}

	notifierService := notifier.New(zap.NewNop())
	telemetryProvider := telemetry.NoopProvider{}
	logger := zap.NewNop()

	cfg := &Config{
		Exchange:          exchange,
		RepositoryFactory: repoFactory,
		NotifierService:   notifierService,
		Telemetry:         &telemetryProvider,
		Logger:            logger,
	}

	importer := New(cfg)
	assert.NotNil(t, importer)
	assert.Equal(t, exchange, importer.exchange)
	assert.Equal(t, tickRepo, importer.tickRepository)
	assert.Equal(t, liqRepo, importer.liquidationRepository)
	assert.Equal(t, notifierService, importer.notifier)
	assert.Equal(t, &telemetryProvider, importer.telemetry)
	assert.Equal(t, logger, importer.logger)
	assert.False(t, importer.importTicks)
	assert.False(t, importer.importLiquidations)

	// Test creation with tick repository errors
	repoFactory.GetTickRepositoryFunc = func(name string) (domain.TickRepository, error) {
		return nil, fmt.Errorf("tick repo error")
	}

	importer = New(cfg)
	assert.Nil(t, importer)

	// Reset and test liquidation repository error
	repoFactory.GetTickRepositoryFunc = func(name string) (domain.TickRepository, error) {
		return tickRepo, nil
	}
	repoFactory.GetLiquidationRepositoryFunc = func(name string) (domain.LiquidationRepository, error) {
		return nil, fmt.Errorf("liquidation repo error")
	}

	importer = New(cfg)
	assert.Nil(t, importer)
}

func TestWithTicksImport(t *testing.T) {
	ts := setupTest()

	// Test successful enabling
	result := ts.importer.WithTicksImport()
	assert.Equal(t, ts.importer, result)
	assert.True(t, ts.importer.importTicks)

	// Test with nil repository
	ts.importer.tickRepository = nil
	result = ts.importer.WithTicksImport()
	assert.Equal(t, ts.importer, result)
	assert.False(t, ts.importer.importTicks)
}

func TestWithLiquidationsImport(t *testing.T) {
	ts := setupTest()

	// Test successful enabling
	result := ts.importer.WithLiquidationsImport()
	assert.Equal(t, ts.importer, result)
	assert.True(t, ts.importer.importLiquidations)

	// Test with nil repository
	ts.importer.liquidationRepository = nil
	result = ts.importer.WithLiquidationsImport()
	assert.Equal(t, ts.importer, result)
	assert.False(t, ts.importer.importLiquidations)
}

func TestRun(t *testing.T) {
	ts := setupTest()

	// Test with no imports enabled
	ts.importer.importTicks = false
	ts.importer.importLiquidations = false
	err := ts.importer.Run(context.Background())
	assert.NoError(t, err)

	// Test with ticks import only
	ts.importer.importTicks = true
	ts.importer.importLiquidations = false

	// Setup tick repository to immediately return when called
	initHistoryCalled := false
	ts.tickRepo.GetHistorySinceFunc = func(ctx context.Context, since time.Time) ([]domain.Tick, error) {
		initHistoryCalled = true
		return []domain.Tick{}, nil
	}

	// Setup ticker fetch to cancel context when called
	fetchCalled := false
	ts.exchange.FetchTickersFunc = func(ctx context.Context) ([]exchanges.Ticker, error) {
		fetchCalled = true
		return []exchanges.Ticker{}, nil
	}

	// Run with cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// We need to wait longer than the sleep period in runTickersImport
	// The import waits until the beginning of the next second
	go func() {
		// Wait 2 seconds to ensure the ticker import has started
		time.Sleep(2 * time.Second)
		cancel()
	}()

	err = ts.importer.Run(ctx)
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.True(t, initHistoryCalled, "initHistory should be called")
	assert.True(t, fetchCalled, "FetchTickers should be called")

	// Test with liquidations import only
	ts.importer.importTicks = false
	ts.importer.importLiquidations = true

	// Setup liquidation subscription
	liquidationCalled := false
	liqChan := make(chan exchanges.Liquidation)
	errChan := make(chan error)
	ts.exchange.SubscribeLiquidationsFunc = func(ctx context.Context) (<-chan exchanges.Liquidation, <-chan error) {
		liquidationCalled = true
		return liqChan, errChan
	}

	// Run with cancellable context
	ctx, cancel = context.WithCancel(context.Background())
	go func() {
		// Wait for liquidation import to start
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err = ts.importer.Run(ctx)
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.True(t, liquidationCalled, "SubscribeLiquidations should be called")

	// Test with both imports enabled
	ts.importer.importTicks = true
	ts.importer.importLiquidations = true

	// Reset tracking variables
	initHistoryCalled = false
	fetchCalled = false
	liquidationCalled = false

	// Run with cancellable context
	ctx, cancel = context.WithCancel(context.Background())
	go func() {
		// Wait longer to ensure ticker import starts after the sleep period
		time.Sleep(2 * time.Second)
		cancel()
	}()

	err = ts.importer.Run(ctx)
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.True(t, initHistoryCalled, "initHistory should be called")
	assert.True(t, fetchCalled, "FetchTickers should be called")
	assert.True(t, liquidationCalled, "SubscribeLiquidations should be called")
}

func TestWithNotifier(t *testing.T) {
	// Create mock notifier service
	mockNotifier := &importerMocks.NotifierServiceMock{
		SubscribeFunc: func(topic string, client notify.Client, strategy notify.Strategy) {
			// Verify subscribe parameters
			assert.Equal(t, "test_topic", topic)
			assert.NotNil(t, client)
			assert.NotNil(t, strategy)
		},
	}

	// Create importer with mock notifier
	importer := &Importer{
		notifier: mockNotifier,
		logger:   zap.NewNop(),
	}

	// Create mock client and strategy
	mockClient := &notifyMock.ClientMock{}
	mockStrategy := &notifyMock.StrategyMock{}

	// Call WithNotifier
	err := importer.WithNotifier(mockClient, "test_topic", mockStrategy)
	assert.NoError(t, err)

	// Verify notifier.Subscribe was called
	assert.Equal(t, 1, len(mockNotifier.SubscribeCalls()))
	call := mockNotifier.SubscribeCalls()[0]
	assert.Equal(t, "test_topic", call.Topic)
	assert.Equal(t, mockClient, call.Client)
	assert.Equal(t, mockStrategy, call.Strategy)
}
