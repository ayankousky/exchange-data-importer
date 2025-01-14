package domain

import "time"

// Liquidation represents a market liquidation event
// Basically liquidation is a regular order but it could have either a buy or sell side
type Liquidation struct {
	Order      Order     `json:"o"`
	EventTime  time.Time `db:"et" json:"et" bson:"et"` // event could come from exchange with a delay
	StoredTime time.Time `db:"st" json:"st" bson:"st"` // time when the event was stored in the database
}
