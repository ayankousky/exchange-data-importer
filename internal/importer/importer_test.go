package importer

import (
	"context"
	"testing"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
	domainMocks "github.com/ayankousky/exchange-data-importer/internal/domain/mocks"
	importerMocks "github.com/ayankousky/exchange-data-importer/internal/importer/mocks"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges"
	exchangeMocks "github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges/mocks"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/notify"
	notifyMock "github.com/ayankousky/exchange-data-importer/internal/infrastructure/notify/mocks"
	"github.com/ayankousky/exchange-data-importer/pkg/utils/mathutils"
	"github.com/stretchr/testify/assert"
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
		GetTickRepositoryFunc: func(name string) domain.TickRepository {
			return tickRepo
		},
		GetLiquidationRepositoryFunc: func(name string) domain.LiquidationRepository {
			return liqRepo
		},
	}

	return &testSuite{
		exchange:    exchange,
		repoFactory: repoFactory,
		tickRepo:    tickRepo,
		liqRepo:     liqRepo,
		importer:    NewImporter(exchange, repoFactory),
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

	lastTicker, _ := ts.importer.getLastTicker("BTCUSDT")
	tickerHistory := ts.importer.getTickerHistory("BTCUSDT")
	lastItem, _ := tickerHistory.Last()

	assert.Equal(t, 17, tickerHistory.Len(), "Only 1 ticker per minute should be stored")
	assert.Equal(t, 39, lastItem.CreatedAt.Second(), "Last inserted ticker should be at the 39th second")
	assert.Equal(t, 39, lastTicker.CreatedAt.Second(), "getLastTicker should return the last inserted ticker")
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
	assert.Equal(t, domain.MaxTickHistory, ts.importer.getTickerHistory("BTCUSDT").Len(), "Ticker history should be limited")
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

	history := ts.importer.getTickerHistory("BTCUSDT")
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

	ts.importer.initHistory(ctx)

	assert.Equal(t, domain.MaxTickHistory, ts.importer.tickHistory.Len())
	assert.Equal(t, 17, ts.importer.getTickerHistory("BTCUSDT").Len())

	lastTick, exists := ts.importer.tickHistory.Last()
	btcHistory := ts.importer.getTickerHistory("BTCUSDT")
	assert.True(t, exists)
	assert.Equal(t, 625810.2565, lastTick.Data["BTCUSDT"].Ask)
	assert.Equal(t, 625810.2565, btcHistory.At(btcHistory.Len()-1).Ask)
	assert.Equal(t, 604932.5165, btcHistory.At(btcHistory.Len()-2).Ask)
	assert.Equal(t, 573615.9065, btcHistory.At(btcHistory.Len()-3).Ask)
	assert.Equal(t, lastTick.Data["BTCUSDT"].Ask, btcHistory.At(btcHistory.Len()-1).Ask)
}

func TestBuildTick(t *testing.T) {
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
				{Symbol: "BTCUSDT", AskPrice: 50000, BidPrice: 49900},
				{Symbol: "ETHUSDT", AskPrice: 3000, BidPrice: 2990},
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
			wantEventType: domain.EventTypeTicker,
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
			wantEventType: domain.EventTypeTicker,
			wantCalls:     3, // One call per notifier
		},
		{
			name: "should handle empty tick data",
			tick: &domain.Tick{
				Data: map[domain.TickerName]*domain.Ticker{},
			},
			notifierCount: 1,
			wantEventType: domain.EventTypeTicker,
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
				notifier := &notifyMock.ClientMock{
					SendFunc: func(ctx context.Context, event notify.Event) error {
						// Verify event properties
						assert.Equal(t, tt.wantEventType, event.EventType)
						assert.NotZero(t, event.Time)
						assert.NotNil(t, event.Data)
						return nil
					},
				}
				notifiers[i] = notifier
				ts.importer.WithMarketNotify(notifier)
			}

			// Execute the notification
			ts.importer.notifyNewTick(tt.tick)

			// Verify the notifications
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
