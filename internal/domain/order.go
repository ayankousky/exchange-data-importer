package domain

import "time"

// OrderSide represents all possible order sides
type OrderSide string

const (
	// SideBuy represents a buy order
	SideBuy OrderSide = "BUY"
	// SideSell represents a sell order
	SideSell OrderSide = "SELL"
)

// Order represents any order in the system
type Order struct {
	Date       time.Time  `db:"tt" json:"tt" bson:"tt"`
	Symbol     TickerName `db:"s" json:"s" bson:"s"`
	Side       OrderSide  `db:"sd" json:"sd" bson:"sd" validate:"required,oneof=BUY SELL"`
	Price      float64    `db:"p" json:"p" bson:"p"`
	Quantity   float64    `db:"q" json:"q" bson:"q"`
	TotalPrice float64    `db:"tp" json:"tp" bson:"tp"`
}
