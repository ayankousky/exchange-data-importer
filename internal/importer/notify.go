package importer

import (
	"context"
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

// WithMarketNotify adds a new publisher to the list of marketNotifiers
func (i *Importer) WithMarketNotify(notifier notify.Client) {
	if notifier == nil {
		i.logger.Warn("Cannot add nil notifier to market notifiers")
		return
	}
	i.marketNotifiers = append(i.marketNotifiers, notifier)
}

// WithAlertNotify adds a new publisher to the list of alertNotifiers
func (i *Importer) WithAlertNotify(notifier notify.Client) {
	if notifier == nil {
		i.logger.Warn("Cannot add nil notifier to alert notifiers")
		return
	}
	i.alertNotifiers = append(i.alertNotifiers, notifier)
}

// notifyNewTick sends a notification to all services who are subscribed to market data
func (i *Importer) notifyNewTick(tick *domain.Tick) {
	// notify bots
	for _, publisher := range i.marketNotifiers {
		for tickerName := range tick.Data {
			tickerNotification, err := domain.NewTickerNotification(tick, tickerName)
			if err != nil {
				i.logger.Error("Failed to create ticker notification", zap.Error(err))
				continue
			}

			event := notify.Event{
				Time:      time.Now(),
				EventType: domain.EventTypeTicker,
				Data:      tickerNotification,
			}

			if err := publisher.Send(context.Background(), event); err != nil {
				i.logger.Error("Failed to publish tick", zap.Error(err))
			}
		}
	}

	// notify alerts
	marketStateAlertMessage, hasAlerts := domain.FormatTickAlert(tick, tgAlertThresholds)
	if hasAlerts {
		for _, publisher := range i.alertNotifiers {
			event := notify.Event{
				Time:      time.Now(),
				EventType: domain.EventTypeMarketAlert,
				Data:      marketStateAlertMessage,
			}

			if err := publisher.Send(context.Background(), event); err != nil {
				i.logger.Error("Failed to publish alert", zap.Error(err))
			}
		}
	}

}
