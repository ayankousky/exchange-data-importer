package importer

import (
	"context"
	"fmt"
	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"github.com/ayankousky/exchange-data-importer/pkg/exchanges"
	"time"
)

// RepositoryFactory is a contract for creating repositories
// each exchange must have its own separate repository
type RepositoryFactory interface {
	GetTickRepository(name string) domain.TickSnapshotRepository
	GetLiquidationRepository(name string) domain.LiquidationRepository
}

// Importer is responsible for importing data from an exchange and storing it in the database
type Importer struct {
	Exchange              exchanges.Exchange
	TickRepository        domain.TickSnapshotRepository
	LiquidationRepository domain.LiquidationRepository
}

// NewImporter creates a new Importer
func NewImporter(exchange exchanges.Exchange, repositoryFactory RepositoryFactory) *Importer {
	return &Importer{
		Exchange:              exchange,
		TickRepository:        repositoryFactory.GetTickRepository(exchange.GetName()),
		LiquidationRepository: repositoryFactory.GetLiquidationRepository(exchange.GetName()),
	}
}

// StartImport imports tick data from the exchange (temporary method)
func (i *Importer) StartImport() error {
	startAt := time.Now()

	tickers, err := i.Exchange.FetchTickers(context.Background())
	if err != nil {
		return err
	}
	fetchedAt := time.Now()

	// Generate ISO timestamp ID
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	tick := &domain.Tick{
		ID:        timestamp,
		StartAt:   startAt,
		FetchedAt: fetchedAt,
	}
	data := make(map[string]domain.Ticker, 0)
	for _, ticker := range tickers {
		data[ticker.Symbol] = domain.Ticker{}
	}

	tick.CreatedAt = time.Now()
	err = i.TickRepository.Create(context.Background(), tick)
	if err != nil {
		return err
	}

	return nil
}

// StartImportEverySecond starts the import process every second
func (i *Importer) StartImportEverySecond() {
	for {
		// Calculate the duration until the next second
		now := time.Now()
		next := now.Truncate(time.Second).Add(time.Second)
		time.Sleep(time.Until(next))

		// Run StartImport
		err := i.StartImport()
		if err != nil {
			fmt.Printf("Error in StartImport: %v\n", err)
		}
	}
}
