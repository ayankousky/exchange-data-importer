package notifier

import (
	"context"
	"fmt"

	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/notify"
	"go.uber.org/zap"
)

// Topic represents a notification topic
type Topic string

// Validate checks if the topic exists
func (t Topic) Validate() error {
	switch t {
	case MarketDataTopic, AlertTopic, TickInfoTopic:
		return nil
	default:
		return fmt.Errorf("invalid topic: '%s'", t)
	}
}

const (
	// MarketDataTopic is the event type for ticker data
	MarketDataTopic Topic = "TICKER"

	// AlertTopic is the event triggered when something significant happens in the market
	AlertTopic Topic = "ALERT_MARKET_STATE"

	// TickInfoTopic is the event triggered to send common information about the tick
	TickInfoTopic Topic = "TICK_INFO"
)

// Notifier is the service responsible for handling notifications
type Notifier struct {
	handlers map[Topic][]handler
	logger   *zap.Logger
}

type handler struct {
	client   notify.Client
	strategy notify.Strategy
}

// New creates a new Notifier
func New(logger *zap.Logger) *Notifier {
	return &Notifier{
		handlers: make(map[Topic][]handler),
		logger:   logger.With(zap.String("component", "notifier")),
	}
}

// Subscribe subscribes client to a topic with a given strategy
func (s *Notifier) Subscribe(topic Topic, client notify.Client, strategy notify.Strategy) {
	if client == nil {
		s.logger.Error("Cannot subscribe with nil client",
			zap.String("topic", string(topic)),
		)
		return
	}

	if strategy == nil {
		s.logger.Error("Cannot subscribe with nil strategy",
			zap.String("topic", string(topic)),
		)
		return
	}

	s.handlers[topic] = append(s.handlers[topic], handler{
		client:   client,
		strategy: strategy,
	})
}

// Notify sends a notification to all subscribers of the topic
func (s *Notifier) Notify(ctx context.Context, data any) {
	if data == nil {
		s.logger.Warn("Received nil data for notification")
		return
	}

	s.notify(ctx, MarketDataTopic, data)
	s.notify(ctx, TickInfoTopic, data)
	s.notify(ctx, AlertTopic, data)
}

func (s *Notifier) notify(ctx context.Context, topic Topic, data any) {
	handlers, exists := s.handlers[topic]
	if !exists {
		return
	}

	for _, h := range handlers {
		events := h.strategy.Format(data)
		for _, event := range events {
			if err := h.client.Send(ctx, event); err != nil {
				s.logger.Error("Failed to send notification",
					zap.String("topic", string(topic)),
					zap.Error(err),
				)
				continue
			}
		}
	}
}
