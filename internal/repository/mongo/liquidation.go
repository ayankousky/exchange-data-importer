package mongo

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// NewLiquidationRepository creates a new Liquidation repository and ensures the required indexes
func NewLiquidationRepository(db *mongo.Collection) (*Liquidation, error) {
	if db == nil {
		return nil, fmt.Errorf("db is required")
	}
	repo := &Liquidation{
		db: db,
	}

	if err := repo.ensureIndexes(context.Background()); err != nil {
		return nil, err
	}

	return repo, nil
}

// Liquidation is a repository for storing liquidation snapshots
type Liquidation struct {
	db *mongo.Collection
}

// Create method stores a liquidation in the database
func (r *Liquidation) Create(ctx context.Context, liquidation domain.Liquidation) error {
	_, err := r.db.InsertOne(ctx, liquidation)
	if err != nil {
		log.Default().Printf("Error inserting tick snapshot: %v", err)
	}

	return nil
}

// GetLiquidationsHistory returns liquidation history for specified time ranges
func (r *Liquidation) GetLiquidationsHistory(ctx context.Context, timeAt time.Time) (history domain.LiquidationsHistory, err error) {
	type liquidationsParams struct {
		Seconds  int
		Side     domain.LiquidationType
		SetField *int64
	}
	timeRanges := []liquidationsParams{
		{1, domain.LongLiquidation, &history.LongLiquidations1s},
		{2, domain.LongLiquidation, &history.LongLiquidations2s},
		{5, domain.LongLiquidation, &history.LongLiquidations5s},
		{60, domain.LongLiquidation, &history.LongLiquidations60s},
		{1, domain.ShortLiquidation, &history.ShortLiquidations1s},
		{2, domain.ShortLiquidation, &history.ShortLiquidations2s},
		{10, domain.ShortLiquidation, &history.ShortLiquidations10s},
	}
	for _, tr := range timeRanges {
		count, err := r.getLiquidationsCount(ctx, timeAt, tr.Seconds, tr.Side)
		if err != nil {
			return history, fmt.Errorf("error getting long liquidations for %d seconds: %w", tr.Seconds, err)
		}

		*tr.SetField = count
	}

	return history, nil
}

func (r *Liquidation) getLiquidationsCount(ctx context.Context, timeAt time.Time, seconds int, liquidationType domain.LiquidationType) (int64, error) {
	filter := bson.M{
		"order.sd": string(liquidationType),
		"st": bson.M{
			"$gte": timeAt.Add(time.Duration(-seconds) * time.Second),
			"$lte": timeAt,
		},
		// sometimes the event could come with a delay
		"et": bson.M{
			"$gte": timeAt.Add(time.Duration(-seconds*5) * time.Second),
			"$lte": timeAt,
		},
	}

	count, err := r.db.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("error counting documents: %w", err)
	}

	return count, nil
}

// ensureIndexes creates the required indexes for optimal query performance
func (r *Liquidation) ensureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "st", Value: 1},
				{Key: "et", Value: 1},
				{Key: "order.sd", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "st", Value: 1},
				{Key: "et", Value: 1},
				{Key: "symbol", Value: 1},
				{Key: "order.sd", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "st", Value: 1},
			},
			Options: options.Index().SetExpireAfterSeconds(60 * 60 * 24 * 7), // 14 days
		},
	}

	_, err := r.db.Indexes().CreateMany(ctx, indexes)
	return err
}
