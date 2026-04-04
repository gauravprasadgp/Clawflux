package main

import (
	"context"
	"log"
	"net/http"

	"github.com/gauravprasad/clawcontrol/internal/app"
)

func main() {
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
