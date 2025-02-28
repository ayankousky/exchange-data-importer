package binance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBinance(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{
			name: "creates client with config",
			cfg: Config{
				Name:       "test-binance",
				APIUrl:     "http://api.test",
				WSUrl:      "ws://ws.test",
				HTTPClient: http.DefaultClient,
			},
			want: "test-binance",
		},
		{
			name: "empty config",
			cfg:  Config{},
			want: "Binance perpetual",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewBinance(tt.cfg)
			assert.Equal(t, tt.want, client.GetName())
		})
	}
}

func TestClient_FetchTickers(t *testing.T) {
	tests := []struct {
		name          string
		response      any
		statusCode    int
		expectError   bool
		wantTickers   []exchanges.Ticker
		contextCancel bool
	}{
		{
			name: "successful fetch",
			response: []TickerDTO{
				{
					Symbol:      "BTCUSDT",
					BidPrice:    "50000.50",
					BidQuantity: "1.5",
					AskPrice:    "50000.75",
					AskQuantity: "2.5",
					Time:        1635739200000,
				},
			},
			statusCode:  http.StatusOK,
			expectError: false,
			wantTickers: []exchanges.Ticker{
				{
					Symbol:      "BTCUSDT",
					BidPrice:    50000.50,
					BidQuantity: 1.5,
					AskPrice:    50000.75,
					AskQuantity: 2.5,
					EventAt:     time.UnixMilli(1635739200000),
				},
			},
		},
		{
			name:          "context cancelled",
			response:      []TickerDTO{},
			contextCancel: true,
			expectError:   true,
		},
		{
			name:        "server error",
			response:    map[string]string{"error": "internal error"},
			statusCode:  http.StatusInternalServerError,
			expectError: true,
		},
		{
			name:        "invalid response",
			response:    "invalid json",
			statusCode:  http.StatusOK,
			expectError: true,
		},
		{
			name: "invalid ticker data",
			response: []TickerDTO{
				{
					Symbol:      "BTCUSDT",
					BidPrice:    "invalid",
					BidQuantity: "1.5",
					AskPrice:    "50000.75",
					AskQuantity: "2.5",
				},
			},
			statusCode:  http.StatusOK,
			expectError: false,
			wantTickers: []exchanges.Ticker{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.statusCode != 0 {
					w.WriteHeader(tt.statusCode)
				}
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			// Setup client
			client := NewBinance(Config{
				Name:       "test",
				APIUrl:     server.URL,
				HTTPClient: http.DefaultClient,
			})

			// Setup context
			ctx := context.Background()
			if tt.contextCancel {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			// Execute test
			got, err := client.FetchTickers(ctx)

			// Assert results
			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantTickers, got)
		})
	}
}

func TestClient_SubscribeLiquidations(t *testing.T) {
	tests := []struct {
		name          string
		messages      []string
		wantCount     int
		expectError   bool
		contextCancel bool
	}{
		{
			name: "successful subscription",
			messages: []string{
				`{
					"e": "forceOrder",
					"E": 1635739200000,
					"o": {
						"s": "BTCUSDT",
						"S": "SELL",
						"o": "LIMIT",
						"f": "IOC",
						"q": "0.001",
						"p": "50000.50",
						"ap": "0",
						"X": "FILLED",
						"l": "0.001",
						"T": 1635739200000
					}
				}`,
			},
			wantCount:   1,
			expectError: false,
		},
		{
			name:          "context cancelled",
			messages:      []string{},
			expectError:   true,
			wantCount:     0,
			contextCancel: true,
		},
		{
			name: "invalid message",
			messages: []string{
				`invalid json`,
			},
			wantCount:   0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Channel to coordinate the start of the test
			wsConnected := make(chan struct{})

			// Create WebSocket test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				upgrader := websocket.Upgrader{
					CheckOrigin: func(r *http.Request) bool { return true },
				}

				// Upgrade the connection
				ws, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					t.Logf("upgrade error: %v", err)
					return
				}
				defer ws.Close()

				// Signal that WebSocket is ready
				close(wsConnected)

				// Send messages or handle context cancellation
				for _, msg := range tt.messages {
					select {
					case <-r.Context().Done():
						return
					default:
						if err := ws.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
							t.Logf("write message error: %v", err)
							return
						}
						time.Sleep(10 * time.Millisecond)
					}
				}

				// Keep connection open until context is done
				<-r.Context().Done()
			}))
			defer server.Close()

			// Setup client with WebSocket URL
			wsURL := "ws" + server.URL[4:]
			client := NewBinance(Config{
				Name:  "test",
				WSUrl: wsURL,
			})

			// Create context based on test case
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			// If test requires immediate cancellation, cancel before subscribing
			if tt.contextCancel {
				cancel()
			}

			// Start subscription
			liquidations, errors := client.SubscribeLiquidations(ctx)

			// Wait for WebSocket connection or timeout
			if !tt.contextCancel {
				select {
				case <-wsConnected:
				case <-time.After(time.Second):
					t.Fatal("timeout waiting for websocket connection")
				}
			}

			// Track messages and errors
			var count int
			var lastError error
			done := make(chan struct{})

			go func() {
				defer close(done)
				for {
					select {
					case liq, ok := <-liquidations:
						if !ok {
							return
						}
						require.NotEmpty(t, liq.Symbol)
						require.NotZero(t, liq.Price)
						require.NotZero(t, liq.Quantity)
						count++
					case err, ok := <-errors:
						if !ok {
							return
						}
						lastError = err
					case <-ctx.Done():
						if tt.expectError && lastError == nil {
							lastError = ctx.Err()
						}
						return
					}
				}
			}()

			// Wait for completion or timeout
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				t.Fatal("test timed out")
			}

			// Verify results
			if tt.expectError {
				assert.Error(t, lastError)
			} else {
				assert.NoError(t, lastError)
			}
			assert.Equal(t, tt.wantCount, count)
		})
	}
}

func TestConvertTickers(t *testing.T) {
	tests := []struct {
		name      string
		input     []TickerDTO
		want      []exchanges.Ticker
		wantCount int
	}{
		{
			name: "all valid tickers",
			input: []TickerDTO{
				{
					Symbol:      "BTCUSDT",
					BidPrice:    "50000.50",
					BidQuantity: "1.5",
					AskPrice:    "50000.75",
					AskQuantity: "2.5",
					Time:        1635739200000,
				},
				{
					Symbol:      "ETHUSDT",
					BidPrice:    "3000.50",
					BidQuantity: "10.5",
					AskPrice:    "3000.75",
					AskQuantity: "12.5",
					Time:        1635739200000,
				},
			},
			wantCount: 2,
			want: []exchanges.Ticker{
				{
					Symbol:      "BTCUSDT",
					BidPrice:    50000.50,
					BidQuantity: 1.5,
					AskPrice:    50000.75,
					AskQuantity: 2.5,
					EventAt:     time.UnixMilli(1635739200000),
				},
				{
					Symbol:      "ETHUSDT",
					BidPrice:    3000.50,
					BidQuantity: 10.5,
					AskPrice:    3000.75,
					AskQuantity: 12.5,
					EventAt:     time.UnixMilli(1635739200000),
				},
			},
		},
		{
			name: "some invalid tickers are skipped",
			input: []TickerDTO{
				{
					Symbol:      "BTCUSDT",
					BidPrice:    "50000.50",
					BidQuantity: "1.5",
					AskPrice:    "50000.75",
					AskQuantity: "2.5",
					Time:        1635739200000,
				},
				{
					Symbol:      "INVALID",
					BidPrice:    "invalid",
					BidQuantity: "1.5",
					AskPrice:    "50000.75",
					AskQuantity: "2.5",
					Time:        1635739200000,
				},
			},
			wantCount: 1,
			want: []exchanges.Ticker{
				{
					Symbol:      "BTCUSDT",
					BidPrice:    50000.50,
					BidQuantity: 1.5,
					AskPrice:    50000.75,
					AskQuantity: 2.5,
					EventAt:     time.UnixMilli(1635739200000),
				},
			},
		},
		{
			name:      "empty input",
			input:     []TickerDTO{},
			want:      []exchanges.Ticker{},
			wantCount: 0,
		},
		{
			name: "all invalid tickers",
			input: []TickerDTO{
				{
					Symbol:      "INVALID1",
					BidPrice:    "invalid",
					BidQuantity: "1.5",
					AskPrice:    "50000.75",
					AskQuantity: "2.5",
				},
				{
					Symbol:      "INVALID2",
					BidPrice:    "50000.50",
					BidQuantity: "invalid",
					AskPrice:    "50000.75",
					AskQuantity: "2.5",
				},
			},
			want:      []exchanges.Ticker{},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertTickers(tt.input)
			assert.Equal(t, tt.wantCount, len(got))
			assert.Equal(t, tt.want, got)
		})
	}
}
