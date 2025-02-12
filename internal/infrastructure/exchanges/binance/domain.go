package binance

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges"
)

const (
	// FuturesAPIURL is the base URL for the Binance Futures API
	FuturesAPIURL = "https://fapi.binance.com/fapi/v1"

	// FuturesWSUrl is the base URL for the Binance Futures Websocket API
	FuturesWSUrl = "wss://fstream.binance.com/ws/!forceOrder@arr"

	// FetchTickersData is the endpoint to fetch tickers data
	FetchTickersData = "/ticker/bookTicker"
)

// TickerDTO represents a ticker event from the Binance WebSocket API
type TickerDTO struct {
	Symbol      string `json:"symbol"`
	BidPrice    string `json:"bidPrice"`
	BidQuantity string `json:"bidQty"`
	AskPrice    string `json:"askPrice"`
	AskQuantity string `json:"askQty"`
	Time        int64  `json:"time"`
	LastUpdated int64  `json:"lastUpdateId"`
}

// toTicker converts a TickerDTO to an exchanges.Ticker
func (bt TickerDTO) toTicker() (exchanges.Ticker, error) {
	ticker := exchanges.Ticker{}

	// Validate and convert the string values to float64
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
	ticker.EventAt = time.Unix(0, bt.Time*int64(time.Millisecond))

	return ticker, nil
}

// LiquidationDTO represents a liquidation event from the Binance WebSocket API
type LiquidationDTO struct {
	EventType string `json:"e"`
	EventTime int64  `json:"E"`
	OrderData struct {
		Symbol       string `json:"s"`
		Side         string `json:"S"`
		OrderType    string `json:"o"`
		TimeInForce  string `json:"f"`
		OrigQuantity string `json:"q"`
		Price        string `json:"p"`
		AveragePrice string `json:"ap"`
		OrderStatus  string `json:"X"`
		LastQuantity string `json:"l"`
		Time         int64  `json:"T"`
	} `json:"o"`
}

// toLiquidation converts a LiquidationDTO to an exchanges.Liquidation
func (bl LiquidationDTO) toLiquidation() (exchanges.Liquidation, error) {
	liquidation := exchanges.Liquidation{}

	priceF, err := strconv.ParseFloat(bl.OrderData.Price, 64)
	if err != nil {
		return liquidation, fmt.Errorf("invalid price '%s': %w", bl.OrderData.Price, err)
	}
	quantityF, err := strconv.ParseFloat(bl.OrderData.OrigQuantity, 64)
	if err != nil {
		return liquidation, fmt.Errorf("invalid quantity '%s': %w", bl.OrderData.OrigQuantity, err)
	}

	liquidation.Price = priceF
	liquidation.Quantity = quantityF
	liquidation.Symbol = bl.OrderData.Symbol
	liquidation.EventAt = time.Unix(0, bl.EventTime*int64(time.Millisecond))
	liquidation.Side = bl.OrderData.Side
	liquidation.TotalPrice = priceF * quantityF

	return liquidation, nil
}
