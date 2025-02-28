package telemetry

import (
	"context"
	"time"
)

// NoopProvider provides a no-operation implementation for telemetry or tracing functionalities
// Useful for scenarios where telemetry/tracing is disabled or not required
type NoopProvider struct{}

// Initialize sets up the NoopProvider instance; for this implementation, it performs no operations and always returns nil
func (p *NoopProvider) Initialize(_ context.Context) error {
	return nil
}

// Shutdown performs a no-operation cleanup for the NoopProvider. It is a placeholder to satisfy Provider interface
func (p *NoopProvider) Shutdown() {}

// StartSpan creates a new noop span and returns it along with the context, providing a no-op implementation of span creation
func (p *NoopProvider) StartSpan(ctx context.Context, _ string) (Span, context.Context) {
	return &noopSpan{}, ctx
}

// IncrementCounter is a no-op implementation that does not perform any action when called
func (p *NoopProvider) IncrementCounter(_ string, _ int64, _ ...string) {}

// Gauge records a gauge value with a given name, value, and optional tags. No operation is performed in NoopProvider
func (p *NoopProvider) Gauge(_ string, _ float64, _ ...string) {}

// Timing records a timing metric with a specified name and duration, allowing optional labels
func (p *NoopProvider) Timing(_ string, _ time.Duration, _ ...string) {}
