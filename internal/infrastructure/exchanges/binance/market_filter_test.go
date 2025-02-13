package binance

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterTickers(t *testing.T) {
	tests := []struct {
		name            string
		tickers         []TickerDTO
		expectedCount   int
		expectedSymbols []string
		expectError     bool
	}{
		{
			name: "all valid tickers",
			tickers: []TickerDTO{
				{Symbol: "BTCUSDT"},
				{Symbol: "ETHUSDT"},
				{Symbol: "BNBUSDT"},
			},
			expectedCount:   3,
			expectedSymbols: []string{"BTCUSDT", "ETHUSDT", "BNBUSDT"},
			expectError:     false,
		},
		{
			name: "all invalid tickers",
			tickers: []TickerDTO{
				{Symbol: "INVALID1"},
				{Symbol: "INVALID2"},
			},
			expectedCount:   0,
			expectedSymbols: []string{},
			expectError:     false,
		},
		{
			name: "mix of valid and invalid tickers",
			tickers: []TickerDTO{
				{Symbol: "BTCUSDT"},
				{Symbol: "INVALID1"},
				{Symbol: "ETHUSDT"},
				{Symbol: "INVALID2"},
			},
			expectedCount:   2,
			expectedSymbols: []string{"BTCUSDT", "ETHUSDT"},
			expectError:     false,
		},
		{
			name:            "empty tickers list",
			tickers:         []TickerDTO{},
			expectedCount:   0,
			expectedSymbols: []string{},
			expectError:     false,
		},
		{
			name: "actual market data symbols",
			tickers: []TickerDTO{
				{Symbol: "DOGEUSDT"},
				{Symbol: "SOLUSDT"},
				{Symbol: "AVAXUSDT"},
			},
			expectedCount:   3,
			expectedSymbols: []string{"DOGEUSDT", "SOLUSDT", "AVAXUSDT"},
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FilterTickers(tt.tickers)

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, result, tt.expectedCount)

			// Verify all expected symbols are present
			resultSymbols := make([]string, len(result))
			for i, ticker := range result {
				resultSymbols[i] = ticker.Symbol
			}
			assert.ElementsMatch(t, tt.expectedSymbols, resultSymbols)
		})
	}
}
