// Package binance provides a client for interacting with the Binance exchange API
package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges"
	"github.com/gorilla/websocket"
)

const (
	// DefaultReconnectDelay is the time to wait before attempting to reconnect to websocket
	DefaultReconnectDelay = 5 * time.Second

	// DefaultWebsocketTimeout is the read deadline timeout for websocket connections
	DefaultWebsocketTimeout = 60 * time.Second

	// DefaultChannelBuffer is the default size for channels
	DefaultChannelBuffer = 100
)

// Config holds the configuration for the Binance client
type Config struct {
	// Name identifies the client instance
	Name string

	// APIUrl is the base URL for REST API endpoints
	APIUrl string

	// WSUrl is the websocket endpoint URL
	WSUrl string

	// HTTPClient is a custom HTTP client for making requests
	HTTPClient *http.Client
}

// Client implements a Binance exchange client
type Client struct {
	name       string
	httpURL    string
	wsURL      string
	httpClient *http.Client
}

// NewBinance creates a new Binance client with the provided configuration
func NewBinance(cfg Config) *Client {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}
	if cfg.WSUrl == "" {
		cfg.WSUrl = FuturesWSUrl
	}
	if cfg.APIUrl == "" {
		cfg.APIUrl = FuturesAPIURL
	}

	return &Client{
		name:       cfg.Name,
		httpURL:    cfg.APIUrl,
		wsURL:      cfg.WSUrl,
		httpClient: cfg.HTTPClient,
	}
}

//------------------------------------------------------------------------------
// Fetch Tickers API Methods
//------------------------------------------------------------------------------

// FetchTickers retrieves current ticker information for all trading pairs
// It returns a slice of normalized Ticker objects or an error if the request fails
func (bc *Client) FetchTickers(ctx context.Context) ([]exchanges.Ticker, error) {
	url := bc.httpURL + FetchTickersData

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating request for %s: %w", url, err)
	}

	resp, err := bc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request for %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, resp.Status)
	}

	var binanceTickers []TickerDTO
	err = json.NewDecoder(resp.Body).Decode(&binanceTickers)
	if err != nil {
		return nil, fmt.Errorf("decoding response from %s: %w", url, err)
	}

	// Validate tickers against market data
	filteredTickers, err := FilterTickers(binanceTickers)
	if err != nil {
		return nil, fmt.Errorf("validating market data: %w", err)
	}

	return convertTickers(filteredTickers), nil
}

// convertTickers converts Binance-specific ticker DTOs to normalized tickers
func convertTickers(binanceTickers []TickerDTO) []exchanges.Ticker {
	tickers := make([]exchanges.Ticker, 0, len(binanceTickers))

	for _, bt := range binanceTickers {
		ticker, err := bt.toTicker()
		if err != nil {
			log.Printf("Warning: failed to convert ticker: %v", err)
			continue
		}
		tickers = append(tickers, ticker)
	}

	return tickers
}

//------------------------------------------------------------------------------
// Fetch Liquidations API Methods
//------------------------------------------------------------------------------

// SubscribeLiquidations initiates a websocket connection to receive liquidation events
// It returns two channels: one for receiving liquidation events and one for errors
func (bc *Client) SubscribeLiquidations(ctx context.Context) (liquidations <-chan exchanges.Liquidation, errors <-chan error) {
	out := make(chan exchanges.Liquidation, DefaultChannelBuffer)
	errCh := make(chan error, DefaultChannelBuffer)

	go bc.handleLiquidationSubscription(ctx, out, errCh)

	return out, errCh
}

// handleLiquidationSubscription manages the websocket connection lifecycle
// It continuously attempts to maintain a connection and handles errors gracefully
func (bc *Client) handleLiquidationSubscription(ctx context.Context, out chan<- exchanges.Liquidation, errCh chan<- error) {
	defer close(out)
	defer close(errCh)

	for {
		if err := bc.connectAndHandle(ctx, out, errCh); err != nil {
			select {
			case errCh <- fmt.Errorf("websocket error: %w", err):
			default:
				log.Printf("Error: %v", err)
			}
		}

		select {
		case <-ctx.Done():
			return
		default:
			log.Printf("Reconnecting in %s...", DefaultReconnectDelay)
			time.Sleep(DefaultReconnectDelay)
		}
	}
}

// connectAndHandle establishes and manages a single websocket connection
// It connects and reads messages from the websocket
func (bc *Client) connectAndHandle(ctx context.Context, out chan<- exchanges.Liquidation, errCh chan<- error) error {
	conn, _, err := websocket.DefaultDialer.Dial(bc.wsURL, nil)
	if err != nil {
		return fmt.Errorf("websocket dial: %w", err)
	}
	defer conn.Close()

	return bc.readMessages(ctx, conn, out, errCh)
}

// readMessages reads and processes messages from the websocket connection
func (bc *Client) readMessages(ctx context.Context, conn *websocket.Conn, out chan<- exchanges.Liquidation, errCh chan<- error) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			if err := conn.SetReadDeadline(time.Now().Add(DefaultWebsocketTimeout)); err != nil {
				return fmt.Errorf("setting read deadline: %w", err)
			}

			_, msg, err := conn.ReadMessage()
			if err != nil {
				return fmt.Errorf("reading message: %w", err)
			}

			if err := bc.processMessage(ctx, msg, out, errCh); err != nil {
				log.Printf("Warning: message processing error: %v", err)
			}
		}
	}
}

// processMessage handles the deserialization and conversion of websocket messages
func (bc *Client) processMessage(ctx context.Context, msg []byte, out chan<- exchanges.Liquidation, errCh chan<- error) error {
	var event LiquidationDTO
	if err := json.Unmarshal(msg, &event); err != nil {
		select {
		case errCh <- err:
		default:
			log.Printf("unmarshaling message error: %v", err)
		}
		return err
	}

	liquidation, err := event.toLiquidation()
	if err != nil {
		select {
		case errCh <- err:
		default:
			log.Printf("converting liquidation error:: %v", err)
		}
		return err
	}

	select {
	case out <- liquidation:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context canceled")
	}
}

//------------------------------------------------------------------------------
// Other methods
//------------------------------------------------------------------------------

// GetName returns the name of the client instance
func (bc *Client) GetName() string {
	return bc.name
}
