package binance

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/ayankousky/exchange-data-importer/pkg/exchanges"
	"net/http"
	"strconv"
)

// Config is a configuration for Binance client
type Config struct {
	Name       string
	APIUrl     string
	WSUrl      string
	HTTPClient *http.Client
}

// Client is a Binance client to access Binance data
type Client struct {
	name       string
	httpURL    string
	wsURL      string
	httpClient *http.Client
}

// NewBinance creates a new Binance client
func NewBinance(cfg Config) *Client {
	return &Client{
		name:       cfg.Name,
		httpURL:    cfg.APIUrl,
		wsURL:      cfg.WSUrl,
		httpClient: cfg.HTTPClient,
	}
}

// GetName returns the name of the client
// f.i. when you need a separate db collection or for logging
func (bc *Client) GetName() string {
	return bc.name
}

// FetchTickers fetches tickers from Binance
func (bc *Client) FetchTickers(ctx context.Context) ([]exchanges.Ticker, error) {
	url := bc.httpURL + FetchTickersData
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := bc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("unexpected status code %d" + resp.Status)
	}

	var binanceTickers []struct {
		Symbol      string `json:"symbol"`
		BidPrice    string `json:"bidPrice"`
		BidQuantity string `json:"bidQty"`
		AskPrice    string `json:"askPrice"`
		AskQuantity string `json:"askQty"`
		Time        int64  `json:"time"`
		LastUpdated int64  `json:"lastUpdateId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&binanceTickers); err != nil {
		return nil, err
	}

	tickers := make([]exchanges.Ticker, 0, len(binanceTickers))
	for _, bt := range binanceTickers {
		bidPrice, err := strconv.ParseFloat(bt.BidPrice, 64)
		if err != nil {
			return nil, err
		}
		askPrice, err := strconv.ParseFloat(bt.AskPrice, 64)
		if err != nil {
			return nil, err
		}
		bidQuantity, err := strconv.ParseFloat(bt.BidQuantity, 64)
		if err != nil {
			return nil, err
		}
		askQuantity, err := strconv.ParseFloat(bt.AskQuantity, 64)
		if err != nil {
			return nil, err
		}
		tickers = append(tickers, exchanges.Ticker{
			Symbol:      bt.Symbol,
			BidPrice:    bidPrice,
			AskPrice:    askPrice,
			BidQuantity: bidQuantity,
			AskQuantity: askQuantity,
		})
	}

	return tickers, nil
}
