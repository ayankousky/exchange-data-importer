package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/importer"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/db"
	binanceExchange "github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges/binance"
	"github.com/ayankousky/exchange-data-importer/internal/repository/mongo"
)

func main() {
	log.Printf("Starting importer...")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	binanceClient := binanceExchange.NewBinance(binanceExchange.Config{
		APIUrl:     binanceExchange.FuturesAPIURL,
		WSUrl:      binanceExchange.FuturesWSUrl,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		Name:       "binance",
	})

	// Create a new mongo client
	mongoClient, err := db.NewMongoClient("mongodb://beatbet-db-mongo:27017")
	if err != nil {
		log.Printf("Error creating mongo client: %v", err)
		return
	}

	mongoFactory, err := mongo.NewMongoRepoFactory(mongoClient)
	if err != nil {
		log.Printf("Error creating mongo factory: %v", err)
		return
	}

	binanceImporter := importer.NewImporter(binanceClient, mongoFactory)
	if err := binanceImporter.StartImportLoop(ctx, time.Second); err != nil {
		log.Printf("Error starting import loop: %v", err)
		return
	}

	log.Printf("Exiting...")
}
