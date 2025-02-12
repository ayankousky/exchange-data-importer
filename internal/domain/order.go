package domain

import (
	"fmt"
	"time"
)

// OrderSide represents all possible order sides
type OrderSide string

const (
	// OrderSideBuy represents a buy order
	OrderSideBuy OrderSide = "BUY"

	// OrderSideSell represents a sell order
	OrderSideSell OrderSide = "SELL"
)

// Order represents any order in the system
type Order struct {
	EventAt    time.Time  `db:"et" json:"et" bson:"et"`
	Symbol     TickerName `db:"s" json:"s" bson:"s"`
	Side       OrderSide  `db:"sd" json:"sd" bson:"sd" validate:"required,oneof=BUY SELL"`
	Price      float64    `db:"p" json:"p" bson:"p"`
	Quantity   float64    `db:"q" json:"q" bson:"q"`
	TotalPrice float64    `db:"tp" json:"tp" bson:"tp"`
}

// Validate performs validation of the Order
func (o *Order) Validate() error {
	if o.EventAt.IsZero() {
		return ValidationError{
			Field: "EventAt",
			Err:   fmt.Errorf("event time cannot be zero for %s", o.Symbol),
		}
	}

	if o.Symbol == "" {
		return ValidationError{
			Field: "Symbol",
			Err:   fmt.Errorf("symbol cannot be empty for %s", o.Symbol),
		}
	}

	if o.Side != OrderSideBuy && o.Side != OrderSideSell {
		return ValidationError{
			Field: "Side",
			Err:   fmt.Errorf("invalid order side: %s for %s", o.Side, o.Symbol),
		}
	}

	if o.Price <= 0 {
		return ValidationError{
			Field: "Price",
			Err:   fmt.Errorf("price must be greater than 0 for %s", o.Symbol),
		}
	}

	if o.Quantity <= 0 {
		return ValidationError{
			Field: "Quantity",
			Err:   fmt.Errorf("quantity must be greater than 0 for %s", o.Symbol),
		}
	}

	expectedTotal := o.Price * o.Quantity
	if o.TotalPrice != expectedTotal {
		return ValidationError{
			Field: "TotalPrice",
			Err:   fmt.Errorf("total price %f does not match price * quantity = %f for %s", o.TotalPrice, expectedTotal, o.Symbol),
		}
	}

	return nil
}
