package importer

import (
	"context"
	"fmt"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/notify"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/telemetry"
	"go.uber.org/zap"
)

//go:generate moq --out mocks/repository_factory.go --pkg mocks --with-resets --skip-ensure . RepositoryFactory
//go:generate moq --out mocks/notifier.go --pkg mocks --with-resets --skip-ensure . NotifierService

const defaultTickInterval = time.Second // defines the default time interval between each tick operation in the import loop.

// RepositoryFactory is a contract for creating repositories
type RepositoryFactory interface {
	GetTickRepository(name string) (domain.TickRepository, error)
	GetLiquidationRepository(name string) (domain.LiquidationRepository, error)
}

// NotifierService represents the notifier service contract
type NotifierService interface {
	Subscribe(topic string, client notify.Client, strategy notify.Strategy)
	Notify(ctx context.Context, data any)
}

// Importer is responsible for importing data from an exchange and storing it in the database
type Importer struct {
	exchange              exchanges.Exchange
	tickRepository        domain.TickRepository
	liquidationRepository domain.LiquidationRepository

	tickHistory   *tickHistory
	tickerHistory *tickerHistoryMap

	notifier  NotifierService
	telemetry telemetry.Provider
	logger    *zap.Logger
}

// Config represents the configuration for initializing the importer
type Config struct {
	Exchange          exchanges.Exchange
	RepositoryFactory RepositoryFactory
	NotifierService   NotifierService
	Telemetry         telemetry.Provider
	Logger            *zap.Logger
}

// New creates a new Importer
func New(cfg *Config) *Importer {
	tickRepository, err := cfg.RepositoryFactory.GetTickRepository(cfg.Exchange.GetName())
	if err != nil {
		return nil
	}
	liquidationRepository, err := cfg.RepositoryFactory.GetLiquidationRepository(cfg.Exchange.GetName())
	if err != nil {
		return nil
	}
	return &Importer{
		exchange:              cfg.Exchange,
		tickRepository:        tickRepository,
		liquidationRepository: liquidationRepository,

		tickHistory:   newTickHistory(domain.MaxTickHistory),
		tickerHistory: newTickerHistoryMap(),

		notifier:  cfg.NotifierService,
		telemetry: cfg.Telemetry,
		logger:    cfg.Logger,
	}
}

// Start starts a loop that imports data from the exchange periodically.
func (i *Importer) Start(ctx context.Context) error {
	if err := i.startLiquidationsImport(ctx); err != nil {
		return fmt.Errorf("failed to start liquidations import: %w", err)
	}
	if err := i.startTickersImport(ctx); err != nil {
		return fmt.Errorf("failed to start tickers import: %w", err)
	}

	return nil
}

// LiquidationsImportOptions contains options for importing liquidations
type LiquidationsImportOptions struct{}

// StartLiquidationsImport starts importing liquidations from the exchange
func (i *Importer) startLiquidationsImport(ctx context.Context) error {
	liqChan, errChan := i.exchange.SubscribeLiquidations(ctx)
	if liqChan == nil || errChan == nil {
		i.logger.Error("Failed to subscribe to liquidations", zap.String("exchange", i.exchange.GetName()))
		return fmt.Errorf("failed to subscribe to liquidations")
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				i.logger.Info("Liquidation import stopped (context canceled).")
				return
			case liq := <-liqChan:
				// Convert the `exchanges.Liquidation` to your domain model
				domainLiq := i.convertLiquidationToDomain(liq)

				if err := domainLiq.Validate(); err != nil {
					i.logger.Error("Liquidation validation failed", zap.Error(err))
					continue
				}

				// Store it
				err := i.liquidationRepository.Create(ctx, domainLiq)
				if err != nil {
					i.logger.Error("Failed to store liquidation", zap.Error(err))
				}
			case err := <-errChan:
				i.telemetry.IncrementCounter(telemetryLiquidationsErrors, 1, fmt.Sprintf("exchange:%s", i.exchange.GetName()))
				i.logger.Error("Error on liquidation stream", zap.Error(err))
			}
		}
	}()
	return nil
}

// convertLiquidationToDomain converts the exchange Liquidation to a domain Liquidation
func (i *Importer) convertLiquidationToDomain(liq exchanges.Liquidation) domain.Liquidation {
	return domain.Liquidation{
		Order: domain.Order{
			Symbol:     domain.TickerName(liq.Symbol),
			EventAt:    liq.EventAt,
			Side:       domain.OrderSide(liq.Side),
			Price:      liq.Price,
			Quantity:   liq.Quantity,
			TotalPrice: liq.TotalPrice,
		},
		EventAt:  liq.EventAt,
		StoredAt: time.Now(),
	}
}

// StartTickersImport starts a loop that imports data from the exchange periodically.
func (i *Importer) startTickersImport(ctx context.Context) error {
	// Initialize the history data for calculating tick indicators
	if err := i.initHistory(ctx); err != nil {
		return fmt.Errorf("failed to init history: %w", err)
	}

	// Import should be started exactly at the beginning of the next second
	now := time.Now()
	nextSecond := now.Truncate(time.Second).Add(time.Second)
	time.Sleep(time.Until(nextSecond))

	// Start the import loop with the specified interval
	timeTicker := time.NewTicker(defaultTickInterval)
	defer timeTicker.Stop()

	i.logger.Info(i.generateImporterInfo())
	for {
		select {
		case <-ctx.Done():
			i.logger.Info("Context canceled, stopping import loop...")
			return ctx.Err()
		case <-timeTicker.C:
			// Attempt to import a single "tick" of data
			if err := i.importTick(ctx); err != nil {
				i.logger.Error("Error importing tick", zap.Error(err))
			}
		}
	}
}

// GetInfo returns a string with the current state of the Importer
func (i *Importer) generateImporterInfo() string {
	var info string
	info += "\n________________________________________________________________________________\n"
	info += fmt.Sprintf("exchange: %s\n", i.exchange.GetName())
	info += fmt.Sprintf("Tick history length: %d\n", i.tickHistory.Len())
	info += fmt.Sprintf("Ticker history length: %d\n", len(i.tickerHistory.data))

	info += "________________________________________________________________________________\n"

	return info
}
