package domain

import (
	"context"
	"fmt"
	"time"
)

//go:generate moq --out mocks/liquidation_repository.go --pkg mocks --with-resets --skip-ensure . LiquidationRepository

// LiquidationType represents the type of liquidation
type LiquidationType string

const (
	// LongLiquidation represents a long liquidation (meaning a force sell order)
	LongLiquidation LiquidationType = "SELL"

	// ShortLiquidation represents a short liquidation (meaning a force buy order)
	ShortLiquidation LiquidationType = "BUY"
)

// Liquidation represents a market liquidation event
type Liquidation struct {
	Order    Order     `json:"o"`
	EventAt  time.Time `db:"et" json:"et" bson:"et"` // event could come from exchange with a delay
	StoredAt time.Time `db:"st" json:"st" bson:"st"` // time when the event was stored in the database
}

// Validate performs validation of the Liquidation
func (l *Liquidation) Validate() error {
	if l.EventAt.IsZero() {
		return ValidationError{
			Field: "EventAt",
			Err:   fmt.Errorf("event time cannot be zero"),
		}
	}

	if l.StoredAt.IsZero() {
		return ValidationError{
			Field: "StoredAt",
			Err:   fmt.Errorf("stored time cannot be zero"),
		}
	}

	if err := l.Order.Validate(); err != nil {
		return ValidationError{
			Field: "Order",
			Err:   err,
		}
	}

	// Validate that Order.Side matches LiquidationType
	switch l.Order.Side {
	case OrderSide(LongLiquidation):
		if l.Order.Side != OrderSideSell {
			return ValidationError{
				Field: "Order.Side",
				Err:   fmt.Errorf("long liquidation must have SELL order side"),
			}
		}
	case OrderSide(ShortLiquidation):
		if l.Order.Side != OrderSideBuy {
			return ValidationError{
				Field: "Order.Side",
				Err:   fmt.Errorf("short liquidation must have BUY order side"),
			}
		}
	default:
		return ValidationError{
			Field: "Order.Side",
			Err:   fmt.Errorf("invalid liquidation type"),
		}
	}

	return nil
}

// LiquidationsHistory represents the liquidation history at a specific point in time
type LiquidationsHistory struct {
	LongLiquidations1s   int64
	LongLiquidations2s   int64
	LongLiquidations5s   int64
	LongLiquidations60s  int64
	ShortLiquidations1s  int64
	ShortLiquidations2s  int64
	ShortLiquidations10s int64
}

// LiquidationRepository represents the liquidation repository contract
type LiquidationRepository interface {
	Create(ctx context.Context, l Liquidation) error
	GetLiquidationsHistory(ctx context.Context, timeAt time.Time) (LiquidationsHistory, error)
}
