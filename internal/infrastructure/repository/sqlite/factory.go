package sqlite

import (
	"database/sql"
	"fmt"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
)

// Factory implements a repository factory using SQLite.
type Factory struct {
	db *sql.DB
}

// NewSQLiteRepoFactory opens (or creates) a SQLite database file (dsn)
// and creates the necessary tables if they do not exist.
func NewSQLiteRepoFactory(dsn string) (*Factory, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite db: %w", err)
	}

	return &Factory{db: db}, nil
}

// GetTickRepository returns a TickRepository instance.
func (f *Factory) GetTickRepository(_ string) (domain.TickRepository, error) {
	repo := &TickRepository{
		db: f.db,
	}
	if err := repo.init(); err != nil {
		return nil, err
	}
	return repo, nil
}

// GetLiquidationRepository returns a LiquidationRepository instance.
func (f *Factory) GetLiquidationRepository(_ string) (domain.LiquidationRepository, error) {
	repo := &LiquidationRepository{
		db: f.db,
	}
	if err := repo.init(); err != nil {
		return nil, err
	}
	return repo, nil
}
