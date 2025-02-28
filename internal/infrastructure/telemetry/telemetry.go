package telemetry

import (
	"context"
	"time"
)

// Span represents a tracing span
type Span interface {
	// SetTag adds a tag to the span
	SetTag(key string, value any)

	// Finish completes the span
	Finish()
}

// noopSpan for when tracing is disabled
type noopSpan struct{}

func (s *noopSpan) SetTag(_ string, _ any) {}
func (s *noopSpan) Finish()                {}

// Provider defines the interface for telemetry providers
type Provider interface {
	// Initialize sets up the telemetry provider
	Initialize(ctx context.Context) error

	// Shutdown stops the telemetry provider
	Shutdown()

	// StartSpan starts a new span for the given operation
	StartSpan(ctx context.Context, operationName string) (Span, context.Context)

	// IncrementCounter increments a counter metric
	IncrementCounter(name string, value int64, tags ...string)

	// Gauge sets a gauge metric
	Gauge(name string, value float64, tags ...string)

	// Timing records a timing metric
	Timing(name string, value time.Duration, tags ...string)
}
