package main

import (
	"context"
	"fmt"
	"github.com/ayankousky/exchange-data-importer/pkg/exchanges/binance"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	fmt.Println("App started !!")

	binanceClient := binance.NewBinance(binance.Config{
		APIUrl:     binance.FuturesAPIURL,
		WSUrl:      binance.FuturesWSUrl,
		HTTPClient: http.DefaultClient,
	})
	fmt.Println(binanceClient.GetName())
	result, err := binanceClient.FetchTickers(context.Background())
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(result)
	_, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fmt.Println("Exiting...")
}
