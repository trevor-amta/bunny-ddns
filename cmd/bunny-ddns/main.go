package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/trevorspencer/bunny-dynamic-dns/internal/app"
	"github.com/trevorspencer/bunny-dynamic-dns/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := app.Run(ctx, cfg, os.Stdout); err != nil {
		log.Fatalf("application error: %v", err)
	}
}
