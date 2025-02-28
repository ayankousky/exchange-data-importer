package bootstrap

import (
	"fmt"

	"github.com/jessevdk/go-flags"
)

// Options holds all configuration options
type Options struct {
	Env         string `long:"env" env:"ENV" description:"Environment"`
	ServiceName string `long:"service-name" env:"SERVICE_NAME" description:"Service name"`

	Repository RepositoryOptions `group:"repository" namespace:"repository" env-namespace:"REPOSITORY"`
	Exchange   ExchangeOptions   `group:"exchange" namespace:"exchange" env-namespace:"EXCHANGE"`
	Notify     NotifyOptions     `group:"notify" namespace:"notify" env-namespace:"NOTIFY"`
	Telemetry  TelemetryOptions  `group:"telemetry" namespace:"telemetry" env-namespace:"TELEMETRY"`
}

// RepositoryOptions holds configuration Options for repositories to use (only 1 allowed)
type RepositoryOptions struct {
	Mongo struct {
		Enabled bool   `long:"enabled" env:"ENABLED" description:"Enable MongoDB repository"`
		URL     string `long:"url" env:"URL" description:"MongoDB URL"`
	} `group:"mongo" namespace:"mongo" env-namespace:"MONGO"`
	Sqlite struct {
		Enabled bool   `long:"enabled" env:"ENABLED" description:"Enable SQLite repository"`
		Path    string `long:"path" env:"PATH" description:"SQLite path"`
	} `group:"sqlite" namespace:"sqlite" env-namespace:"SQLITE"`
}

// ExchangeOptions holds configuration Options for exchanges to use (only 1 allowed)
type ExchangeOptions struct {
	Binance struct {
		Enabled bool   `long:"enabled" env:"ENABLED" description:"Enable Binance exchange"`
		APIUrl  string `long:"api-url" env:"API_URL" description:"(optional) Binance API URL"`
		WSUrl   string `long:"ws-url" env:"WS_URL" description:"(optional) Binance WebSocket URL"`
	} `group:"binance" namespace:"binance" env-namespace:"BINANCE"`

	Bybit struct {
		Enabled bool   `long:"enabled" env:"ENABLED" description:"Enable Bybit exchange"`
		APIUrl  string `long:"api-url" env:"API_URL" description:"(optional) Bybit API URL"`
		WSUrl   string `long:"ws-url" env:"WS_URL" description:"(optional) Bybit WebSocket URL"`
	} `group:"bybit" namespace:"bybit" env-namespace:"BYBIT"`

	OKX struct {
		Enabled bool   `long:"enabled" env:"ENABLED" description:"Enable OKX exchange"`
		APIUrl  string `long:"api-url" env:"API_URL" description:"(optional) OKX API URL"`
		WSUrl   string `long:"ws-url" env:"WS_URL" description:"(optional) OKX WebSocket URL"`
	} `group:"okx" namespace:"okx" env-namespace:"OKX"`
}

// NotifyOptions holds configuration Options for notifications (multiple allowed)
type NotifyOptions struct {
	Redis struct {
		URL    string `long:"url" env:"URL" description:"Redis URL"`
		Topics string `long:"topics" env:"TOPICS" description:"Comma-separated list of topics"`
	} `group:"redis" namespace:"redis" env-namespace:"REDIS"`

	Telegram struct {
		BotToken string `long:"bot-token" env:"BOT_TOKEN" description:"Telegram bot token"`
		ChatID   string `long:"chat-id" env:"CHAT_ID" description:"Telegram chat ID"`
		Interval int    `long:"interval" env:"INTERVAL" description:"Min interval in seconds between notifications"`
		Topics   string `long:"topics" env:"TOPICS" description:"Comma-separated list of topics"`
	} `group:"telegram" namespace:"telegram" env-namespace:"TELEGRAM"`

	Stdout struct {
		Topics string `long:"topics" env:"TOPICS" description:"Comma-separated list of topics"`
	} `group:"stdout" namespace:"stdout" env-namespace:"STDOUT"`
}

// TelemetryOptions holds configuration settings for telemetry
type TelemetryOptions struct {
	Datadog struct {
		Enabled          bool   `long:"enabled" env:"ENABLED" description:"Enable Datadog telemetry"`
		AgentHost        string `long:"agent-host" env:"AGENT_HOST" description:"Datadog agent host"`
		AgentPort        string `long:"agent-port" env:"AGENT_PORT" description:"Datadog agent port"`
		EnabledTracing   bool   `long:"enabled-tracing" env:"ENABLED_TRACING" description:"Enable Datadog tracing"`
		EnabledMetrics   bool   `long:"enabled-metrics" env:"ENABLED_METRICS" description:"Enable Datadog metrics"`
		EnabledProfiling bool   `long:"enabled-profiling" env:"ENABLED_PROFILING" description:"Enable Datadog profiling"`
	} `group:"datadog" namespace:"datadog" env-namespace:"DATADOG"`
}

// ParseOptions parses command line arguments and environment variables
func ParseOptions() (*Options, error) {
	var opts Options
	parser := flags.NewParser(&opts, flags.Default)
	if _, err := parser.Parse(); err != nil {
		return nil, fmt.Errorf("parsing options: %w", err)
	}
	return &opts, nil
}
