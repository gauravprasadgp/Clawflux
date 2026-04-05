package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gauravprasad/clawcontrol/internal/app"
)

//go:generate swag init --parseInternal --generalInfo ./cmd/api/main.go --output ./docs/swagger

// @title Clawflux API
// @version 0.1.0
// @description Code-first Swagger docs for the current Clawflux HTTP API.
// @description
// @description Auth behavior:
// @description - Most /v1 endpoints accept either X-API-Key or X-User-Email.
// @description - DEVELOPMENT_AUTH=true allows unauthenticated local requests as developer@local.
// @description - Platform admin endpoints additionally require X-Platform-Admin: true.
// @BasePath /
// @schemes http
// @securityDefinitions.apikey APIKeyHeader
// @in header
// @name X-API-Key
// @securityDefinitions.apikey UserEmailHeader
// @in header
// @name X-User-Email
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

	server := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      runtime.HTTPHandler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		runtime.Logger.Info("clawflux api listening", "addr", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			runtime.Logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	runtime.Logger.Info("shutdown signal received, draining connections")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		runtime.Logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	runtime.Logger.Info("server stopped cleanly")
}
