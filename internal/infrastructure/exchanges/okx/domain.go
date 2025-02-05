package okx

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges"
)

// TickerResponse represents the API response for ticker data
type TickerResponse struct {
	Code string      `json:"code"`
	Msg  string      `json:"msg"`
	Data []TickerDTO `json:"data"`
}

// TickerDTO represents a ticker from the OKX API
type TickerDTO struct {
	InstID      string `json:"instId"`
	LastPrice   string `json:"last"`
	BidPrice    string `json:"bidPx"`
	BidQuantity string `json:"bidSz"`
	AskPrice    string `json:"askPx"`
	AskQuantity string `json:"askSz"`
	Timestamp   string `json:"ts"`
}

// toTicker converts a TickerDTO to an exchanges.Ticker
func (ot TickerDTO) toTicker() (exchanges.Ticker, error) {
	ticker := exchanges.Ticker{}

	bidPrice, err := strconv.ParseFloat(ot.BidPrice, 64)
	if err != nil {
		return ticker, fmt.Errorf("invalid bidPrice '%s': %w", ot.BidPrice, err)
	}
	askPrice, err := strconv.ParseFloat(ot.AskPrice, 64)
	if err != nil {
		return ticker, fmt.Errorf("invalid askPrice '%s': %w", ot.AskPrice, err)
	}
	bidQuantity, err := strconv.ParseFloat(ot.BidQuantity, 64)
	if err != nil {
		return ticker, fmt.Errorf("invalid bidQuantity '%s': %w", ot.BidQuantity, err)
	}
	askQuantity, err := strconv.ParseFloat(ot.AskQuantity, 64)
	if err != nil {
		return ticker, fmt.Errorf("invalid askQuantity '%s': %w", ot.AskQuantity, err)
	}
	ts, err := strconv.ParseInt(ot.Timestamp, 10, 64)
	if err != nil {
		return ticker, fmt.Errorf("invalid timestamp '%s': %w", ot.Timestamp, err)
	}

	ticker.Symbol = ot.InstID
	ticker.BidPrice = bidPrice
	ticker.AskPrice = askPrice
	ticker.BidQuantity = bidQuantity
	ticker.AskQuantity = askQuantity
	ticker.EventAt = time.Unix(0, ts*int64(time.Millisecond))

	return ticker, nil
}

// LiquidationEvent represents a liquidation websocket event
type LiquidationEvent struct {
	Arg struct {
		Channel  string `json:"channel"`
		InstType string `json:"instType"`
	} `json:"arg"`
	Data []LiquidationDTO `json:"data"`
}

// LiquidationDTO represents a liquidation order from OKX
type LiquidationDTO struct {
	Details []struct {
		Side      string `json:"side"` // "buy" or "sell"
		Quantity  string `json:"sz"`   // Size
		Timestamp string `json:"ts"`   // Timestamp in milliseconds
		Price     string `json:"bkPx"` // Price
	} `json:"details"`
	InstID string `json:"instId"`
}

// toLiquidation converts a LiquidationDTO to an exchanges.Liquidation
func (ol LiquidationDTO) toLiquidation() (exchanges.Liquidation, error) {
	if len(ol.Details) > 1 {
		fmt.Println(ol)
	}
	liquidation := exchanges.Liquidation{}

	price, err := strconv.ParseFloat(ol.Details[0].Price, 64)
	if err != nil {
		return liquidation, fmt.Errorf("invalid price '%s': %w", ol.Details[0].Price, err)
	}
	quantity, err := strconv.ParseFloat(ol.Details[0].Quantity, 64)
	if err != nil {
		return liquidation, fmt.Errorf("invalid quantity '%s': %w", ol.Details[0].Quantity, err)
	}
	ts, err := strconv.ParseInt(ol.Details[0].Timestamp, 10, 64)
	if err != nil {
		return liquidation, fmt.Errorf("invalid timestamp '%s': %w", ol.Details[0].Timestamp, err)
	}

	liquidation.Price = price
	liquidation.Quantity = quantity
	liquidation.Symbol = ol.InstID
	liquidation.EventAt = time.Unix(0, ts*int64(time.Millisecond))

	// Convert OKX-specific side to normalized format
	switch strings.ToLower(ol.Details[0].Side) {
	case "buy":
		liquidation.Side = "BUY"
	case "sell":
		liquidation.Side = "SELL"
	default:
		return liquidation, fmt.Errorf("invalid side '%s'", ol.Details[0].Side)
	}

	return liquidation, nil
}
