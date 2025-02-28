package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewDatadogProvider verifies the provider is created with the right config
func TestNewDatadogProvider(t *testing.T) {
	// Arrange
	config := &DatadogConfig{
		AgentHost:   "localhost",
		AgentPort:   "8126",
		ServiceName: "test-service",
		ServiceEnv:  "test",
	}

	provider := NewDatadogProvider(config)

	assert.NotNil(t, provider)
	assert.Equal(t, config, provider.config)
	assert.False(t, provider.initialized)
	assert.Nil(t, provider.statsd)
}

// TestInitializeAndShutdown tests initialization and shutdown with various configurations
func TestInitializeAndShutdown(t *testing.T) {
	tests := []struct {
		name           string
		config         *DatadogConfig
		wantInitialize bool
		wantShutdown   bool
	}{
		{
			name: "with nothing enabled",
			config: &DatadogConfig{
				AgentHost:   "localhost",
				AgentPort:   "8126",
				ServiceName: "test-service",
				ServiceEnv:  "test",
			},
			wantInitialize: true,
			wantShutdown:   true,
		},
		{
			name: "with only tracing enabled",
			config: &DatadogConfig{
				AgentHost:     "localhost",
				AgentPort:     "8126",
				ServiceName:   "test-service",
				ServiceEnv:    "test",
				EnableTracing: true,
			},
			wantInitialize: true,
			wantShutdown:   true,
		},
		{
			name: "with only profiling enabled",
			config: &DatadogConfig{
				AgentHost:       "localhost",
				AgentPort:       "8126",
				ServiceName:     "test-service",
				ServiceEnv:      "test",
				EnableProfiling: true,
			},
			wantInitialize: true,
			wantShutdown:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			provider := NewDatadogProvider(tt.config)

			// Act & Assert - Initialize
			err := provider.Initialize(context.Background())
			assert.NoError(t, err)
			assert.True(t, provider.initialized)

			// Check that second initialization is a no-op
			err = provider.Initialize(context.Background())
			assert.NoError(t, err)

			// Act & Assert - Shutdown
			provider.Shutdown()
		})
	}
}

// TestSpan tests the span functionality
func TestSpan(t *testing.T) {
	tests := []struct {
		name          string
		enableTracing bool
		wantSpanType  string
	}{
		{
			name:          "tracing enabled",
			enableTracing: true,
			wantSpanType:  "*telemetry.ddSpan",
		},
		{
			name:          "tracing disabled",
			enableTracing: false,
			wantSpanType:  "*telemetry.noopSpan",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			config := &DatadogConfig{
				AgentHost:     "localhost",
				AgentPort:     "8126",
				ServiceName:   "test-service",
				ServiceEnv:    "test",
				EnableTracing: tt.enableTracing,
			}
			provider := NewDatadogProvider(config)
			provider.initialized = true

			// Act
			span, _ := provider.StartSpan(context.Background(), "test.operation")
			tracerFunc := provider.Tracer("component")
			tracerSpan, _ := tracerFunc(context.Background(), "operation")

			// Assert via type checking
			if tt.enableTracing {
				assert.IsType(t, &ddSpan{}, span)
			} else {
				assert.IsType(t, &noopSpan{}, span)
			}

			// Execute span methods for coverage
			span.SetTag("key", "value")
			span.Finish()
			tracerSpan.SetTag("key", "value")
			tracerSpan.Finish()
		})
	}
}

// TestMetricsNoClientNoPanic tests that metric methods don't panic when client is nil
func TestMetricsNoClientNoPanic(t *testing.T) {
	// Arrange
	config := &DatadogConfig{
		EnableMetrics: true, // Even with metrics enabled
	}
	provider := NewDatadogProvider(config)
	provider.initialized = true
	// Explicitly ensure statsd is nil
	provider.statsd = nil

	// Act & Assert - These should not panic
	assert.NotPanics(t, func() {
		provider.IncrementCounter("test.counter", 1, "tag1:value1")
		provider.Gauge("test.gauge", 42.0, "tag1:value1")
		provider.Timing("test.timing", 100*time.Millisecond, "tag1:value1")
	})
}

// TestMetricsDisabled tests that metric methods are no-ops when metrics are disabled
func TestMetricsDisabled(t *testing.T) {
	// Arrange
	config := &DatadogConfig{
		EnableMetrics: false, // Metrics disabled
	}
	provider := NewDatadogProvider(config)
	provider.initialized = true

	// Act & Assert - These should be no-ops
	assert.NotPanics(t, func() {
		provider.IncrementCounter("test.counter", 1, "tag1:value1")
		provider.Gauge("test.gauge", 42.0, "tag1:value1")
		provider.Timing("test.timing", 100*time.Millisecond, "tag1:value1")
	})
}

// TestAllFeaturesDisabled tests that the provider works with all features disabled
func TestAllFeaturesDisabled(t *testing.T) {
	// Arrange
	config := &DatadogConfig{
		AgentHost:       "localhost",
		AgentPort:       "8126",
		ServiceName:     "test-service",
		ServiceEnv:      "test",
		EnableTracing:   false,
		EnableMetrics:   false,
		EnableProfiling: false,
	}

	// Act
	provider := NewDatadogProvider(config)
	err := provider.Initialize(context.Background())

	// Assert
	assert.NoError(t, err)
	assert.True(t, provider.initialized)
	assert.Nil(t, provider.statsd)

	// Shutdown should not panic
	assert.NotPanics(t, func() {
		provider.Shutdown()
	})
}
