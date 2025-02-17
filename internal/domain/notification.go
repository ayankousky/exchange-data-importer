package domain

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

const (
	// MarketDataTopic is the event type for ticker data
	MarketDataTopic = "TICKER"

	// AlertTopic is the event triggered when something significant happens in the market
	AlertTopic = "ALERT_MARKET_STATE"
)

// TopicLevel represents a notification topic
type TopicLevel string

// Validate checks if the topic exists
func (t TopicLevel) Validate() error {
	switch t {
	case MarketDataTopic, AlertTopic:
		return nil
	default:
		return fmt.Errorf("invalid topic: '%s'", t)
	}
}
func (t TopicLevel) String() string {
	return string(t)
}

// TickerNotification represents a ticker notification event without excessive data
type TickerNotification struct {
	Tick   Tick   `json:"tick"`
	Ticker Ticker `json:"ticker"`
}

// NewTickerNotification creates a new TickerNotification from a tick and a given ticker name
func NewTickerNotification(tick *Tick, symbol TickerName) (*TickerNotification, error) {
	if tick == nil {
		return nil, errors.New("tick cannot be nil")
	}

	tickCopy := *tick
	tickCopy.Data = nil

	if _, exists := tick.Data[symbol]; !exists {
		return nil, errors.New("ticker not found in tick data")
	}

	notification := &TickerNotification{
		Tick:   tickCopy,
		Ticker: *tick.Data[symbol],
	}

	return notification, nil
}

// TickAlertThresholds defines thresholds for generating market alerts
type TickAlertThresholds struct {
	AvgPrice1mChange    float64 // price change in 1 minute for the entire market
	AvgPrice20mChange   float64 // price change in 20 minutes for the entire market
	TickerPrice1mChange float64 // price change in 1 minute for a single ticker
}

// MarketAlert represents a formatted market alert message
type MarketAlert struct {
	Message string `json:"message"`
}

// FormatTickerAlert formats a single ticker's data into a readable message
func FormatTickerAlert(ticker *Ticker) string {
	parts := []string{
		fmt.Sprintf("<b>%s</b>", string(ticker.Symbol)),
		fmt.Sprintf("%.2f/%.2f", ticker.Ask, ticker.Bid),
	}

	if ticker.Change1m != 0 {
		change := fmt.Sprintf("1m: %.2f%%", ticker.Change1m)
		if ticker.Change1m > 0 {
			change = fmt.Sprintf("1m: +%.2f%%", ticker.Change1m)
		}
		parts = append(parts, change)
	}
	if ticker.Change20m != 0 {
		change := fmt.Sprintf("20m: %.2f%%", ticker.Change20m)
		if ticker.Change20m > 0 {
			change = fmt.Sprintf("20m: +%.2f%%", ticker.Change20m)
		}
		parts = append(parts, change)
	}
	if ticker.RSI20 != 0 {
		parts = append(parts, fmt.Sprintf("RSI: %.1f", ticker.RSI20))
	}

	return strings.Join(parts, " | ")
}

// FormatTickAlert formats a market tick into a readable message
func FormatTickAlert(tick *Tick, thresholds TickAlertThresholds) (string, bool) {
	if tick == nil {
		return "", false
	}

	var lines []string
	hasAlert := false

	if math.Abs(tick.Avg.Change1m) >= thresholds.AvgPrice1mChange {
		hasAlert = true
		sign := ""
		if tick.Avg.Change1m > 0 {
			sign = "+"
		}
		lines = append(lines, fmt.Sprintf("⚠️ <b>Significant Market Move</b>\nPrice Change 1m: %s%.2f%%", sign, tick.Avg.Change1m))
	}
	if math.Abs(tick.Avg.Change20m) >= thresholds.AvgPrice20mChange {
		hasAlert = true
		sign := ""
		if tick.Avg.Change20m > 0 {
			sign = "+"
		}
		lines = append(lines, fmt.Sprintf("Price Change 20m: %s%.2f%%", sign, tick.Avg.Change20m))
	}

	var significantTickers []string
	for _, ticker := range tick.Data {
		if math.Abs(ticker.Change1m) >= thresholds.TickerPrice1mChange {
			significantTickers = append(significantTickers, FormatTickerAlert(ticker))
			hasAlert = true
		}
	}
	if len(significantTickers) > 0 {
		movesSection := append([]string{"🔍 <b>Active Pairs:</b>"}, significantTickers...)
		lines = append(lines, strings.Join(movesSection, "\n"))
	}

	var liquidationInfo []string
	if tick.LL5 > 500 || tick.LL60 > 2000 || tick.SL10 > 30 {
		liquidationInfo = append(liquidationInfo, fmt.Sprintf("5s: %dL | 60s: %dL | 10s: %dS",
			tick.LL5,
			tick.LL60,
			tick.SL10,
		))
	}
	if len(liquidationInfo) > 0 {
		lines = append(lines, "💥 <b>Liquidations:</b>\n"+strings.Join(liquidationInfo, " | "))
	}

	if hasAlert {
		sign1m := ""
		if tick.Avg.Change1m > 0 {
			sign1m = "+"
		}
		sign20m := ""
		if tick.Avg.Change20m > 0 {
			sign20m = "+"
		}

		signAskChange := ""
		if tick.Avg.AskChange > 0 {
			signAskChange = "+"
		}
		signBidChange := ""
		if tick.Avg.BidChange > 0 {
			signBidChange = "+"
		}

		marketOverview := fmt.Sprintf("📈 <b>Market Avg Overview (%d pairs):</b>\n 1m: %s%.2f%% | 20m: %s%.2f%% \n Ask: %s%.2f%% | Bid: %s%.2f \n Max10: %.2f | Min10: %.2f",
			tick.Avg.TickersCount,
			sign1m, tick.Avg.Change1m,
			sign20m, tick.Avg.Change20m,
			signAskChange, tick.Avg.AskChange,
			signBidChange, tick.Avg.BidChange,
			tick.Avg.Max10,
			tick.Avg.Min10,
		)
		lines = append(lines, marketOverview)
	}

	if !hasAlert {
		return "", false
	}

	return strings.Join(lines, "\n\n"), true
}
