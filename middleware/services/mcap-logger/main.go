package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
)

func main() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	outDir := os.Getenv("MCAP_OUT_DIR")
	if outDir == "" {
		outDir = "./recordings"
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatalf("[mcap-logger] failed to create output dir: %v", err)
	}

	outPath := fmt.Sprintf("%s/robotos_%s.mcap", outDir, time.Now().Format("20060102_150405"))

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})

	logger, err := NewLogger(rdb, outPath)
	if err != nil {
		log.Fatalf("[mcap-logger] init failed: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Printf("[mcap-logger] recording to %s", outPath)
	if err := logger.Run(ctx); err != nil {
		log.Printf("[mcap-logger stopped]: %v", err)
	}
}
