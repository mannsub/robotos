package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mannsub/robotos/pkg/bus"
	"github.com/mannsub/robotos/pkg/health"
	"github.com/mannsub/robotos/services/behavior"
	"github.com/mannsub/robotos/services/motion"
	"github.com/mannsub/robotos/services/navigation"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	b := bus.New()
	h := health.New(5 * time.Second)

	nav := navigation.New(b, h)
	mot := motion.New(b, h, motion.NewMockDriver(12))
	beh := behavior.New(b, "neodm:50051")

	go func() {
		nav.Run(ctx)
	}()

	go func() {
		mot.Run(ctx)
	}()

	go func() {
		if err := beh.Run(ctx); err != nil {
			log.Printf("[beh] error: %v", err)
		}
	}()

	log.Println("[robotos] all services started")
	<-ctx.Done()
	log.Println("[robotos] shutting down")
}
