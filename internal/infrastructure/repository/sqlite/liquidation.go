package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
)

// LiquidationRepository is a repository for liquidations.
type LiquidationRepository struct {
	db *sql.DB
}

func (r *LiquidationRepository) init() error {
	liqTable := `
	CREATE TABLE IF NOT EXISTS liquidations (
	  id INTEGER PRIMARY KEY AUTOINCREMENT,
	  event_at DATETIME,
	  stored_at DATETIME,
	  liquidation_json TEXT
	);
	`
	if _, err := r.db.Exec(liqTable); err != nil {
		return fmt.Errorf("failed to create ticks table: %w", err)
	}

	return nil
}

// Create inserts a new liquidation into the database.
func (r *LiquidationRepository) Create(ctx context.Context, l domain.Liquidation) error {
	data, err := json.Marshal(l)
	if err != nil {
		return fmt.Errorf("failed to marshal liquidation: %w", err)
	}
	query := `INSERT INTO liquidations (event_at, stored_at, liquidation_json) VALUES (?, ?, ?)`
	_, err = r.db.ExecContext(ctx, query, l.EventAt, l.StoredAt, string(data))
	if err != nil {
		return fmt.Errorf("failed to insert liquidation: %w", err)
	}
	return nil
}

// GetLiquidationsHistory returns the liquidations history for the last 60 seconds.
func (r *LiquidationRepository) GetLiquidationsHistory(ctx context.Context, timeAt time.Time) (domain.LiquidationsHistory, error) {
	// For simplicity, consider a window of the last 60 seconds.
	windowStart := timeAt.Add(-60 * time.Second)
	query := `SELECT liquidation_json FROM liquidations WHERE event_at BETWEEN ? AND ?`
	rows, err := r.db.QueryContext(ctx, query, windowStart, timeAt)
	if err != nil {
		return domain.LiquidationsHistory{}, fmt.Errorf("failed to query liquidations: %w", err)
	}
	defer rows.Close()

	var history domain.LiquidationsHistory
	for rows.Next() {
		var liqJSON string
		if err := rows.Scan(&liqJSON); err != nil {
			return domain.LiquidationsHistory{}, fmt.Errorf("failed to scan liquidation row: %w", err)
		}
		var liq domain.Liquidation
		if err := json.Unmarshal([]byte(liqJSON), &liq); err != nil {
			return domain.LiquidationsHistory{}, fmt.Errorf("failed to unmarshal liquidation: %w", err)
		}
		delta := timeAt.Sub(liq.EventAt).Seconds()

		// For long liquidations, the order side should be SELL.
		if liq.Order.Side == domain.OrderSideSell {
			if delta <= 1 {
				history.LongLiquidations1s++
			}
			if delta <= 2 {
				history.LongLiquidations2s++
			}
			if delta <= 5 {
				history.LongLiquidations5s++
			}
			if delta <= 60 {
				history.LongLiquidations60s++
			}
		}

		// For short liquidations, the order side should be BUY.
		if liq.Order.Side == domain.OrderSideBuy {
			if delta <= 1 {
				history.ShortLiquidations1s++
			}
			if delta <= 2 {
				history.ShortLiquidations2s++
			}
			if delta <= 10 {
				history.ShortLiquidations10s++
			}
		}
	}
	return history, nil
}
