package notify

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// RedisNotifier is a Redis-based implementation of domain.NotificationService
type RedisNotifier struct {
	client  *redis.Client
	channel string
}

// NewRedisNotifier creates a new RedisNotifier
func NewRedisNotifier(client *redis.Client, channel string) *RedisNotifier {
	return &RedisNotifier{
		client:  client,
		channel: channel,
	}
}

// Send event to the listeners
func (p *RedisNotifier) Send(ctx context.Context, event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshaling event: %w", err)
	}

	if err := p.client.Publish(ctx, p.channel, data).Err(); err != nil {
		return fmt.Errorf("publishing to Redis: %w", err)
	}

	return nil
}
