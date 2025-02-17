package importer

import (
	"context"
	"fmt"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/notify"
	"go.uber.org/zap"
)

var tgAlertThresholds = domain.TickAlertThresholds{
	AvgPrice1mChange:    2.0,
	AvgPrice20mChange:   5.0,
	TickerPrice1mChange: 15.0,
}

// WithNotifier adds a new notifier to the importer
func (i *Importer) WithNotifier(client notify.Client, topicString string) error {
	topic := domain.TopicLevel(topicString)
	if err := topic.Validate(); err != nil {
		return fmt.Errorf("invalid topic when adding notifier: %w", err)
	}
	return i.notificationHub.Subscribe(topic.String(), client)
}

// notifyNewTick sends a notification to all services who are subscribed to market data
func (i *Importer) notifyNewTick(tick *domain.Tick) {
	// notify bots
	if i.notificationHub.GetSubscriberCount(domain.MarketDataTopic) > 0 {
		for tickerName := range tick.Data {
			tickerNotification, err := domain.NewTickerNotification(tick, tickerName)
			if err != nil {
				i.logger.Error("Failed to create ticker notification", zap.Error(err))
				continue
			}

			event := notify.Event{
				Time:      time.Now(),
				EventType: domain.MarketDataTopic,
				Data:      tickerNotification,
			}

			i.notificationHub.Publish(context.Background(), domain.MarketDataTopic, event)
		}
	}

	// notify alerts
	if i.notificationHub.GetSubscriberCount(domain.AlertTopic) > 0 {
		marketStateAlertMessage, hasAlerts := domain.FormatTickAlert(tick, tgAlertThresholds)
		if hasAlerts {
			event := notify.Event{
				Time:      time.Now(),
				EventType: domain.AlertTopic,
				Data:      marketStateAlertMessage,
			}
			i.notificationHub.Publish(context.Background(), domain.AlertTopic, event)
		}
	}

}
