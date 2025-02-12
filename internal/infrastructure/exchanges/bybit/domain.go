package bybit

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges"
)

const (
	// FuturesAPIURL is the base URL for the Bybit Futures API
	FuturesAPIURL = "https://api.bybit.com/v5"

	// FuturesWSUrl is the base URL for the Bybit Futures Websocket API
	FuturesWSUrl = "wss://stream.bybit.com/v5/public/linear"

	// FetchTickersData is the endpoint to fetch tickers data
	FetchTickersData = "/market/tickers?category=linear"
)

// TickerResponse represents the API response for ticker data
type TickerResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		Category string      `json:"category"`
		List     []TickerDTO `json:"list"`
	} `json:"result"`
	Time int64 `json:"time"`
}

// TickerDTO represents a ticker from the Bybit API
type TickerDTO struct {
	Symbol      string `json:"symbol"`
	BidPrice    string `json:"bid1Price"`
	BidQuantity string `json:"bid1Size"`
	AskPrice    string `json:"ask1Price"`
	AskQuantity string `json:"ask1Size"`
	LastPrice   string `json:"lastPrice"`
}

// toTicker converts a TickerDTO to an exchanges.Ticker
func (bt TickerDTO) toTicker() (exchanges.Ticker, error) {
	ticker := exchanges.Ticker{}

	bidPrice, err := strconv.ParseFloat(bt.BidPrice, 64)
	if err != nil {
		return ticker, fmt.Errorf("invalid bidPrice '%s': %w", bt.BidPrice, err)
	}
	askPrice, err := strconv.ParseFloat(bt.AskPrice, 64)
	if err != nil {
		return ticker, fmt.Errorf("invalid askPrice '%s': %w", bt.AskPrice, err)
	}
	bidQuantity, err := strconv.ParseFloat(bt.BidQuantity, 64)
	if err != nil {
		return ticker, fmt.Errorf("invalid bidQuantity '%s': %w", bt.BidQuantity, err)
	}
	askQuantity, err := strconv.ParseFloat(bt.AskQuantity, 64)
	if err != nil {
		return ticker, fmt.Errorf("invalid askQuantity '%s': %w", bt.AskQuantity, err)
	}

	ticker.Symbol = bt.Symbol
	ticker.BidPrice = bidPrice
	ticker.AskPrice = askPrice
	ticker.BidQuantity = bidQuantity
	ticker.AskQuantity = askQuantity

	return ticker, nil
}

// LiquidationEvent represents a liquidation websocket event
type LiquidationEvent struct {
	Topic string         `json:"topic"`
	Type  string         `json:"type"`
	Data  LiquidationDTO `json:"data"`
	TS    int64          `json:"ts"`
}

// LiquidationDTO represents a liquidation order from Bybit
type LiquidationDTO struct {
	Symbol      string `json:"symbol"`
	Side        string `json:"side"`
	Price       string `json:"price"`
	Quantity    string `json:"size"`
	UpdatedTime int64  `json:"updatedTime"`
}

// toLiquidation converts a LiquidationDTO to an exchanges.Liquidation
func (bl LiquidationDTO) toLiquidation() (exchanges.Liquidation, error) {
	liquidation := exchanges.Liquidation{}

	price, err := strconv.ParseFloat(bl.Price, 64)
	if err != nil {
		return liquidation, fmt.Errorf("invalid price '%s': %w", bl.Price, err)
	}
	quantity, err := strconv.ParseFloat(bl.Quantity, 64)
	if err != nil {
		return liquidation, fmt.Errorf("invalid quantity '%s': %w", bl.Quantity, err)
	}

	liquidation.Price = price
	liquidation.Quantity = quantity
	liquidation.Symbol = bl.Symbol
	liquidation.EventAt = time.Unix(0, bl.UpdatedTime*int64(time.Millisecond))
	liquidation.TotalPrice = price * quantity
	switch bl.Side {
	case "Buy":
		liquidation.Side = "SELL"
	case "Sell":
		liquidation.Side = "BUY"
	default:
		return liquidation, fmt.Errorf("invalid side '%s'", bl.Side)

	}

	return liquidation, nil
}
