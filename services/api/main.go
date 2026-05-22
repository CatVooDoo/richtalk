package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"richtalk/api/internal/app"
	"richtalk/api/internal/config"
	"richtalk/api/internal/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	log := logger.New(cfg.AppEnv)
	slog.SetDefault(log)

	a, err := app.New(cfg, log)
	if err != nil {
		log.Error("failed to initialize app", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := a.Run(ctx); err != nil {
		log.Error("app exited with error", "error", err)
		os.Exit(1)
	}
}
