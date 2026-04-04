package main

import (
	"context"
	"log"
	"net/http"

	"github.com/gauravprasad/clawcontrol/internal/app"
)

//go:generate swag init --parseInternal --generalInfo ./cmd/api/main.go --output ./docs/swagger

// @title ClawPlane API
// @version 0.1.0
// @description Code-first Swagger docs for the current ClawPlane HTTP API.
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
		log.Fatal(err)
	}
	cfg := app.LoadConfig()
	runtime, err := app.NewRuntime(context.Background(), cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer runtime.Close()
	server := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: runtime.HTTPHandler(),
	}
	log.Printf("clawplane api listening on %s", cfg.HTTPAddr)
	log.Fatal(server.ListenAndServe())
}
