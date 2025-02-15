package importer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/notify"
	"github.com/ayankousky/exchange-data-importer/pkg/utils"
	"go.uber.org/zap"
)

//go:generate moq --out mocks/repository_factory.go --pkg mocks --with-resets --skip-ensure . RepositoryFactory

// RepositoryFactory is a contract for creating repositories
type RepositoryFactory interface {
	GetTickRepository(name string) (domain.TickRepository, error)
	GetLiquidationRepository(name string) (domain.LiquidationRepository, error)
}

// Importer is responsible for importing data from an exchange and storing it in the database
type Importer struct {
	exchange exchanges.Exchange
	logger   *zap.Logger

	tickRepository        domain.TickRepository
	liquidationRepository domain.LiquidationRepository
	tickHistory           *utils.RingBuffer[*domain.Tick]
	tickerHistory         map[domain.TickerName]*utils.RingBuffer[*domain.Ticker]

	marketNotifiers []notify.Client
	alertNotifiers  []notify.Client

	tickerMutex sync.Mutex
}

// NewImporter creates a new Importer
func NewImporter(exchange exchanges.Exchange, repositoryFactory RepositoryFactory, logger *zap.Logger) *Importer {
	tickRepository, err := repositoryFactory.GetTickRepository(exchange.GetName())
	if err != nil {
		return nil
	}
	liquidationRepository, err := repositoryFactory.GetLiquidationRepository(exchange.GetName())
	if err != nil {
		return nil
	}
	return &Importer{
		exchange: exchange,
		logger:   logger,

		tickRepository:        tickRepository,
		liquidationRepository: liquidationRepository,
		tickerHistory:         make(map[domain.TickerName]*utils.RingBuffer[*domain.Ticker]),
		tickHistory:           utils.NewRingBuffer[*domain.Tick](domain.MaxTickHistory),
	}
}

// StartLiquidationsImport starts importing liquidations from the exchange
func (i *Importer) StartLiquidationsImport(ctx context.Context) {
	liqChan, errChan := i.exchange.SubscribeLiquidations(ctx)
	if liqChan == nil || errChan == nil {
		i.logger.Error("Failed to subscribe to liquidations", zap.String("exchange", i.exchange.GetName()))
		return
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				i.logger.Info("Liquidation import stopped (context canceled).")
				return
			case liq := <-liqChan:
				// Convert the `exchanges.Liquidation` to your domain model
				domainLiq := domain.Liquidation{
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

				if err := domainLiq.Validate(); err != nil {
					i.logger.Error("Liquidation validation failed", zap.Error(err))
					continue
				}

				// Store it
				err := i.liquidationRepository.Create(context.Background(), domainLiq)
				if err != nil {
					i.logger.Error("Failed to store liquidation", zap.Error(err))
				}
			case err := <-errChan:
				i.logger.Error("Error on liquidation stream", zap.Error(err))
			}
		}
	}()
}

// StartImportLoop starts a loop that imports data from the exchange periodically.
func (i *Importer) StartImportLoop(ctx context.Context, interval time.Duration) error {
	i.StartLiquidationsImport(ctx)
	// Initialize the history data for calculating tick indicators
	if err := i.initHistory(ctx); err != nil {
		return fmt.Errorf("failed to init history: %w", err)
	}

	// Import should be started exactly at the beginning of the next second
	now := time.Now()
	nextSecond := now.Truncate(time.Second).Add(time.Second)
	time.Sleep(time.Until(nextSecond))

	// Start the import loop with the specified interval
	timeTicker := time.NewTicker(interval)
	defer timeTicker.Stop()
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
