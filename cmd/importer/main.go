package main

import (
	"context"
	"fmt"
	"github.com/ayankousky/exchange-data-importer/internal/importer"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/db"
	binance2 "github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges/binance"
	"github.com/ayankousky/exchange-data-importer/internal/repository/mongo"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	fmt.Println("App started !!")

	binanceClient := binance2.NewBinance(binance2.Config{
		APIUrl:     binance2.FuturesAPIURL,
		WSUrl:      binance2.FuturesWSUrl,
		HTTPClient: http.DefaultClient,
		Name:       "binance",
	})

	// Create a new mongo client
	mongoClient, err := db.NewMongoClient("mongodb://beatbet-db-mongo:27017")
	if err != nil {
		fmt.Println(err)
	}

	mongoFactory, _ := mongo.NewMongoRepoFactory(mongoClient)

	binanceImporter := importer.NewImporter(binanceClient, mongoFactory)
	binanceImporter.StartImportEverySecond()

	_, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fmt.Println("Exiting...")
}
