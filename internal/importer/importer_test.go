package importer

import (
	"context"
	"testing"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
	domainMock "github.com/ayankousky/exchange-data-importer/internal/domain/mock"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges"
	exchangeMock "github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges/mock"
	repoMock "github.com/ayankousky/exchange-data-importer/internal/repository/mock"
	"github.com/stretchr/testify/assert"
)

// Tests
func TestStartImport(t *testing.T) {
	ctx := context.Background()
	repoFactoryMocked := repoMock.NewFactoryMock()
	exchangeMocked := exchangeMock.NewMockClient("mockExchange")
	importer := NewImporter(exchangeMocked, repoFactoryMocked)

	tickers, err := importer.fetchTickers(ctx)
	assert.Equal(t, 2, len(tickers))
	assert.NoError(t, err)

	err = importer.importTick(ctx)

	assert.NoError(t, err)
}

func TestTickerHistory(t *testing.T) {
	testItemsCount := 1000
	repoFactoryMocked := repoMock.NewFactoryMock()
	exchangeMocked := exchangeMock.NewMockClient("mockExchange")
	exchangeMocked.GenerateData(testItemsCount)
	importer := NewImporter(exchangeMocked, repoFactoryMocked)

	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 1000; i++ {
		ticker := &domain.Ticker{
			Symbol:    "BTCUSDT",
			Ask:       50000 + float64(i),
			Bid:       49950 + float64(i),
			CreatedAt: startDate.Add(time.Second * time.Duration(i)),
		}
		importer.addTickerHistory(ticker)
	}

	lastTicker, _ := importer.getLastTicker("BTCUSDT")
	tickerHistory := importer.getTickerHistory("BTCUSDT")
	lastItem, _ := tickerHistory.Last()
	assert.Equal(t, 17, tickerHistory.Len(), "Only 1 ticker per minute should be stored")
	assert.Equal(t, 39, lastItem.CreatedAt.Second(), "Last inserted ticker should be at the 39th second")
	assert.Equal(t, 39, lastTicker.CreatedAt.Second(), "getLastTicker should return the last inserted ticker")
	assert.Equal(t, 59, tickerHistory.At(tickerHistory.Len()-2).CreatedAt.Second(), "Last second inserted ticker should be at the 39th second")
	assert.Equal(t, 59, tickerHistory.At(tickerHistory.Len()-3).CreatedAt.Second(), "Last third inserted ticker should be at the 39th second")
	for i := 0; i < (60+10)*domain.MaxTickHistory; i++ {
		ticker := &domain.Ticker{
			Symbol:    "BTCUSDT",
			Ask:       50000 + float64(i),
			Bid:       49950 + float64(i),
			CreatedAt: startDate.Add(time.Second * time.Duration(i)),
		}
		importer.addTickerHistory(ticker)
	}
	assert.Equal(t, domain.MaxTickHistory, importer.getTickerHistory("BTCUSDT").Len(), "Ticker history should be limited")
}

func TestCorruptedData(t *testing.T) {
	repoFactoryMocked := repoMock.NewFactoryMock()
	exchangeMocked := exchangeMock.NewMockClient("mockExchange")
	importer := NewImporter(exchangeMocked, repoFactoryMocked)

	startDate := time.Now().Truncate(time.Hour)
	for i := 0; i < 1500; i++ {
		ticker := &domain.Ticker{
			Symbol:    "BTCUSDT",
			Ask:       50000,
			Bid:       49950,
			CreatedAt: startDate.Add(time.Second),
		}
		importer.addTickerHistory(ticker)
	}

	history := importer.getTickerHistory("BTCUSDT")
	assert.Equal(t, 1, history.Len(), "Only 1 ticker per minute should be stored")
}

func TestInitHistory(t *testing.T) {
	ctx := context.Background()
	repoFactoryMocked := repoMock.NewFactoryMock()
	exchangeMocked := exchangeMock.NewMockClient("mockExchange")
	importer := NewImporter(exchangeMocked, repoFactoryMocked)

	for _, tick := range domainMock.GenerateTicks(1000) {
		importer.tickRepository.Create(nil, tick)
	}

	importer.initHistory(ctx)
	assert.Equal(t, domain.MaxTickHistory, importer.tickHistory.Len())
	assert.Equal(t, 17, importer.getTickerHistory("BTCUSDT").Len())
	lastTick, exists := importer.tickHistory.Last()
	btcHistory := importer.getTickerHistory("BTCUSDT")
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
			ctx := context.Background()
			repoFactory := repoMock.NewFactoryMock()
			exchange := exchangeMock.NewMockClient("mockExchange")
			importer := NewImporter(exchange, repoFactory)

			tick := &domain.Tick{
				StartAt: time.Now(),
				Data:    make(map[domain.TickerName]*domain.Ticker),
			}

			importer.buildTick(ctx, tick, tt.tickers)

			assert.Len(t, tick.Data, tt.expectedTickersLen)
			assert.Equal(t, tt.expectedLL60, tick.LL60)
			assert.Equal(t, tt.expectedSL10, tick.SL10)
		})
	}
}
