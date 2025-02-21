package importer

import (
	"context"
	"fmt"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/notify"
	"github.com/ayankousky/exchange-data-importer/internal/notifier"
)

// WithNotifier adds a new notifier to the importer
func (i *Importer) WithNotifier(client notify.Client, topicString string, strategy notify.Strategy) error {
	topic := notifier.Topic(topicString)
	if err := topic.Validate(); err != nil {
		return fmt.Errorf("invalid topic when adding notifier: %w", err)
	}
	i.notifier.Subscribe(topic, client, strategy)
	return nil
}

// notifyNewTick sends a notification to all services who are subscribed to market data
func (i *Importer) notifyNewTick(tick *domain.Tick) {
	i.notifier.Notify(context.Background(), tick)
}
