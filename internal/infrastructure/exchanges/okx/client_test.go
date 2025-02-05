package okx

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

func TestNewOKX(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{
			name: "creates client with config",
			cfg: Config{
				Name:       "test-okx",
				APIUrl:     "http://api.test",
				WSUrl:      "ws://ws.test",
				HTTPClient: http.DefaultClient,
			},
			want: "test-okx",
		},
		{
			name: "empty config",
			cfg:  Config{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewOKX(tt.cfg)
			assert.Equal(t, tt.want, client.GetName())
		})
	}
}

func TestClient_FetchTickers(t *testing.T) {
	tests := []struct {
		name          string
		response      interface{}
		statusCode    int
		expectError   bool
		wantTickers   []exchanges.Ticker
		contextCancel bool
	}{
		{
			name: "successful fetch",
			response: TickerResponse{
				Code: "0",
				Data: []TickerDTO{
					{
						InstID:      "BTC-USDT-SWAP",
						LastPrice:   "50000.50",
						BidPrice:    "50000.25",
						BidQuantity: "1.5",
						AskPrice:    "50000.75",
						AskQuantity: "2.5",
						Timestamp:   "1635739200000",
					},
				},
			},
			statusCode:  http.StatusOK,
			expectError: false,
			wantTickers: []exchanges.Ticker{
				{
					Symbol:      "BTC-USDT-SWAP",
					BidPrice:    50000.25,
					BidQuantity: 1.5,
					AskPrice:    50000.75,
					AskQuantity: 2.5,
					EventAt:     time.Unix(0, 1635739200000*int64(time.Millisecond)),
				},
			},
		},
		{
			name:          "context cancelled",
			response:      TickerResponse{},
			contextCancel: true,
			expectError:   true,
		},
		{
			name: "invalid response data",
			response: TickerResponse{
				Code: "0",
				Data: []TickerDTO{
					{
						InstID:      "BTC-USDT-SWAP",
						BidPrice:    "invalid",
						BidQuantity: "1.5",
						AskPrice:    "50000.75",
						AskQuantity: "2.5",
					},
				},
			},
			statusCode:  http.StatusOK,
			expectError: false,
			wantTickers: []exchanges.Ticker{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.statusCode != 0 {
					w.WriteHeader(tt.statusCode)
				}
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			client := NewOKX(Config{
				Name:       "test",
				APIUrl:     server.URL,
				HTTPClient: http.DefaultClient,
			})

			ctx := context.Background()
			if tt.contextCancel {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			got, err := client.FetchTickers(ctx)
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
		name             string
		messages         []string
		availableTickers []string
		wantCount        int
		expectError      bool
		contextCancel    bool
		skipTickerSetup  bool
	}{
		{
			name: "successful subscription",
			messages: []string{
				`{
					"arg": {
						"channel": "liquidation-orders",
						"instType": "SWAP"
					},
					"data": [{
						"details": [{
							"side": "sell",
							"sz": "0.001",
							"ts": "1635739200000",
							"bkPx": "50000.50"
						}],
						"instId": "BTC-USDT-SWAP"
					}]
				}`,
			},
			availableTickers: []string{"BTC-USDT-SWAP"},
			wantCount:        1,
			expectError:      false,
		},
		{
			name:            "no available tickers",
			messages:        []string{},
			skipTickerSetup: true,
			wantCount:       0,
			expectError:     false,
		},
		{
			name:             "context cancelled",
			messages:         []string{},
			availableTickers: []string{"BTC-USDT-SWAP"},
			expectError:      true,
			wantCount:        0,
			contextCancel:    true,
		},
		{
			name: "invalid message",
			messages: []string{
				`invalid json`,
			},
			availableTickers: []string{"BTC-USDT-SWAP"},
			wantCount:        0,
			expectError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wsConnected := make(chan struct{})

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				upgrader := websocket.Upgrader{
					CheckOrigin: func(r *http.Request) bool { return true },
				}

				ws, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					t.Logf("upgrade error: %v", err)
					return
				}
				defer ws.Close()

				close(wsConnected)

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

				<-r.Context().Done()
			}))
			defer server.Close()

			wsURL := "ws" + server.URL[4:]
			client := NewOKX(Config{
				Name:  "test",
				WSUrl: wsURL,
			})

			if !tt.skipTickerSetup {
				client.setAvailableTickers(tt.availableTickers)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
			defer cancel()

			if tt.contextCancel {
				cancel()
			}

			liquidations, errors := client.SubscribeLiquidations(ctx)

			if !tt.contextCancel {
				select {
				case <-wsConnected:
				case <-time.After(time.Second):
					t.Fatal("timeout waiting for websocket connection")
				}
			}

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

			select {
			case <-done:
			case <-time.After(3 * time.Second):
				t.Fatal("test timed out")
			}

			if tt.expectError {
				assert.Error(t, lastError)
			} else {
				assert.NoError(t, lastError)
			}
			assert.Equal(t, tt.wantCount, count)
		})
	}
}
