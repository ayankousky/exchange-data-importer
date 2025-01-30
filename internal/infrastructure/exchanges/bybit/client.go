// Package bybit provides a client for interacting with the Bybit futures exchange API
package bybit

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
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
)

// Config holds the configuration for the Bybit client
type Config struct {
	Name       string
	APIUrl     string
	WSUrl      string
	HTTPClient *http.Client
}

// Client implements a Bybit exchange client
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

// NewBybit creates a new Bybit client with the provided configuration
func NewBybit(cfg Config) *Client {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
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

	var response TickerResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decoding response from %s: %w", url, err)
	}

	if len(bc.tickersInfo.availableTickers) == 0 || time.Since(bc.tickersInfo.updatedAt) > DefaultTickersUpdateInterval {
		var availableTickers []string
		for _, ticker := range response.Result.List {
			availableTickers = append(availableTickers, ticker.Symbol)
		}
		bc.setAvailableTickers(availableTickers)
	}

	return convertTickers(response.Result.List, time.Unix(0, response.Time*int64(time.Millisecond))), nil
}

// convertTickers converts Bybit-specific ticker DTOs to normalized tickers
func convertTickers(bybitTickers []TickerDTO, eventAt time.Time) []exchanges.Ticker {
	tickers := make([]exchanges.Ticker, 0, len(bybitTickers))

	for _, bt := range bybitTickers {
		ticker, err := bt.toTicker()
		ticker.EventAt = eventAt
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
func (bc *Client) SubscribeLiquidations(ctx context.Context) (liquidations <-chan exchanges.Liquidation, errors <-chan error) {
	out := make(chan exchanges.Liquidation, DefaultChannelBuffer)
	errCh := make(chan error, DefaultChannelBuffer)

	go bc.handleLiquidationSubscription(ctx, out, errCh)

	return out, errCh
}

// handleLiquidationSubscription manages the websocket connection lifecycle
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
func (bc *Client) connectAndHandle(ctx context.Context, out chan<- exchanges.Liquidation, errCh chan<- error) error {
	conn, _, err := websocket.DefaultDialer.Dial(bc.wsURL, nil)
	if err != nil {
		return fmt.Errorf("websocket dial: %w", err)
	}
	defer conn.Close()

	availableTickers := bc.getAvailableTickers()
	if len(availableTickers) == 0 {
		log.Printf("Warning: no available tickers to subscribe (%s)", bc.GetName())
		return nil
	}

	// Subscribe to liquidations topic
	tickersToSubscribe := make([]string, 0, len(availableTickers))
	for _, ticker := range availableTickers {
		tickersToSubscribe = append(tickersToSubscribe, fmt.Sprintf("liquidation.%s", ticker))
	}
	subscribeMsg := map[string]interface{}{
		"op":     "subscribe",
		"req_id": "liquidations",
		"args":   tickersToSubscribe,
	}
	if err := conn.WriteJSON(subscribeMsg); err != nil {
		return fmt.Errorf("subscribing to liquidation topic: %w", err)
	}

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
	if !strings.HasPrefix(event.Topic, "liquidation") {
		return nil
	}

	liquidation, err := event.Data.toLiquidation()
	if err != nil {
		select {
		case errCh <- err:
		default:
			log.Printf("converting liquidation error: %v", err)
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

// setAvailableTickers updates the available tickers with proper locking
func (bc *Client) setAvailableTickers(tickers []string) {
	bc.tickersInfo.availableTickers = tickers
	bc.tickersInfo.updatedAt = time.Now()
}

// getAvailableTickers safely retrieves the available tickers
func (bc *Client) getAvailableTickers() []string {
	return append([]string{}, bc.tickersInfo.availableTickers...)
}
