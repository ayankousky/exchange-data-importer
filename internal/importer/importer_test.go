package importer

import (
	"context"
	"fmt"
	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

// Mock implementations

type MockExchange struct {
	mock.Mock
}

func (m *MockExchange) FetchTickers(ctx context.Context) ([]exchanges.Ticker, error) {
	args := m.Called(ctx)
	return args.Get(0).([]exchanges.Ticker), args.Error(1)
}

func (m *MockExchange) GetName() string {
	args := m.Called()
	return args.String(0)
}

type MockTickRepository struct {
	mock.Mock
}

func (m *MockTickRepository) Create(ctx context.Context, tick *domain.Tick) error {
	args := m.Called(ctx, tick)
	return args.Error(0)
}

func (m *MockTickRepository) GetHistorySince(ctx context.Context, since time.Time) ([]*domain.Tick, error) {
	args := m.Called(ctx, since)
	return args.Get(0).([]*domain.Tick), args.Error(1)
}

type MockLiquidationRepository struct {
	mock.Mock
}

func (m *MockLiquidationRepository) Create(ctx context.Context, liquidation *domain.Liquidation) error {
	args := m.Called(ctx, liquidation)
	return args.Error(0)
}

type MockRepositoryFactory struct {
	mock.Mock
}

func (m *MockRepositoryFactory) GetTickRepository(name string) domain.TickRepository {
	args := m.Called(name)
	return args.Get(0).(domain.TickRepository)
}

func (m *MockRepositoryFactory) GetLiquidationRepository(name string) domain.LiquidationRepository {
	args := m.Called(name)
	return args.Get(0).(domain.LiquidationRepository)
}

// Tests
func TestStartImport(t *testing.T) {
	exchange := new(MockExchange)
	exchange.On("GetName").Return("mockExchange")
	exchange.On("FetchTickers", mock.Anything).Return([]exchanges.Ticker{
		{
			Symbol:   "BTC",
			AskPrice: 50000,
			BidPrice: 49950,
		},
	}, nil)
	tickRepository := new(MockTickRepository)
	tickRepository.On("Create", mock.Anything, mock.Anything).Return(nil)
	liquidationRepository := new(MockLiquidationRepository)
	liquidationRepository.On("Create", mock.Anything, mock.Anything).Return(nil)
	repoFactory := new(MockRepositoryFactory)
	repoFactory.On("GetTickRepository", "mockExchange").Return(tickRepository)
	repoFactory.On("GetLiquidationRepository", "mockExchange").Return(liquidationRepository)

	imp := NewImporter(exchange, repoFactory)

	err := imp.importTickers()

	assert.NoError(t, err)
	exchange.AssertExpectations(t)
	tickRepository.AssertExpectations(t)
}

func TestAddTickHistory(t *testing.T) {
	imp := &Importer{
		tickHistory: make([]*domain.Tick, 0),
	}

	for i := 0; i < MaxTickHistory-5; i++ {
		tick := &domain.Tick{ID: fmt.Sprintf("tick-%d", i)}
		imp.addTickHistory(tick)
	}
	assert.Equal(t, MaxTickHistory-5, len(imp.getTickHistory()), "Every tick should be added to the history")

	for i := 0; i < MaxTickHistory; i++ {
		tick := &domain.Tick{ID: fmt.Sprintf("tick-%d", i)}
		imp.addTickHistory(tick)
	}
	assert.Equal(t, MaxTickHistory, len(imp.getTickHistory()), "Tick history should be limited")
}

func TestAddTickerHistory(t *testing.T) {
	imp := &Importer{
		tickerHistory: make(map[domain.TickerName][]*domain.Ticker),
	}

	startDate := time.Now().Truncate(time.Hour)
	for i := 0; i < 1100; i++ {
		ticker := domain.Ticker{
			Symbol: "BTC",
			Ask:    50000,
			Bid:    49950,
			Date:   startDate.Add(time.Second * time.Duration(i)),
		}
		imp.addTickerHistory(&ticker)
	}
	assert.Equal(t, 19, len(imp.getTickerHistory("BTC")), "Only 1 ticker per minute should be stored")
	for i := 0; i < (60+10)*MaxTickHistory; i++ {
		ticker := domain.Ticker{
			Symbol: "BTC",
			Ask:    50000,
			Bid:    49950,
			Date:   startDate.Add(time.Second * time.Duration(i)),
		}
		imp.addTickerHistory(&ticker)
	}
	assert.Equal(t, MaxTickHistory-1, len(imp.getTickerHistory("BTC")), "Ticker history should be limited")
}

func TestStartImportEverySecond(t *testing.T) {
	// This test simulates the method behavior for a limited duration
	// and verifies that importTickers is called in a loop.
	exchange := new(MockExchange)
	tickRepository := new(MockTickRepository)
	liqudationRepository := new(MockLiquidationRepository)
	repoFactory := new(MockRepositoryFactory)

	exchange.On("GetName").Return("mockExchange")
	repoFactory.On("GetTickRepository", "mockExchange").Return(tickRepository)
	repoFactory.On("GetLiquidationRepository", "mockExchange").Return(liqudationRepository)
	exchange.On("FetchTickers", mock.Anything).Return([]exchanges.Ticker{}, nil)
	tickRepository.On("Create", mock.Anything, mock.Anything).Return(nil)
	tickRepository.On("GetHistorySince", mock.Anything, mock.Anything).Return([]*domain.Tick{}, nil)

	imp := NewImporter(exchange, repoFactory)

	// Run the method in a goroutine for a short time
	_, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		imp.StartImportEverySecond()
	}()

	// Allow the loop to run a couple of iterations
	time.Sleep(3 * time.Second)

	// Cancel the loop
	cancel()
	exchange.AssertExpectations(t)
	tickRepository.AssertExpectations(t)
}

func TestCorruptedData(t *testing.T) {
	imp := &Importer{
		tickerHistory: make(map[domain.TickerName][]*domain.Ticker),
	}

	startDate := time.Now().Truncate(time.Hour)
	for i := 0; i < 1500; i++ {
		ticker := domain.Ticker{
			Symbol: "BTC",
			Ask:    50000,
			Bid:    49950,
			Date:   startDate.Add(time.Second * time.Duration(i)),
		}
		imp.addTickerHistory(&ticker)
	}

	history := imp.getTickerHistory("BTC")
	assert.Equal(t, 24, len(history))
}
