# Exchange Data Importer

Simple and extensible service for importing market data (tickers and liquidations) from cryptocurrency exchanges. The app imports data every second and shows market averages. The app consists of two main components:
- **Importer**: Fetches market data from configured exchange
- **Notifier**: Processes and sends notifications about imported data in different formats

## Key Features

- Data import every second with market analytics
- Easy to extend with new exchanges or notification methods without changing existing code
- Main business logic covered with tests
- Configurable via environment variables or command-line flags

## Project Structure

```
/cmd
  /importer           # Main application entry point
/internal
  /bootstrap         # Application initialization and configuration
  /domain           # Core business entities and interfaces
  /importer         # Market data import implementation
  /infrastructure   # External integrations (exchanges, storage, notifications)
  /notifier         # Notification system and strategies
```

## Quick Start

Build and run:

```bash
# Build
go build -o exchange-importer cmd/importer/main.go

# Run with basic configuration (show market averages in console)
./exchange-importer --exchange.binance.enabled --notify.stdout.topics=TICK_INFO
```

Or use environment variables:

```bash
# Simple console output with Binance data
EXCHANGE_BINANCE_ENABLED=true
NOTIFY_STDOUT_TOPICS=TICK_INFO

# Optional: other exchanges available
# EXCHANGE_BYBIT_ENABLED=true
# EXCHANGE_OKX_ENABLED=true

# Optional: persistent storage
# REPOSITORY_SQLITE_ENABLED=true
# REPOSITORY_SQLITE_PATH=exchange.db
```

## Output Format For TICK_INFO Topic

When using TICK_INFO notifications, data is displayed in the following format:
```
TIME                 | MKTS |  Max10 % |  Min10 % |  AVG BUY |    LL5 |   LL60 |    SL2 |   SL10
```

All values represent averages across all tracked markets (trading pairs):
- `TIME`: Timestamp of the data
- `MKTS`: Number of markets being tracked
- `Max10 %`: Average maximum price change in last 10 minutes
- `Min10 %`: Average minimum price change in last 10 minutes
- `AVG BUY`: Average buy price change for the last tick
- `LL5`: Total long liquidations in last 5 seconds
- `LL60`: Total long liquidations in last 60 seconds
- `SL2`: Total short liquidations in last 2 seconds
- `SL10`: Total short liquidations in last 10 seconds