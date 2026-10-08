package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/fentezi/mcp-google-health/internal/app"
	"github.com/fentezi/mcp-google-health/internal/config"
	"github.com/fentezi/mcp-google-health/pkg/logger"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	cfg := config.MustLoad()
	log := logger.New(cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := app.New(*cfg, log).Run(ctx); err != nil {
		log.Error("app run failed", "error", err)
		return err
	}
	return nil
}
