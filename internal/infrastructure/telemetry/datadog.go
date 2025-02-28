package telemetry

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"gopkg.in/DataDog/dd-trace-go.v1/profiler"
)

// DatadogConfig holds configuration for Datadog services
type DatadogConfig struct {
	AgentHost       string
	AgentPort       string
	ServiceName     string
	ServiceEnv      string
	Tags            []string
	EnableTracing   bool
	EnableMetrics   bool
	EnableProfiling bool
}

// DatadogProvider provides access to DataDog services
type DatadogProvider struct {
	config      *DatadogConfig
	statsd      *statsd.Client
	initialized bool
}

// NewDatadogProvider creates a new DatadogProvider with the given config
func NewDatadogProvider(config *DatadogConfig) *DatadogProvider {
	return &DatadogProvider{
		config: config,
	}
}

// Initialize sets up all configured DataDog services
func (dp *DatadogProvider) Initialize(_ context.Context) error {
	if dp.initialized {
		return nil
	}

	var err error

	// Initialize tracing if enabled
	if dp.config.EnableTracing {
		tracer.Start(
			tracer.WithServiceName(dp.config.ServiceName),
			tracer.WithEnv(dp.config.ServiceEnv),
			tracer.WithRuntimeMetrics(),
			tracer.WithAgentAddr(fmt.Sprintf("%s:%s", dp.config.AgentHost, dp.config.AgentPort)),
		)
	}

	// Initialize metrics if enabled
	if dp.config.EnableMetrics {
		dp.statsd, err = statsd.New(fmt.Sprintf("%s:%s", dp.config.AgentHost, "8125"), statsd.WithTags(dp.config.Tags))
		if err != nil {
			return fmt.Errorf("failed to initialize statsd client: %w", err)
		}
	}

	// Initialize profiling if enabled
	if dp.config.EnableProfiling {
		err = profiler.Start(
			profiler.WithService(dp.config.ServiceName),
			profiler.WithEnv(dp.config.ServiceEnv),
			profiler.WithTags(dp.config.Tags...),
			profiler.WithAgentAddr(fmt.Sprintf("%s:%s", dp.config.AgentHost, dp.config.AgentPort)),
		)
		if err != nil {
			return fmt.Errorf("failed to initialize profiler: %w", err)
		}
	}

	dp.initialized = true
	return nil
}

// Shutdown stops all DataDog services
func (dp *DatadogProvider) Shutdown() {
	if dp.config.EnableTracing {
		tracer.Stop()
	}

	if dp.config.EnableMetrics && dp.statsd != nil {
		err := dp.statsd.Close()
		if err != nil {
			fmt.Printf("failed to close datadog statsd client: %v\n", err)
		}
	}

	if dp.config.EnableProfiling {
		profiler.Stop()
	}
}

// ddSpan is a simple wrapper for DataDog span
type ddSpan struct {
	span tracer.Span
}

func (s *ddSpan) SetTag(key string, value any) {
	s.span.SetTag(key, value)
}

func (s *ddSpan) Finish() {
	s.span.Finish()
}

// StartSpan starts a new trace span
func (dp *DatadogProvider) StartSpan(ctx context.Context, operationName string) (Span, context.Context) {
	if !dp.config.EnableTracing {
		// Return a no-op span if tracing is disabled
		return &noopSpan{}, ctx
	}

	span, ctx := tracer.StartSpanFromContext(ctx, operationName)

	// Set the component name as a tag for better organization
	// Extract component from operation name if needed (e.g., "fetchTickers" from "fetchTickers.operation")
	parts := strings.Split(operationName, ".")
	component := parts[0]
	span.SetTag("component", component)

	return &ddSpan{span: span}, ctx
}

// Tracer returns a function to start a new span (for backward compatibility)
func (dp *DatadogProvider) Tracer(name string) func(ctx context.Context, operationName string) (Span, context.Context) {
	return func(ctx context.Context, operationName string) (Span, context.Context) {
		fullOperationName := name
		if operationName != name {
			fullOperationName = name + "." + operationName
		}
		return dp.StartSpan(ctx, fullOperationName)
	}
}

// IncrementCounter increments a counter metric
func (dp *DatadogProvider) IncrementCounter(name string, value int64, tags ...string) {
	if !dp.config.EnableMetrics || dp.statsd == nil {
		return
	}
	err := dp.statsd.Count(name, value, tags, 1)
	if err != nil {
		fmt.Printf("failed to increment datdog counter %s: %v\n", name, err)
	}
}

// Gauge sets a gauge metric
func (dp *DatadogProvider) Gauge(name string, value float64, tags ...string) {
	if !dp.config.EnableMetrics || dp.statsd == nil {
		return
	}
	if err := dp.statsd.Gauge(name, value, tags, 1); err != nil {
		fmt.Printf("failed to set datadog gauge %s: %v\n", name, err)
	}
}

// Timing records a timing metric
func (dp *DatadogProvider) Timing(name string, value time.Duration, tags ...string) {
	if !dp.config.EnableMetrics || dp.statsd == nil {
		return
	}
	if err := dp.statsd.Timing(name, value, tags, 1); err != nil {
		fmt.Printf("failed to record datadog timing %s: %v\n", name, err)
	}
}
