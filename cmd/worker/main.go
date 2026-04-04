package main

import (
	"context"
	"log"

	"github.com/gauravprasad/clawcontrol/internal/app"
)

func main() {
	cfg := app.LoadConfig()
	runtime, err := app.NewRuntime(context.Background(), cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer runtime.Close()
	log.Printf("clawplane worker consuming from %s", cfg.RedisQueue)
	if err := runtime.Worker().Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
