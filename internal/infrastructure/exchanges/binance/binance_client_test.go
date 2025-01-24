package binance

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_GetName(t *testing.T) {
	cfg := Config{
		Name:       "Binance",
		APIUrl:     "http://mockapi.com/",
		HTTPClient: http.DefaultClient,
	}

	client := NewBinance(cfg)
	assert.Equal(t, "Binance", client.GetName())
}

func TestClient_FetchTickers(t *testing.T) {
	testCases := []struct {
		name               string
		responseStatusCode int
		responseBody       string
		expectedError      bool
		expectedTickers    []exchanges.Ticker
	}{
		{
			name:               "valid response",
			responseStatusCode: http.StatusOK,
			responseBody: `[
				{"symbol": "BTCUSDT", "bidPrice": "40000.0", "bidQty": "1.5", "askPrice": "40010.0", "askQty": "1.0", "time": 1735286548259, "lastUpdateId": 987654321},
				{"symbol": "ETHUSDT", "bidPrice": "2500.0", "bidQty": "2.0", "askPrice": "2510.0", "askQty": "2.5", "time": 1735286548259, "lastUpdateId": 987654322}
			]`,
			expectedError: false,
			expectedTickers: []exchanges.Ticker{
				{Symbol: "BTCUSDT", BidPrice: 40000.0, AskPrice: 40010.0, BidQuantity: 1.5, AskQuantity: 1.0, EventAt: time.Unix(0, 1735286548259*int64(time.Millisecond))},
				{Symbol: "ETHUSDT", BidPrice: 2500.0, AskPrice: 2510.0, BidQuantity: 2.0, AskQuantity: 2.5, EventAt: time.Unix(0, 1735286548259*int64(time.Millisecond))},
			},
		},
		{
			name:               "invalid response",
			responseStatusCode: http.StatusInternalServerError,
			responseBody:       "",
			expectedError:      true,
			expectedTickers:    nil,
		},
		{
			name:               "malformed JSON",
			responseStatusCode: http.StatusOK,
			responseBody:       "[{\"symbol\": \"BTCUSDT\", \"bidPrice\": \"not-a-number\"}]",
			expectedError:      true,
			expectedTickers:    nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.responseStatusCode)
				w.Write([]byte(tc.responseBody))
			}))
			defer server.Close()

			cfg := Config{
				Name:       "Binance",
				APIUrl:     server.URL,
				HTTPClient: http.DefaultClient,
			}
			client := NewBinance(cfg)

			ctx := context.Background()
			tickers, err := client.FetchTickers(ctx)

			if tc.expectedError {
				require.Error(t, err)
				assert.Nil(t, tickers)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedTickers, tickers)
			}
		})
	}
}
