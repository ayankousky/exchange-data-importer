package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/mattn/go-sqlite3"

	"github.com/ayankousky/exchange-data-importer/internal/bootstrap"
)

var revision = "local"

func main() {
	fmt.Printf("Exchange Data Importer: %s\n", revision)
	// Create context that can be canceled by system signals
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Build the application
	app, err := bootstrap.NewBuilder().
		WithLogger(ctx).
		WithExchange(ctx).
		WithRepository(ctx).
		WithNotifiers(ctx).
		WithTelemetry(ctx, revision).
		Build()
	if err != nil {
		fmt.Printf("Error building application: %v\n", err)
		os.Exit(1)
	}

	// Start the application
	if err := app.Start(ctx); err != nil {
		fmt.Printf("Error starting application: %v\n", err)
		os.Exit(1)
	}

	// Wait for shutdown signal
	<-ctx.Done()
	fmt.Println("Shutting down gracefully...")
}
