package binance

import (
	"encoding/json"
	"sync"
)

// SymbolInfo represents the structure of market data for a single ticker
type SymbolInfo struct {
	Symbol    string  `json:"symbol"`
	CMCRank   int     `json:"cmc_rank"`
	Volume24h float64 `json:"volume_24h"`
	MarketCap float64 `json:"market_cap"`
}

// AllowedSymbolsMap represents a map of ticker symbols to their market data
type AllowedSymbolsMap map[string]SymbolInfo

var (
	// cachedAllowedSymbolsMap holds the parsed allowed symbols for reuse
	cachedAllowedSymbolsMap AllowedSymbolsMap
	// once ensures the allowed symbols map is only parsed once
	once sync.Once
	// parseErr stores any error from parsing the market data
	parseErr error
)

// loadAllowedSymbols loads the allowed symbols map from marketDataJSON
func loadAllowedSymbols() (AllowedSymbolsMap, error) {
	once.Do(func() {
		cachedAllowedSymbolsMap = make(AllowedSymbolsMap)
		parseErr = json.Unmarshal([]byte(marketDataJSON), &cachedAllowedSymbolsMap)
	})
	return cachedAllowedSymbolsMap, parseErr
}

// FilterTickers filters tickers based on allowed symbols
func FilterTickers(tickers []TickerDTO) ([]TickerDTO, error) {
	allowedSymbolsMap, err := loadAllowedSymbols()
	if err != nil {
		return nil, err
	}
	validTickers := make([]TickerDTO, 0, len(allowedSymbolsMap))

	for _, ticker := range tickers {
		if _, exists := allowedSymbolsMap[ticker.Symbol]; !exists {
			continue
		}
		validTickers = append(validTickers, ticker)
	}

	return validTickers, nil
}

// IsSymbolAllowed checks whether a given symbol is in the allowed symbols map
func IsSymbolAllowed(symbol string) bool {
	allowedSymbolsMap, _ := loadAllowedSymbols()

	_, exists := allowedSymbolsMap[symbol]
	return exists
}
