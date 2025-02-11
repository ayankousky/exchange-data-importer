package main

import (
	"context"
	"fmt"
	"log"
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
	redisNotificator "github.com/ayankousky/exchange-data-importer/internal/infrastructure/notify"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/repository/mongo"
)

func main() {
	log.Printf("Starting importer...")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Create a new mongo client
	mongoClient, err := infrastructure.NewMongoClient("mongodb://beatbet-db-mongo:27017")
	if err != nil {
		log.Printf("Error creating mongo client: %v", err)
		return
	}

	mongoFactory, err := mongo.NewMongoRepoFactory(mongoClient)
	if err != nil {
		log.Printf("Error creating mongo factory: %v", err)
		return
	}

	// we will have a list of importers so we could add new exchange in one place
	importers := make([]*importer.Importer, 0)

	redisClient, err := infrastructure.NewRedisClient(ctx, "redis://beatbet-redis:6379", 1)
	if err != nil {
		log.Printf("Error creating redis client: %v", err)
		return
	}

	binanceClient := binanceExchange.NewBinance(binanceExchange.Config{
		APIUrl:     binanceExchange.FuturesAPIURL,
		WSUrl:      binanceExchange.FuturesWSUrl,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		Name:       "binance",
	})
	binanceImporter := importer.NewImporter(binanceClient, mongoFactory)
	binanceImporter.WithMarketNotify(redisNotificator.NewRedisNotifier(redisClient, fmt.Sprintf("exchange:%s:market", binanceClient.GetName())))
	importers = append(importers, binanceImporter)

	bybitClient := bybitExchange.NewBybit(bybitExchange.Config{
		APIUrl:     bybitExchange.FuturesAPIURL,
		WSUrl:      bybitExchange.FuturesWSUrl,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		Name:       "bybit",
	})
	importers = append(importers, importer.NewImporter(bybitClient, mongoFactory))

	okxClient := okxExchange.NewOKX(okxExchange.Config{
		APIUrl:     okxExchange.FuturesAPIURL,
		WSUrl:      okxExchange.FuturesWSUrl,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		Name:       "okx",
	})
	importers = append(importers, importer.NewImporter(okxClient, mongoFactory))
	importers = importers[:1] // temporary use only binance

	var wg sync.WaitGroup
	wg.Add(len(importers))

	for _, i := range importers {
		go func(i *importer.Importer) {
			defer wg.Done()
			if err := i.StartImportLoop(ctx, time.Second); err != nil {
				log.Printf("Error starting import loop: %v", err)
			}
		}(i)
	}

	go func() {
		wg.Wait()
		fmt.Println("All importers have stopped!!!!!")
		stop()
	}()

	<-ctx.Done()
	log.Printf("Exiting...")
}
