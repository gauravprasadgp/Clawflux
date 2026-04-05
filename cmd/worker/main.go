package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gauravprasad/clawcontrol/internal/app"
)

func main() {
	if err := app.LoadDotEnv(".env"); err != nil {
		slog.Error("failed to load .env", "error", err)
		os.Exit(1)
	}

	cfg := app.LoadConfig()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	runtime, err := app.NewRuntime(ctx, cfg)
	if err != nil {
		slog.Error("failed to initialise runtime", "error", err)
		os.Exit(1)
	}
	defer runtime.Close()

	runtime.Logger.Info("clawplane worker starting", "queue", cfg.RedisQueue)

	if err := runtime.Worker().Run(ctx); err != nil && err != context.Canceled {
		runtime.Logger.Error("worker exited with error", "error", err)
		os.Exit(1)
	}
	runtime.Logger.Info("worker stopped cleanly")
}
