package importer

// Telemetry constants for counters
const (
	// telemetryLiquidationsErrors tracks the number of errors encountered during liquidation stream processing
	telemetryLiquidationsErrors = "liquidations.errors"

	// telemetryTickFetchErrors counts errors that occur when fetching tickers from the exchange
	telemetryTickFetchErrors = "tick.fetch.errors"
)

// Telemetry constants for timings
const (
	// telemetryTickFetchDuration measures the time taken to fetch tickers from the exchange
	telemetryTickFetchDuration = "tick.fetch.duration"

	// telemetryTickBuildSetLiquidations tracks time spent populating liquidation data in a tick
	telemetryTickBuildSetLiquidations = "tick.build.set_tick_liquidations"

	// telemetryTickCalculateIndicators measures time spent calculating tick indicators from history
	telemetryTickCalculateIndicators = "tick.calculate_indicators.duration"
)

// Telemetry constants for gauges
const (
	// telemetryTickFetchTickersCount tracks the number of tickers fetched from the exchange
	telemetryTickFetchTickersCount = "tick.fetch.tickers_count"

	// telemetryTickBuildTickersProcessed measures the number of tickers successfully processed in a tick
	telemetryTickBuildTickersProcessed = "tick.build.tickers_processed"
)

// Telemetry constants for spans
const (
	// telemetrySpanImportTick represents the overall process of importing a single tick
	telemetrySpanImportTick = "importTick"

	// telemetrySpanFetchTickers tracks the operation of fetching tickers from an exchange
	telemetrySpanFetchTickers = "fetchTickers"

	// telemetrySpanBuildTick represents the process of building a tick from fetched data
	telemetrySpanBuildTick = "buildTick"
)
