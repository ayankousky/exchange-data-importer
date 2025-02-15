package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/importer"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure"
	binanceExchange "github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges/binance"
	bybitExchange "github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges/bybit"
	okxExchange "github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges/okx"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/repository/mongo"
	"go.uber.org/zap"

	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/notify"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Create a new logger
	logger, _ := infrastructure.NewLogger("development", "importer")
	defer logger.Sync()

	logger.Info("Starting importer...")

	// Create a new mongo client
	mongoClient, err := infrastructure.NewMongoClient("mongodb://beatbet-db-mongo:27017")
	if err != nil {
		logger.Error("Error creating mongo client", zap.Error(err))
		return
	}

	mongoFactory, err := mongo.NewMongoRepoFactory(mongoClient)
	if err != nil {
		logger.Error("Error creating mongo factory", zap.Error(err))
		return
	}

	// we will have a list of importers so we could add new exchange in one place
	importers := make([]*importer.Importer, 0)

	redisClient, err := infrastructure.NewRedisClient(ctx, "redis://beatbet-redis:6379", 1)
	if err != nil {
		logger.Error("Error creating redis client", zap.Error(err))
		return
	}

	tgNotifier, err := notify.NewTelegramNotifier(os.Getenv("TELEGRAM_BOT_TOKEN"), os.Getenv("TELEGRAM_CHAT_ID"))
	if err != nil {
		logger.Error("Error creating telegram notifier", zap.Error(err))
		return
	}

	binanceClient := binanceExchange.NewBinance(binanceExchange.Config{
		APIUrl:     binanceExchange.FuturesAPIURL,
		WSUrl:      binanceExchange.FuturesWSUrl,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		Name:       "binance",
	})
	binanceImporter := importer.NewImporter(binanceClient, mongoFactory, logger)
	binanceImporter.WithMarketNotify(notify.NewRedisNotifier(redisClient, fmt.Sprintf("exchange:%s:market", binanceClient.GetName())))
	binanceImporter.WithAlertNotify(tgNotifier)
	importers = append(importers, binanceImporter)

	bybitClient := bybitExchange.NewBybit(bybitExchange.Config{
		APIUrl:     bybitExchange.FuturesAPIURL,
		WSUrl:      bybitExchange.FuturesWSUrl,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		Name:       "bybit",
	})
	importers = append(importers, importer.NewImporter(bybitClient, mongoFactory, logger))

	okxClient := okxExchange.NewOKX(okxExchange.Config{
		APIUrl:     okxExchange.FuturesAPIURL,
		WSUrl:      okxExchange.FuturesWSUrl,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		Name:       "okx",
	})
	importers = append(importers, importer.NewImporter(okxClient, mongoFactory, logger))
	importers = importers[:1] // temporary use only binance

	var wg sync.WaitGroup
	wg.Add(len(importers))

	for _, i := range importers {
		go func(i *importer.Importer) {
			defer wg.Done()
			if err := i.StartImportLoop(ctx, time.Second); err != nil {
				logger.Error("Error starting import loop", zap.Error(err))
			}
		}(i)
	}

	go func() {
		wg.Wait()
		fmt.Println("All importers have stopped!!!!!")
		stop()
	}()

	<-ctx.Done()
	logger.Info("Exiting...")
}
