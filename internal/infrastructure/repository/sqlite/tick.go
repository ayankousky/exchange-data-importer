package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
)

// TickRepository is a repository for ticks.
type TickRepository struct {
	db *sql.DB
}

func (r *TickRepository) init() error {
	tickTable := `
	CREATE TABLE IF NOT EXISTS ticks (
	  id INTEGER PRIMARY KEY AUTOINCREMENT,
	  start_at DATETIME,
	  created_at DATETIME,
	  tick_json TEXT
	);
	`
	if _, err := r.db.Exec(tickTable); err != nil {
		return fmt.Errorf("failed to create ticks table: %w", err)
	}

	return nil
}

// Create inserts a new tick into the database.
func (r *TickRepository) Create(ctx context.Context, ts domain.Tick) error {
	// Serialize the tick to JSON.
	data, err := json.Marshal(ts)
	if err != nil {
		return fmt.Errorf("failed to marshal tick: %w", err)
	}
	query := `INSERT INTO ticks (start_at, created_at, tick_json) VALUES (?, ?, ?)`
	_, err = r.db.ExecContext(ctx, query, ts.StartAt, ts.CreatedAt, string(data))
	if err != nil {
		return fmt.Errorf("failed to insert tick: %w", err)
	}
	return nil
}

// GetHistorySince returns all ticks created since the given time.
func (r *TickRepository) GetHistorySince(ctx context.Context, since time.Time) ([]domain.Tick, error) {
	query := `SELECT tick_json FROM ticks WHERE created_at >= ? ORDER BY created_at ASC`
	rows, err := r.db.QueryContext(ctx, query, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query ticks: %w", err)
	}
	defer rows.Close()

	var ticks []domain.Tick
	for rows.Next() {
		var tickJSON string
		if err := rows.Scan(&tickJSON); err != nil {
			return nil, fmt.Errorf("failed to scan tick row: %w", err)
		}
		var tick domain.Tick
		if err := json.Unmarshal([]byte(tickJSON), &tick); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tick: %w", err)
		}
		ticks = append(ticks, tick)
	}
	return ticks, nil
}
