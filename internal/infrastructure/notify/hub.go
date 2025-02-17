package notify

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type clientTopic string

// NotificationHub manages subscribers for different topics
type NotificationHub struct {
	subscribers map[clientTopic][]Client
	mu          sync.RWMutex
	logger      *zap.Logger
}

// NewNotificationHub creates a new NotificationHub
func NewNotificationHub(logger *zap.Logger) *NotificationHub {
	return &NotificationHub{
		subscribers: make(map[clientTopic][]Client),
		logger:      logger,
	}
}

// Subscribe adds a client to a specific topic
func (h *NotificationHub) Subscribe(topic string, client Client) error {
	if client == nil {
		return fmt.Errorf("client cannot be nil")
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	h.subscribers[clientTopic(topic)] = append(h.subscribers[clientTopic(topic)], client)
	return nil
}

// Publish sends an event to all subscribers of a topic
func (h *NotificationHub) Publish(ctx context.Context, topic string, event Event) {
	h.mu.RLock()
	clients := make([]Client, len(h.subscribers[clientTopic(topic)]))
	copy(clients, h.subscribers[clientTopic(topic)])
	h.mu.RUnlock()

	// Use errgroup for parallel publishing with context
	g, ctx := errgroup.WithContext(ctx)

	for _, client := range clients {
		c := client // Capture for goroutine
		g.Go(func() error {
			// Add timeout for each publish
			timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			if err := c.Send(timeoutCtx, event); err != nil {
				h.logger.Error("Failed to publish event",
					zap.Error(err),
					zap.String("topic", topic))
				return err
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		h.logger.Error("Some publish operations failed", zap.Error(err))
	}
}

// GetSubscriberCount returns the number of subscribers for a topic
func (h *NotificationHub) GetSubscriberCount(topic string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subscribers[clientTopic(topic)])
}
