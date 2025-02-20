package importer

import (
	"context"
	"fmt"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/notify"
)

// WithNotifier adds a new notifier to the importer
func (i *Importer) WithNotifier(client notify.Client, topicString string, strategy notify.Strategy) error {
	topic := domain.TopicLevel(topicString)
	if err := topic.Validate(); err != nil {
		return fmt.Errorf("invalid topic when adding notifier: %w", err)
	}
	i.notifications.Subscribe(topicString, client, strategy)
	return nil
}

// notifyNewTick sends a notification to all services who are subscribed to market data
func (i *Importer) notifyNewTick(tick *domain.Tick) {
	i.notifications.Notify(context.Background(), domain.TickInfoTopic, tick)
	i.notifications.Notify(context.Background(), domain.MarketDataTopic, tick)
	i.notifications.Notify(context.Background(), domain.AlertTopic, tick)
}
