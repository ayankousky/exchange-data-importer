// Package okx provides a client for interacting with the OKX futures exchange API
package okx

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
	DefaultWebsocketTimeout = 120 * time.Second

	// DefaultChannelBuffer is the default size for channels
	DefaultChannelBuffer = 100

	// DefaultTickersUpdateInterval is the interval to update available tickers
	DefaultTickersUpdateInterval = 5 * time.Minute

	// FuturesAPIURL is the base URL for the OKX Futures API
	FuturesAPIURL = "https://www.okx.com/api/v5"

	// FuturesWSUrl is the base URL for the OKX Futures Websocket API
	FuturesWSUrl = "wss://ws.okx.com:8443/ws/v5/public"

	// FetchTickersData is the endpoint to fetch tickers data
	FetchTickersData = "/market/tickers?instType=SWAP"
)

// Config holds the configuration for the OKX client
type Config struct {
	Name       string
	APIUrl     string
	WSUrl      string
	HTTPClient *http.Client
}

// Client implements an OKX exchange client
type Client struct {
	name       string
	httpURL    string
	wsURL      string
	httpClient *http.Client

	tickersInfo struct {
		availableTickers []string
		updatedAt        time.Time
	}
}

// NewOKX creates a new OKX client with the provided configuration
func NewOKX(cfg Config) *Client {
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
func (oc *Client) FetchTickers(ctx context.Context) ([]exchanges.Ticker, error) {
	url := oc.httpURL + FetchTickersData

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating request for %s: %w", url, err)
	}

	resp, err := oc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request for %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, resp.Status)
	}

	var response TickerResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decoding response from %s: %w", url, err)
	}

	if len(oc.tickersInfo.availableTickers) == 0 || time.Since(oc.tickersInfo.updatedAt) > DefaultTickersUpdateInterval {
		var availableTickers []string
		for _, ticker := range response.Data {
			availableTickers = append(availableTickers, ticker.InstID)
		}
		oc.setAvailableTickers(availableTickers)
	}

	return convertTickers(response.Data), nil
}

// convertTickers converts OKX-specific ticker DTOs to normalized tickers
func convertTickers(okxTickers []TickerDTO) []exchanges.Ticker {
	tickers := make([]exchanges.Ticker, 0, len(okxTickers))

	for _, ot := range okxTickers {
		ticker, err := ot.toTicker()
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
func (oc *Client) SubscribeLiquidations(ctx context.Context) (liquidations <-chan exchanges.Liquidation, errors <-chan error) {
	out := make(chan exchanges.Liquidation, DefaultChannelBuffer)
	errCh := make(chan error, DefaultChannelBuffer)

	go oc.handleLiquidationSubscription(ctx, out, errCh)

	return out, errCh
}

// handleLiquidationSubscription manages the websocket connection lifecycle
func (oc *Client) handleLiquidationSubscription(ctx context.Context, out chan<- exchanges.Liquidation, errCh chan<- error) {
	defer close(out)
	defer close(errCh)

	for {
		if err := oc.connectAndHandle(ctx, out, errCh); err != nil {
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
func (oc *Client) connectAndHandle(ctx context.Context, out chan<- exchanges.Liquidation, errCh chan<- error) error {
	conn, _, err := websocket.DefaultDialer.Dial(oc.wsURL, nil)
	if err != nil {
		return fmt.Errorf("websocket dial: %w", err)
	}
	defer conn.Close()

	availableTickers := oc.getAvailableTickers()
	if len(availableTickers) == 0 {
		return nil
	}

	subscribeMsg := map[string]interface{}{
		"op": "subscribe",
		"args": []interface{}{
			map[string]interface{}{
				"channel":  "liquidation-orders",
				"instType": "SWAP",
			},
		},
	}
	if err := conn.WriteJSON(subscribeMsg); err != nil {
		return fmt.Errorf("subscribing to liquidation channel: %w", err)
	}

	return oc.readMessages(ctx, conn, out, errCh)
}

// readMessages reads and processes messages from the websocket connection
func (oc *Client) readMessages(ctx context.Context, conn *websocket.Conn, out chan<- exchanges.Liquidation, errCh chan<- error) error {
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

			if err := oc.processMessage(ctx, msg, out, errCh); err != nil {
				log.Printf("Warning: message processing error: %v", err)
			}
		}
	}
}

// processMessage handles the deserialization and conversion of websocket messages
func (oc *Client) processMessage(ctx context.Context, msg []byte, out chan<- exchanges.Liquidation, errCh chan<- error) error {
	var event LiquidationEvent
	if err := json.Unmarshal(msg, &event); err != nil {
		select {
		case errCh <- err:
		default:
			log.Printf("unmarshaling message error: %v", err)
		}
		return err
	}

	// Skip non-liquidation messages
	if event.Arg.Channel != "liquidation-orders" {
		return nil
	}
	if len(event.Data) == 0 {
		return nil
	}

	for _, data := range event.Data {
		liquidation, err := data.toLiquidation()
		if err != nil {
			select {
			case errCh <- err:
			default:
				log.Printf("converting liquidation error: %v", err)
			}
			continue
		}

		select {
		case out <- liquidation:
		case <-ctx.Done():
			return fmt.Errorf("context canceled")
		}
	}

	return nil
}

//------------------------------------------------------------------------------
// Other methods
//------------------------------------------------------------------------------

// GetName returns the name of the client instance
func (oc *Client) GetName() string {
	return oc.name
}

// setAvailableTickers updates the available tickers with proper locking
func (oc *Client) setAvailableTickers(tickers []string) {
	oc.tickersInfo.availableTickers = tickers
	oc.tickersInfo.updatedAt = time.Now()
}

// getAvailableTickers safely retrieves the available tickers
func (oc *Client) getAvailableTickers() []string {
	return append([]string{}, oc.tickersInfo.availableTickers...)
}
