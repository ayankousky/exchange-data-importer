package importer

import (
	"context"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/notify"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/telemetry"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
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

	importTicks        bool
	importLiquidations bool
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

// WithTicksImport enables tick import for this importer
func (i *Importer) WithTicksImport() *Importer {
	if i.tickRepository == nil {
		i.importTicks = false
		i.logger.Error("No tick repository found. Skipping ticks import")
		return i
	}
	i.importTicks = true
	return i
}

// WithLiquidationsImport enables liquidations import for this importer
func (i *Importer) WithLiquidationsImport() *Importer {
	if i.liquidationRepository == nil {
		i.importLiquidations = false
		i.logger.Error("No liquidation repository found. Skipping liquidations import")
		return i
	}
	i.importLiquidations = true
	return i
}

// Run starts all configured import processes in a non-blocking manner
func (i *Importer) Run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)

	// Start importing liquidations if the process is enabled
	if i.importLiquidations {
		eg.Go(func() error {
			return i.runLiquidationsImport(ctx)
		})
	}

	// Start importing ticks if the process is enabled
	if i.importTicks {
		eg.Go(func() error {
			return i.runTickersImport(ctx)
		})
	}

	return eg.Wait()
}
