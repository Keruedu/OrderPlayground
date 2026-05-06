package main

import (
	"context"
	"log"

	"order-playground/apps/gateway-api/internal/app"
	"order-playground/apps/gateway-api/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if err := app.New(cfg).Run(context.Background()); err != nil {
		log.Fatalf("run app: %v", err)
	}
}
