package importer

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
	domainMocks "github.com/ayankousky/exchange-data-importer/internal/domain/mocks"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges"
	"github.com/stretchr/testify/assert"
)

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

func TestRunLiquidationsImport(t *testing.T) {
	ts := setupTest()
	ctx, cancel := context.WithCancel(context.Background())

	// Set up test channels
	liquidationChan := make(chan exchanges.Liquidation)
	errorChan := make(chan error)

	// Update the mock to return our test channels
	ts.exchange.SubscribeLiquidationsFunc = func(ctx context.Context) (<-chan exchanges.Liquidation, <-chan error) {
		return liquidationChan, errorChan
	}

	// Replace the repository Create func to track calls
	ts.liqRepo.CreateFunc = func(ctx context.Context, l domain.Liquidation) error {
		return nil
	}

	// Start the import in a goroutine
	importDone := make(chan error)
	go func() {
		importDone <- ts.importer.runLiquidationsImport(ctx)
	}()

	// Test data
	testTime := time.Now()
	testLiquidation := exchanges.Liquidation{
		Symbol:     "BTCUSDT",
		Side:       "SELL",
		Price:      50000.0,
		Quantity:   1.5,
		TotalPrice: 75000.0,
		EventAt:    testTime,
	}

	// Send a test liquidation
	liquidationChan <- testLiquidation

	// Allow some time for processing
	time.Sleep(50 * time.Millisecond)

	// Send an error
	testError := fmt.Errorf("test error")
	errorChan <- testError

	// Allow some time for processing
	time.Sleep(50 * time.Millisecond)

	// Cancel context to stop import
	cancel()

	// Wait for import to finish
	err := <-importDone

	// Check that context cancellation was returned
	assert.ErrorIs(t, err, context.Canceled)

	// Verify liquidation was processed by checking repository calls
	calls := ts.liqRepo.CreateCalls()
	assert.Equal(t, 1, len(calls), "Expected one call to Create")

	if len(calls) > 0 {
		// Verify liquidation was correctly converted
		assert.Equal(t, domain.TickerName(testLiquidation.Symbol), calls[0].L.Order.Symbol)
		assert.Equal(t, domain.OrderSide(testLiquidation.Side), calls[0].L.Order.Side)
		assert.Equal(t, testLiquidation.Price, calls[0].L.Order.Price)
		assert.Equal(t, testLiquidation.Quantity, calls[0].L.Order.Quantity)
		assert.Equal(t, testLiquidation.TotalPrice, calls[0].L.Order.TotalPrice)
		assert.Equal(t, testLiquidation.EventAt, calls[0].L.EventAt)
	}
}

func TestRunLiquidationsImportErrors(t *testing.T) {
	tests := []struct {
		name                string
		setupSubscribe      func(ctx context.Context) (<-chan exchanges.Liquidation, <-chan error)
		expectImportFailure bool
	}{
		{
			name: "should handle nil liquidation channel",
			setupSubscribe: func(ctx context.Context) (<-chan exchanges.Liquidation, <-chan error) {
				return nil, make(chan error)
			},
			expectImportFailure: true,
		},
		{
			name: "should handle nil error channel",
			setupSubscribe: func(ctx context.Context) (<-chan exchanges.Liquidation, <-chan error) {
				return make(chan exchanges.Liquidation), nil
			},
			expectImportFailure: true,
		},
		{
			name: "should handle both channels nil",
			setupSubscribe: func(ctx context.Context) (<-chan exchanges.Liquidation, <-chan error) {
				return nil, nil
			},
			expectImportFailure: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := setupTest()

			// Set up mock
			ts.exchange.SubscribeLiquidationsFunc = tt.setupSubscribe

			// Run import
			err := ts.importer.runLiquidationsImport(context.Background())

			if tt.expectImportFailure {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProcessLiquidation(t *testing.T) {
	tests := []struct {
		name        string
		liquidation exchanges.Liquidation
		setupRepo   func(*domainMocks.LiquidationRepositoryMock)
		wantErr     bool
	}{
		{
			name: "should process valid liquidation",
			liquidation: exchanges.Liquidation{
				Symbol:     "BTCUSDT",
				Side:       "SELL",
				Price:      50000.0,
				Quantity:   1.5,
				TotalPrice: 75000.0,
				EventAt:    time.Now(),
			},
			setupRepo: func(repo *domainMocks.LiquidationRepositoryMock) {
				repo.CreateFunc = func(ctx context.Context, l domain.Liquidation) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "should handle repository error",
			liquidation: exchanges.Liquidation{
				Symbol:     "BTCUSDT",
				Side:       "SELL",
				Price:      50000.0,
				Quantity:   1.5,
				TotalPrice: 75000.0,
				EventAt:    time.Now(),
			},
			setupRepo: func(repo *domainMocks.LiquidationRepositoryMock) {
				repo.CreateFunc = func(ctx context.Context, l domain.Liquidation) error {
					return fmt.Errorf("database error")
				}
			},
			wantErr: true,
		},
		{
			name: "should fail validation for invalid liquidation",
			liquidation: exchanges.Liquidation{
				Symbol:     "BTCUSDT",
				Side:       "SELL",
				Price:      0.0, // Invalid price
				Quantity:   1.5,
				TotalPrice: 75000.0,
				EventAt:    time.Now(),
			},
			setupRepo: func(repo *domainMocks.LiquidationRepositoryMock) {
				repo.CreateFunc = func(ctx context.Context, l domain.Liquidation) error {
					return nil
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := setupTest()

			// Configure repository mock
			tt.setupRepo(ts.liqRepo)

			// Process liquidation
			err := ts.importer.processLiquidation(context.Background(), tt.liquidation)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify repository was called
				calls := ts.liqRepo.CreateCalls()
				assert.Equal(t, 1, len(calls))

				if len(calls) > 0 {
					// Verify liquidation was correctly converted
					assert.Equal(t, domain.TickerName(tt.liquidation.Symbol), calls[0].L.Order.Symbol)
					assert.Equal(t, domain.OrderSide(tt.liquidation.Side), calls[0].L.Order.Side)
					assert.Equal(t, tt.liquidation.Price, calls[0].L.Order.Price)
					assert.Equal(t, tt.liquidation.Quantity, calls[0].L.Order.Quantity)
					assert.Equal(t, tt.liquidation.TotalPrice, calls[0].L.Order.TotalPrice)
					assert.Equal(t, tt.liquidation.EventAt, calls[0].L.EventAt)
				}
			}
		})
	}
}
