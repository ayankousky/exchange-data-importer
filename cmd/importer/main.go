package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/importer"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges"
	binanceExchange "github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges/binance"
	bybitExchange "github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges/bybit"
	okxExchange "github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges/okx"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/repository/mongo"
	"github.com/jessevdk/go-flags"
	"go.uber.org/zap"

	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/notify"
)

// Options holds all the configuration options
type options struct {
	Env         string `long:"env" env:"ENV" description:"Environment"`
	ServiceName string `long:"service-name" env:"SERVICE_NAME" description:"Service name"`

	Repository struct {
		Mongo struct {
			URL string `long:"url" env:"URL" description:"MongoDB URL"`
		} `group:"mongo" namespace:"mongo" env-namespace:"MONGO"`
	} `group:"repository" namespace:"repository" env-namespace:"REPOSITORY"`

	Exchange struct {
		Binance struct {
			APIUrl string `long:"api-url" env:"API_URL" description:"Binance API URL"`
			WSUrl  string `long:"ws-url" env:"WS_URL" description:"Binance WebSocket URL"`
			Name   string `long:"name" env:"NAME" description:"Binance name"`
		} `group:"binance" namespace:"binance" env-namespace:"BINANCE"`

		Bybit struct {
			APIUrl string `long:"api-url" env:"API_URL" description:"Bybit API URL"`
			WSUrl  string `long:"ws-url" env:"WS_URL" description:"Bybit WebSocket URL"`
			Name   string `long:"name" env:"NAME" description:"Bybit name"`
		} `group:"bybit" namespace:"bybit" env-namespace:"BYBIT"`

		OKX struct {
			APIUrl string `long:"api-url" env:"API_URL" description:"OKX API URL"`
			WSUrl  string `long:"ws-url" env:"WS_URL" description:"OKX WebSocket URL"`
			Name   string `long:"name" env:"OKX_NAME" description:"OKX name"`
		} `group:"okx" namespace:"okx" env-namespace:"OKX"`
	} `group:"exchange" namespace:"exchange" env-namespace:"EXCHANGE"`

	Notify struct {
		Redis struct {
			URL   string `long:"url" env:"URL" description:"Redis URL"`
			Topic string `long:"topic" env:"TOPIC" description:"Redis topic"`
		} `group:"redis" namespace:"redis" env-namespace:"REDIS"`

		Telegram struct {
			BotToken string `long:"bot-token" env:"BOT_TOKEN" description:"Telegram bot token"`
			ChatID   string `long:"chat-id" env:"CHAT_ID" description:"Telegram chat ID"`
			Topic    string `long:"topic" env:"TOPIC" description:"Telegram topic"`
		} `group:"telegram" namespace:"telegram" env-namespace:"TELEGRAM"`
	} `group:"notify" namespace:"notify" env-namespace:"NOTIFY"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Parse app configuration
	var opts options
	parser := flags.NewParser(&opts, flags.Default)
	_, err := parser.Parse()
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			fmt.Println(flagsErr)
			return
		}
		fmt.Printf("Error parsing flags: %v\n", err)
		return
	}

	// Create a new logger
	logger, _ := infrastructure.NewLogger("development", "importer")
	defer logger.Sync()

	logger.Info("Starting importer...")

	repoFactory, err := getRepositoryFactory(opts)
	if err != nil {
		logger.Error("Error creating repository factory", zap.Error(err))
		return
	}

	// Create exchange selected by the user
	exchange, err := getExchange(opts)
	if err != nil {
		logger.Error("Error creating exchange", zap.Error(err))
		return
	}

	// Create importer (the main service)
	exchangeImporter := importer.NewImporter(exchange, repoFactory, logger)

	// Add notifiers to the importer
	notifiers := getNotifiers(opts, logger)
	for _, notifier := range notifiers {
		if err := exchangeImporter.WithNotifier(notifier.Client, notifier.Topic); err != nil {
			logger.Warn("Error adding notifier", zap.Error(err))
		}
	}

	// Start handling imports
	if err := exchangeImporter.StartImportLoop(ctx, time.Second); err != nil {
		logger.Error("Error starting import loop", zap.Error(err))
	}

	<-ctx.Done()
	logger.Info("Exiting...")
}

func getRepositoryFactory(opts options) (importer.RepositoryFactory, error) {
	if opts.Repository.Mongo.URL != "" {
		mongoClient, err := infrastructure.NewMongoClient(opts.Repository.Mongo.URL)
		if err != nil {
			return nil, err
		}

		mongoFactory, err := mongo.NewMongoRepoFactory(mongoClient)
		if err != nil {
			return nil, err
		}

		return mongoFactory, nil
	}

	return nil, fmt.Errorf("no repository factory found")
}

func getNotifiers(opts options, logger *zap.Logger) []struct {
	Client notify.Client
	Topic  string
} {
	var notifiers []struct {
		Client notify.Client
		Topic  string
	}

	// Redis Notifier
	if opts.Notify.Redis.URL != "" {
		redisClient, err := infrastructure.NewRedisClient(context.Background(), opts.Notify.Redis.URL, 1)
		if err != nil {
			logger.Warn("Failed to initialize Redis notifier", zap.Error(err))
		} else {
			notifiers = append(notifiers, struct {
				Client notify.Client
				Topic  string
			}{
				Client: notify.NewRedisNotifier(redisClient, "exchange:notifications"),
				Topic:  opts.Notify.Redis.Topic,
			})
		}
	}

	// Telegram Notifier
	if opts.Notify.Telegram.BotToken != "" && opts.Notify.Telegram.ChatID != "" {
		tgNotifier, err := notify.NewTelegramNotifier(opts.Notify.Telegram.BotToken, opts.Notify.Telegram.ChatID)
		if err != nil {
			logger.Warn("Failed to initialize Telegram notifier", zap.Error(err))
		} else {
			notifiers = append(notifiers, struct {
				Client notify.Client
				Topic  string
			}{
				Client: tgNotifier,
				Topic:  opts.Notify.Telegram.Topic,
			})
		}
	}

	return notifiers
}

func getExchange(opts options) (exchanges.Exchange, error) {
	if opts.Exchange.Binance.Name != "" {
		return binanceExchange.NewBinance(binanceExchange.Config{
			Name:   opts.Exchange.Binance.Name,
			APIUrl: opts.Exchange.Binance.APIUrl,
			WSUrl:  opts.Exchange.Binance.WSUrl,
		}), nil
	}

	if opts.Exchange.Bybit.Name != "" {
		return bybitExchange.NewBybit(bybitExchange.Config{
			Name:   opts.Exchange.Bybit.Name,
			APIUrl: opts.Exchange.Bybit.APIUrl,
			WSUrl:  opts.Exchange.Bybit.WSUrl,
		}), nil
	}

	if opts.Exchange.OKX.Name != "" {
		return okxExchange.NewOKX(okxExchange.Config{
			Name:   opts.Exchange.OKX.Name,
			APIUrl: opts.Exchange.OKX.APIUrl,
			WSUrl:  opts.Exchange.OKX.WSUrl,
		}), nil
	}

	return nil, fmt.Errorf("no exchange found")
}
