package importer

import (
	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"github.com/ayankousky/exchange-data-importer/pkg/exchanges"
)

// Importer is responsible for importing data from an exchange and storing it in the database
type Importer struct {
	Exchange               exchanges.Exchange
	TickSnapshotRepository domain.TickSnapshotRepository
	LiquidationRepository  domain.LiquidationRepository
}
