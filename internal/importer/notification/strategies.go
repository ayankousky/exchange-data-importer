package notification

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/notify"
)

// TickInfoStrategy sends tick data to the console
type TickInfoStrategy struct {
	printCount atomic.Int64
}

// Format creates a notification event for the tick data
func (s *TickInfoStrategy) Format(data any) []notify.Event {
	tick, ok := data.(*domain.Tick)
	if !ok {
		return nil
	}

	const (
		headerFormat = "%-8s | %4s | %8s | %8s | %8s | %6s | %6s | %6s | %6s\n"
		dataFormat   = "%-8s | %4d | %8.2f | %8.2f | %8.2f | %6d | %6d | %6d | %6d\n"
	)

	var output strings.Builder
	count := s.printCount.Add(1)

	if count%10 == 0 {
		fmt.Fprintf(&output, headerFormat,
			"TIME",
			"MKTS",
			"1M CHG%",
			"20M CHG%",
			"AVG BUY",
			"LL5",
			"LL60",
			"SL2",
			"SL10",
		)
	}

	fmt.Fprintf(&output, dataFormat,
		tick.CreatedAt.Format("15:04:05"),
		tick.Avg.TickersCount,
		tick.Avg.Change1m,
		tick.Avg.Change20m,
		tick.AvgBuy10,
		tick.LL5,
		tick.LL60,
		tick.SL2,
		tick.SL10,
	)

	return []notify.Event{{
		Time:      time.Now(),
		EventType: "CONSOLE",
		Data:      output.String(),
	}}
}

// MarketDataStrategy sends market data notifications
type MarketDataStrategy struct{}

// Format creates a notification event for each ticker in the tick data
func (s *MarketDataStrategy) Format(data any) []notify.Event {
	tick, ok := data.(*domain.Tick)
	if !ok {
		return nil
	}

	events := make([]notify.Event, 0)
	for tickerName := range tick.Data {
		notification, err := domain.NewTickerNotification(tick, tickerName)
		if err != nil {
			continue
		}
		events = append(events, notify.Event{
			Time:      time.Now(),
			EventType: domain.MarketDataTopic,
			Data:      notification,
		})
	}
	return events
}

// AlertStrategy sends alerts based on tick data and thresholds configuration
type AlertStrategy struct {
	thresholds domain.TickAlertThresholds
}

// NewAlertStrategy creates a new alert strategy
func NewAlertStrategy(thresholds domain.TickAlertThresholds) *AlertStrategy {
	return &AlertStrategy{thresholds: thresholds}
}

// Format creates a notification event if the tick data meets the alert thresholds
func (s *AlertStrategy) Format(data any) []notify.Event {
	tick, ok := data.(*domain.Tick)
	if !ok {
		return nil
	}

	message, hasAlerts := domain.FormatTickAlert(tick, s.thresholds)
	if !hasAlerts {
		return nil
	}

	return []notify.Event{{
		Time:      time.Now(),
		EventType: domain.AlertTopic,
		Data:      message,
	}}
}
