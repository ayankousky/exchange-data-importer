package importer

import (
	"context"
	"fmt"
	"time"

	"github.com/ayankousky/exchange-data-importer/internal/domain"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges"
	"go.uber.org/zap"
)

// runLiquidationsImport processes the liquidation data stream
func (i *Importer) runLiquidationsImport(ctx context.Context) error {
	liqChan, errChan := i.exchange.SubscribeLiquidations(ctx)
	if liqChan == nil || errChan == nil {
		i.logger.Error("Failed to subscribe to liquidations", zap.String("exchange", i.exchange.GetName()))
		return fmt.Errorf("failed to subscribe to liquidations")
	}

	i.logger.Info("Started liquidations import")

	for {
		select {
		case <-ctx.Done():
			i.logger.Info("Liquidation import stopped (context canceled)")
			return ctx.Err()
		case liq := <-liqChan:
			if err := i.processLiquidation(ctx, liq); err != nil {
				i.logger.Error("failed to store liquidation", zap.Error(err))
			}
		case err := <-errChan:
			i.telemetry.IncrementCounter(telemetryLiquidationsErrors, 1, fmt.Sprintf("exchange:%s", i.exchange.GetName()))
			i.logger.Error("error on liquidation stream", zap.Error(err))
		}
	}
}

func (i *Importer) processLiquidation(ctx context.Context, liq exchanges.Liquidation) error {
	domainLiq := i.convertLiquidationToDomain(liq)
	if err := domainLiq.Validate(); err != nil {
		i.logger.Error("liquidation validation failed", zap.Error(err))
		return fmt.Errorf("validation error: %w", err)
	}

	return i.liquidationRepository.Create(ctx, domainLiq)
}

// convertLiquidationToDomain converts the exchange Liquidation to a domain Liquidation
func (i *Importer) convertLiquidationToDomain(liq exchanges.Liquidation) domain.Liquidation {
	return domain.Liquidation{
		Order: domain.Order{
			Symbol:     domain.TickerName(liq.Symbol),
			EventAt:    liq.EventAt,
			Side:       domain.OrderSide(liq.Side),
			Price:      liq.Price,
			Quantity:   liq.Quantity,
			TotalPrice: liq.TotalPrice,
		},
		EventAt:  liq.EventAt,
		StoredAt: time.Now(),
	}
}
