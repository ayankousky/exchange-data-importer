package notify

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

// Manager handles notification delivery
type Manager struct {
	subscribers map[string][]struct {
		client   Client
		strategy Strategy
	}
	logger *zap.Logger
	mu     sync.RWMutex
}

// NewManager creates a new notification manager
func NewManager(logger *zap.Logger) *Manager {
	return &Manager{
		subscribers: make(map[string][]struct {
			client   Client
			strategy Strategy
		}),
		logger: logger,
	}
}

// Subscribe adds a new subscriber to the manager
func (m *Manager) Subscribe(topic string, client Client, strategy Strategy) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscribers[topic] = append(m.subscribers[topic], struct {
		client   Client
		strategy Strategy
	}{client, strategy})
}

// Notify sends a notification to all subscribers of the topic
func (m *Manager) Notify(ctx context.Context, topic string, data any) {
	m.mu.RLock()
	subs := m.subscribers[topic]
	m.mu.RUnlock()

	for _, sub := range subs {
		events := sub.strategy.Format(data)
		for _, event := range events {
			if err := sub.client.Send(ctx, event); err != nil {
				m.logger.Error("Failed to notify subscriber",
					zap.Error(err),
					zap.String("topic", topic))
			}
		}
	}
}
