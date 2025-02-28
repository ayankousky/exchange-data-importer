// cmd/importer/main_test.go
package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/bootstrap"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockExchange implements Exchange interface for testing
type mockExchange struct {
	exchanges.Exchange
	name string
}

func (m *mockExchange) GetName() string {
	return m.name
}

func (m *mockExchange) Start(ctx context.Context) error {
	return nil
}

func (m *mockExchange) Stop() {}

func (m *mockExchange) FetchTickers(ctx context.Context) ([]exchanges.Ticker, error) {
	return []exchanges.Ticker{
		{
			Symbol:   "BTCUSDT",
			AskPrice: 50000,
			BidPrice: 49900,
			EventAt:  time.Now(),
		},
	}, nil
}

func (m *mockExchange) SubscribeLiquidations(ctx context.Context) (<-chan exchanges.Liquidation, <-chan error) {
	liqChan := make(chan exchanges.Liquidation)
	errChan := make(chan error)
	return liqChan, errChan
}

func TestMain(m *testing.M) {
	os.Args = []string{os.Args[0]}
	os.Exit(m.Run())
}

func TestMainApplicationFlow(t *testing.T) {
	tests := []struct {
		name     string
		setup    func()
		validate func(t *testing.T, err error)
	}{
		{
			name: "successful_startup_with_memory_repo",
			setup: func() {
				os.Setenv("ENV", "test")
				os.Setenv("SERVICE_NAME", "test-service")
				os.Setenv("EXCHANGE_BINANCE_ENABLED", "true")
				// Using memory repository by default
			},
			validate: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "fail_without_exchange",
			setup: func() {
				os.Setenv("ENV", "test")
				os.Setenv("SERVICE_NAME", "test-service")
				os.Setenv("EXCHANGE_BINANCE_ENABLED", "false")
				os.Setenv("EXCHANGE_BYBIT_ENABLED", "false")
				os.Setenv("EXCHANGE_OKX_ENABLED", "false")
			},
			validate: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "no exchange configured")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment before each test
			os.Clearenv()

			// Setup test environment
			tt.setup()

			// Create shorter timeout for tests
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			// Build the application
			app, err := bootstrap.NewBuilder().
				WithLogger(ctx).
				WithExchange(ctx).
				WithRepository(ctx).
				WithNotifiers(ctx).
				Build()

			tt.validate(t, err)

			if err == nil {
				require.NotNil(t, app)

				// Start the application (don't wait for completion)
				go func() {
					startErr := app.Start(ctx)
					if startErr != nil {
						t.Logf("Start error: %v", startErr)
					}
				}()

				// Let it run briefly
				time.Sleep(1000 * time.Millisecond)

				// Cancel context to trigger shutdown
				cancel()

				// Wait briefly to allow for cleanup
				time.Sleep(100 * time.Millisecond)
			}
		})
	}
}
