package binance

const (
	// FuturesAPIURL is the base URL for the Binance Futures API
	FuturesAPIURL = "https://fapi.binance.com/fapi/v1"
	// FuturesWSUrl is the base URL for the Binance Futures Websocket API
	FuturesWSUrl = "wss://fstream.binance.com/ws"

	// FetchTickersData is the endpoint to fetch tickers data
	FetchTickersData = "/ticker/bookTicker"
)

const (
	// ErrorRequestFailed is the error message when a request fails
	ErrorRequestFailed = "request failed"
)
