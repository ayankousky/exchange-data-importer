package bootstrap

import (
	"context"
	"fmt"
	"strings"

	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/repository/memory"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/repository/sqlite"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/telemetry"
	"github.com/ayankousky/exchange-data-importer/internal/notifier"
	notificationStrategies "github.com/ayankousky/exchange-data-importer/internal/notifier/strategies"
	"go.uber.org/zap"

	"github.com/ayankousky/exchange-data-importer/internal/importer"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure"
	binanceExchange "github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges/binance"
	bybitExchange "github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges/bybit"
	okxExchange "github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges/okx"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/notify"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/repository/mongo"
)

// Builder builds the App instance
type Builder struct {
	app *App
	err error
}

// NewBuilder creates a new Builder instance
func NewBuilder() *Builder {
	app := &App{}

	app.logger, _ = infrastructure.NewLogger("development", "exchange-data-importer")
	app.repositoryFactory = memory.NewInMemoryRepoFactory()
	app.telemetry = &telemetry.NoopProvider{}

	builder := &Builder{
		app: app,
	}
	builder.fetchOptions()

	return builder
}

// fetchOptions automatically fetches options from env/flags
func (b *Builder) fetchOptions() *Builder {
	if b.err != nil {
		return b
	}

	opts, err := ParseOptions()
	if err != nil {
		b.err = fmt.Errorf("parsing options: %w", err)
		return b
	}

	b.app.options = opts
	return b
}

// WithLogger initializes the logger
func (b *Builder) WithLogger(_ context.Context) *Builder {
	if b.err != nil {
		return b
	}

	logger, err := infrastructure.NewLogger(b.app.options.Env, b.app.options.ServiceName)
	if err != nil {
		b.err = fmt.Errorf("creating logger: %w", err)
		return b
	}

	b.app.logger = logger
	return b
}

// WithExchange initializes the exchange client
func (b *Builder) WithExchange(_ context.Context) *Builder {
	if b.err != nil {
		return b
	}

	if b.app.options.Exchange.Binance.Enabled {
		b.app.exchange = binanceExchange.NewBinance(binanceExchange.Config{
			Name:   b.app.options.ServiceName,
			APIUrl: b.app.options.Exchange.Binance.APIUrl,
			WSUrl:  b.app.options.Exchange.Binance.WSUrl,
		})
		return b
	}

	if b.app.options.Exchange.Bybit.Enabled {
		b.app.exchange = bybitExchange.NewBybit(bybitExchange.Config{
			Name:   b.app.options.ServiceName,
			APIUrl: b.app.options.Exchange.Bybit.APIUrl,
			WSUrl:  b.app.options.Exchange.Bybit.WSUrl,
		})
		return b
	}

	if b.app.options.Exchange.OKX.Enabled {
		b.app.exchange = okxExchange.NewOKX(okxExchange.Config{
			Name:   b.app.options.ServiceName,
			APIUrl: b.app.options.Exchange.OKX.APIUrl,
			WSUrl:  b.app.options.Exchange.OKX.WSUrl,
		})
		return b
	}

	b.err = fmt.Errorf("no exchange configured")
	return b
}

// WithRepository initializes the repository factory
func (b *Builder) WithRepository(ctx context.Context) *Builder {
	if b.err != nil {
		return b
	}

	if b.app.options.Repository.Mongo.Enabled {
		mongoClient, err := infrastructure.NewMongoClient(ctx, b.app.options.Repository.Mongo.URL)
		if err != nil {
			b.err = fmt.Errorf("creating mongo client: %w", err)
			return b
		}
		repoFactory, err := mongo.NewMongoRepoFactory(mongoClient)
		if err != nil {
			b.err = fmt.Errorf("creating repository factory: %w", err)
			return b
		}
		b.app.repositoryFactory = repoFactory
		return b
	}

	if b.app.options.Repository.Sqlite.Enabled && b.app.options.Repository.Sqlite.Path != "" {
		dsn := fmt.Sprintf("file:%s_%s?cache=shared&_foreign_keys=on", b.app.options.ServiceName, b.app.options.Repository.Sqlite.Path)
		repoFactory, err := sqlite.NewSQLiteRepoFactory(dsn)
		if err != nil {
			b.err = fmt.Errorf("creating repository factory: %w", err)
			return b
		}
		b.app.repositoryFactory = repoFactory
		return b
	}

	return b
}

// WithNotifiers initializes the notifiers
func (b *Builder) WithNotifiers(ctx context.Context) *Builder {
	if b.err != nil {
		return b
	}

	var notifiers []NotifierConfig

	// Helper function to split topics
	splitTopics := func(topics string) []string {
		var result []string
		for _, t := range strings.Split(topics, ",") {
			if trimmed := strings.TrimSpace(t); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}

	// Initialize Redis notifier if configured
	if b.app.options.Notify.Redis.Topics != "" {
		redisClient, err := infrastructure.NewRedisClient(ctx, b.app.options.Notify.Redis.URL, 1)
		if err != nil {
			b.app.logger.Warn("Failed to initialize Redis notifier", zap.Error(err))
		} else {
			for _, topic := range splitTopics(b.app.options.Notify.Redis.Topics) {
				notifiers = append(notifiers, NotifierConfig{
					Client:   notify.NewRedisNotifier(redisClient, fmt.Sprintf("%s:%s", b.app.options.ServiceName, topic)),
					Topic:    topic,
					Strategy: &notificationStrategies.MarketDataStrategy{},
				})
			}
		}
	}

	// Initialize Telegram notifier if configured
	if b.app.options.Notify.Telegram.Topics != "" {
		tgNotifier, err := notify.NewTelegramNotifier(
			b.app.options.Notify.Telegram.BotToken,
			b.app.options.Notify.Telegram.ChatID,
			b.app.options.Notify.Telegram.Interval,
		)
		if err != nil {
			b.app.logger.Warn("Failed to initialize Telegram notifier", zap.Error(err))
		} else {
			var tgAlertThresholds = notificationStrategies.AlertStrategyThresholds{
				AvgPrice1mChange:    2.0,
				AvgPrice20mChange:   5.0,
				TickerPrice1mChange: 15.0,
			}
			for _, topic := range splitTopics(b.app.options.Notify.Telegram.Topics) {
				notifiers = append(notifiers, NotifierConfig{
					Client:   tgNotifier,
					Topic:    topic,
					Strategy: notificationStrategies.NewAlertStrategy(tgAlertThresholds),
				})
			}
		}
	}

	// Initialize stdout notifier if configured
	if b.app.options.Notify.Stdout.Topics != "" {
		stdoutNotifier := notify.NewConsoleNotifier()
		for _, topic := range splitTopics(b.app.options.Notify.Stdout.Topics) {
			notifiers = append(notifiers, NotifierConfig{
				Client:   stdoutNotifier,
				Topic:    topic,
				Strategy: notificationStrategies.NewTickInfoStrategy(),
			})
		}
	}

	b.app.notifiers = notifiers
	return b
}

// WithTelemetry initializes telemetry (e.g., metrics and tracing)
func (b *Builder) WithTelemetry(ctx context.Context, revision string) *Builder {
	if b.err != nil {
		return b
	}

	revisionTag := fmt.Sprintf("revision:%s", revision)

	// Initialize datadog provider
	if b.app.options.Telemetry.Datadog.Enabled {
		datadogConfig := &telemetry.DatadogConfig{
			AgentHost:       b.app.options.Telemetry.Datadog.AgentHost,
			AgentPort:       b.app.options.Telemetry.Datadog.AgentPort,
			ServiceName:     b.app.options.ServiceName,
			ServiceEnv:      b.app.options.Env,
			EnableTracing:   b.app.options.Telemetry.Datadog.EnabledTracing,
			EnableMetrics:   b.app.options.Telemetry.Datadog.EnabledMetrics,
			EnableProfiling: b.app.options.Telemetry.Datadog.EnabledProfiling,
			Tags:            []string{revisionTag},
		}

		fmt.Printf("Datadog Config: %+v\n", datadogConfig)
		telemetryProvider := telemetry.NewDatadogProvider(datadogConfig)
		if err := telemetryProvider.Initialize(ctx); err != nil {
			b.err = fmt.Errorf("initializing telemetry provider: %w", err)
		}
		b.app.telemetry = telemetryProvider
	}

	return b
}

// Build returns the built App instance
func (b *Builder) Build() (*App, error) {
	if b.err != nil {
		return nil, b.err
	}
	notifier := notifier.New(b.app.logger) // currently hardcoded as there is no alternatives

	b.app.importer = importer.New(&importer.Config{
		Exchange:          b.app.exchange,
		RepositoryFactory: b.app.repositoryFactory,
		NotifierService:   notifier,
		Logger:            b.app.logger,
		Telemetry:         b.app.telemetry,
	})

	if b.app.exchange == nil || b.app.importer == nil {
		return nil, fmt.Errorf("missing required dependencies")
	}

	return b.app, nil
}
