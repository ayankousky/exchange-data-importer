package importer

import (
	"context"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/notify"
)

// WithNotifier adds a new notifier to the importer
func (i *Importer) WithNotifier(client notify.Client, topic string, strategy notify.Strategy) error {
	i.notifier.Subscribe(topic, client, strategy)
	return nil
}

// notifyNewTick sends a notification to all services who are subscribed to market data
func (i *Importer) notifyNewTick(tick *domain.Tick) {
	i.notifier.Notify(context.Background(), tick)
}
