package bootstrap

import (
	"context"
	"fmt"

	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/telemetry"
	"go.uber.org/zap"

	"github.com/ayankousky/exchange-data-importer/internal/importer"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/exchanges"
	"github.com/ayankousky/exchange-data-importer/internal/infrastructure/notify"
)

// App represents the bootstrapped application
type App struct {
	logger            *zap.Logger
	exchange          exchanges.Exchange
	importer          *importer.Importer
	repositoryFactory importer.RepositoryFactory
	notifiers         []NotifierConfig
	telemetry         telemetry.Provider
	options           *Options
}

// NotifierConfig holds notifier configuration
type NotifierConfig struct {
	Client   notify.Client
	Topic    string
	Strategy notify.Strategy
}

// Start initializes and starts the application
func (a *App) Start(ctx context.Context) error {
	// Add notifiers to the importer
	for _, notifier := range a.notifiers {
		if err := a.importer.WithNotifier(notifier.Client, notifier.Topic, notifier.Strategy); err != nil {
			a.logger.Warn("Error adding notifier", zap.Error(err))
		}
	}

	// Start handling imports
	if err := a.importer.Start(ctx); err != nil {
		return fmt.Errorf("starting import loop: %w", err)
	}

	return nil
}
