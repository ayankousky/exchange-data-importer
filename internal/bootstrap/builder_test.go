package bootstrap

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// newTestOptions returns Options configured for testing.
func newTestOptions(exchangeEnabled bool) *Options {
	return &Options{
		Env:         "test",
		ServiceName: "test-service",
		Exchange: ExchangeOptions{
			Binance: struct {
				Enabled bool   `long:"enabled" env:"ENABLED" description:"Enable Binance exchange"`
				APIUrl  string `long:"api-url" env:"API_URL" description:"(optional) Binance API URL"`
				WSUrl   string `long:"ws-url" env:"WS_URL" description:"(optional) Binance WebSocket URL"`
			}{
				Enabled: exchangeEnabled,
				APIUrl:  "https://dummy-api.binance.com",
				WSUrl:   "wss://dummy-ws.binance.com",
			},
		},
		Repository: RepositoryOptions{},
		Notify:     NotifyOptions{}, // leave empty for tests
	}
}

func TestBuilder(t *testing.T) {
	tests := []struct {
		name         string
		setupBuilder func() *Builder
		wantBuildErr bool
		validate     func(t *testing.T, app *App)
	}{
		{
			name: "no exchange enabled should fail",
			setupBuilder: func() *Builder {
				b := NewBuilder()
				b.app.options = newTestOptions(false)
				ctx := context.Background()
				b.WithExchange(ctx)
				return b
			},
			wantBuildErr: true,
		},
		{
			name: "successful build with all components",
			setupBuilder: func() *Builder {
				b := NewBuilder()
				b.app.options = newTestOptions(true)
				ctx := context.Background()
				b.WithLogger(ctx)
				b.WithExchange(ctx)
				b.WithRepository(ctx)
				b.WithNotifiers(ctx)
				return b
			},
			wantBuildErr: false,
			validate: func(t *testing.T, app *App) {
				assert.NotNil(t, app.exchange, "exchange should be set")
				assert.NotNil(t, app.importer, "importer should be set")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := tt.setupBuilder()
			app, err := b.Build()

			if tt.wantBuildErr {
				assert.Error(t, err, "Build should return error")
				return
			}

			assert.NoError(t, err, "Build should not return error")
			if tt.validate != nil {
				tt.validate(t, app)
			}
		})
	}
}

func TestBuilderWithEmptyNotifiers(t *testing.T) {
	// Setup builder with empty notifier topics
	b := NewBuilder()
	opts := newTestOptions(true)
	opts.Notify = NotifyOptions{
		Redis: struct {
			URL    string `long:"url" env:"URL" description:"Redis URL"`
			Topics string `long:"topics" env:"TOPICS" description:"Comma-separated list of topics"`
		}{
			URL:    "redis://dummy",
			Topics: "",
		},
		Telegram: struct {
			BotToken string `long:"bot-token" env:"BOT_TOKEN" description:"Telegram bot token"`
			ChatID   string `long:"chat-id" env:"CHAT_ID" description:"Telegram chat ID"`
			Interval int    `long:"interval" env:"INTERVAL" description:"Min interval in seconds between notifications"`
			Topics   string `long:"topics" env:"TOPICS" description:"Comma-separated list of topics"`
		}{
			BotToken: "",
			ChatID:   "",
			Interval: 0,
			Topics:   "",
		},
		Stdout: struct {
			Topics string `long:"topics" env:"TOPICS" description:"Comma-separated list of topics"`
		}{
			Topics: "random topic",
		},
	}
	b.app.options = opts

	ctx := context.Background()
	b.WithNotifiers(ctx)

	assert.Nil(t, b.err, "no error should be set")
	assert.Equal(t, 1, len(b.app.notifiers), "no notifiers should be configured when topics are empty")
}

func TestMain(m *testing.M) {
	// Clear os.Args to prevent interference with flag parsing.
	os.Args = []string{os.Args[0]}
	os.Exit(m.Run())
}
