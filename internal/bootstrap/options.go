package bootstrap

import "github.com/jessevdk/go-flags"

// Options holds all configuration options
type Options struct {
	Env         string `long:"env" env:"ENV" description:"Environment"`
	ServiceName string `long:"service-name" env:"SERVICE_NAME" description:"Service name"`

	Repository struct {
		Mongo struct {
			URL string `long:"url" env:"URL" description:"MongoDB URL"`
		} `group:"mongo" namespace:"mongo" env-namespace:"MONGO"`
	} `group:"repository" namespace:"repository" env-namespace:"REPOSITORY"`

	Exchange struct {
		Binance struct {
			APIUrl string `long:"api-url" env:"API_URL" description:"(optional) Binance API URL"`
			WSUrl  string `long:"ws-url" env:"WS_URL" description:"(optional) Binance WebSocket URL"`
			Name   string `long:"name" env:"NAME" description:"Binance name"`
		} `group:"binance" namespace:"binance" env-namespace:"BINANCE"`

		Bybit struct {
			APIUrl string `long:"api-url" env:"API_URL" description:"(optional) Bybit API URL"`
			WSUrl  string `long:"ws-url" env:"WS_URL" description:"(optional) Bybit WebSocket URL"`
			Name   string `long:"name" env:"NAME" description:"Bybit name"`
		} `group:"bybit" namespace:"bybit" env-namespace:"BYBIT"`

		OKX struct {
			APIUrl string `long:"api-url" env:"API_URL" description:"(optional) OKX API URL"`
			WSUrl  string `long:"ws-url" env:"WS_URL" description:"(optional) OKX WebSocket URL"`
			Name   string `long:"name" env:"NAME" description:"OKX name"`
		} `group:"okx" namespace:"okx" env-namespace:"OKX"`
	} `group:"exchange" namespace:"exchange" env-namespace:"EXCHANGE"`

	Notify struct {
		Redis struct {
			URL    string `long:"url" env:"URL" description:"Redis URL"`
			Topics string `long:"topics" env:"TOPICS" description:"Comma-separated list of topics"`
		} `group:"redis" namespace:"redis" env-namespace:"REDIS"`

		Telegram struct {
			BotToken string `long:"bot-token" env:"BOT_TOKEN" description:"Telegram bot token"`
			ChatID   string `long:"chat-id" env:"CHAT_ID" description:"Telegram chat ID"`
			Topics   string `long:"topics" env:"TOPICS" description:"Comma-separated list of topics"`
		} `group:"telegram" namespace:"telegram" env-namespace:"TELEGRAM"`
	} `group:"notify" namespace:"notify" env-namespace:"NOTIFY"`
}

// ParseOptions parses command line arguments and environment variables
func ParseOptions() (*Options, error) {
	var opts Options
	parser := flags.NewParser(&opts, flags.Default)
	if _, err := parser.Parse(); err != nil {
		return nil, err
	}
	return &opts, nil
}
