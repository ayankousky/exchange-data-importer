package binance

import (
	"encoding/json"
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

// FilterTickers filters tickers based on allowed symbols
func FilterTickers(tickers []TickerDTO) ([]TickerDTO, error) {
	var allowedSymbolsMap AllowedSymbolsMap
	if err := json.Unmarshal([]byte(marketDataJSON), &allowedSymbolsMap); err != nil {
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
