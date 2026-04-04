package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/redis/go-redis/v9"
)

func main() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	listenAddr := os.Getenv("FOXGLOVE_ADDR")
	if listenAddr == "" {
		listenAddr = ":8765"
	}

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})

	bridge := NewBridge(rdb)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go bridge.Run(ctx)

	mux := http.NewServeMux()
	mux.Handle("/", bridge)

	srv := &http.Server{
		Addr:    listenAddr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()

	log.Printf("[foxglove-bridge] listening on ws://localhost%s", listenAddr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("[foxglove-bridge] server error: %v", err)
	}
}
