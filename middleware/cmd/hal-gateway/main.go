package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	halgateway "github.com/mannsub/robotos/services/hal-gateway"
)

func main() {
	addr := os.Getenv("HAL_GATEWAY_ADDR")
	if addr == "" {
		addr = halgateway.DefaultAdd
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Printf("[hal-gateway] starting on %s", addr)
	if err := halgateway.RunWithContext(ctx, addr); err != nil {
		log.Fatalf("[hal-gateway] fatal: %v", err)
	}
}
