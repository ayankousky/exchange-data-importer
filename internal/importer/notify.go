package importer

import (
	"context"
	"log"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/notify"
)

// WithMarketNotify adds a new publisher to the list of marketNotifiers
func (i *Importer) WithMarketNotify(notifier notify.Client) {
	i.marketNotifiers = append(i.marketNotifiers, notifier)
}

// notifyNewTick sends a notification to all services who are subscribed to market data
func (i *Importer) notifyNewTick(tick *domain.Tick) {
	// notify bots
	for _, publisher := range i.marketNotifiers {
		for tickerName := range tick.Data {
			tickerNotification, err := domain.NewTickerNotification(tick, tickerName)
			if err != nil {
				log.Printf("Failed to create ticker notification: %v", err)
				continue
			}

			event := notify.Event{
				Time:      time.Now(),
				EventType: domain.EventTypeTicker,
				Data:      tickerNotification,
			}

			if err := publisher.Send(context.Background(), event); err != nil {
				log.Printf("Failed to publish tick: %v", err)
			}
		}
	}
}
