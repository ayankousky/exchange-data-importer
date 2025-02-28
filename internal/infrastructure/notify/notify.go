package notify

import (
	"context"
	"time"
)

//go:generate moq --out mocks/client.go --pkg mocks --with-resets --skip-ensure . Client
//go:generate moq --out mocks/strategy.go --pkg mocks --with-resets --skip-ensure . Strategy

// Event represents a notification event
type Event struct {
	Time      time.Time `json:"ct"`
	EventType string    `json:"event_type"`
	Data      any       `json:"data"`
}

// Client represents a notification service contract
type Client interface {
	Send(ctx context.Context, event Event) error
}

// Strategy defines how to format data for notifications
type Strategy interface {
	Format(data any) []Event
}
