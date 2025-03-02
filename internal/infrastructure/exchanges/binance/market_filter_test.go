package binance

import (
	"encoding/json"
	"sync"
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

func TestIsSymbolAllowed(t *testing.T) {
	tests := []struct {
		name     string
		symbol   string
		expected bool
	}{
		{
			name:     "known symbol BTCUSDT",
			symbol:   "BTCUSDT",
			expected: true,
		},
		{
			name:     "known symbol ETHUSDT",
			symbol:   "ETHUSDT",
			expected: true,
		},
		{
			name:     "known symbol with low cap DOGEUSDT",
			symbol:   "DOGEUSDT",
			expected: true,
		},
		{
			name:     "unknown symbol",
			symbol:   "UNKNOWN123",
			expected: false,
		},
		{
			name:     "empty symbol",
			symbol:   "",
			expected: false,
		},
		{
			name:     "case sensitive check",
			symbol:   "btcusdt", // lowercase
			expected: false,     // should be false as map keys are case-sensitive
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSymbolAllowed(tt.symbol)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadAllowedSymbols(t *testing.T) {
	// Reset the static variables to ensure consistent test behavior
	cachedAllowedSymbolsMap = nil
	once = sync.Once{}
	parseErr = nil

	// First call should parse the JSON and populate the cache
	symbolsMap, err := loadAllowedSymbols()
	require.NoError(t, err)
	require.NotNil(t, symbolsMap)

	// Verify a few known symbols from the market data
	assert.Contains(t, symbolsMap, "BTCUSDT")
	assert.Contains(t, symbolsMap, "ETHUSDT")
	assert.Contains(t, symbolsMap, "SOLUSDT")

	// Verify content of a known symbol
	btcInfo, exists := symbolsMap["BTCUSDT"]
	assert.True(t, exists)
	assert.NotZero(t, btcInfo.Volume24h)
	assert.NotZero(t, btcInfo.MarketCap)

	// Count the total number of symbols to ensure proper parsing
	expectedMinimumSymbols := 100 // We know there are more than 100 symbols in the marketDataJSON
	assert.GreaterOrEqual(t, len(symbolsMap), expectedMinimumSymbols)
}

func TestSingletonBehavior(t *testing.T) {
	// Reset the static variables to ensure consistent test behavior
	cachedAllowedSymbolsMap = nil
	once = sync.Once{}
	parseErr = nil

	// Create a counter to track the number of times the initialization occurs
	initCount := 0

	// Create a custom implementation to track function calls
	customInit := func() {
		// This is the contents of once.Do
		initCount++
		cachedAllowedSymbolsMap = make(AllowedSymbolsMap)
		parseErr = json.Unmarshal([]byte(marketDataJSON), &cachedAllowedSymbolsMap)
	}

	// Run multiple calls to loadAllowedSymbols by reimplementing its core logic
	for i := 0; i < 5; i++ {
		// This mimics what loadAllowedSymbols does, but with our custom init function
		once.Do(customInit)
	}

	// Verify the customInit function was called exactly once
	assert.Equal(t, 1, initCount, "Init function should be called exactly once")

	// Verify the map was populated
	assert.NotNil(t, cachedAllowedSymbolsMap)
	assert.Greater(t, len(cachedAllowedSymbolsMap), 0)
}

func TestConcurrentLoadAllowedSymbols(t *testing.T) {
	// Reset the static variables to ensure consistent test behavior
	cachedAllowedSymbolsMap = nil
	once = sync.Once{}
	parseErr = nil

	// Run multiple goroutines to test concurrent access
	const numGoroutines = 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			symbolsMap, err := loadAllowedSymbols()
			assert.NoError(t, err)
			assert.NotNil(t, symbolsMap)
			assert.Contains(t, symbolsMap, "BTCUSDT")
		}()
	}

	wg.Wait()

	// After all goroutines complete, check that the map was initialized
	assert.NotNil(t, cachedAllowedSymbolsMap)
	assert.Contains(t, cachedAllowedSymbolsMap, "BTCUSDT")
}
