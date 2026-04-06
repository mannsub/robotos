package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	goredis "github.com/redis/go-redis/v9"
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

	go func() { nav.Run(ctx) }()
	go func() { mot.Run(ctx) }()
	go func() {
		if err := beh.Run(ctx); err != nil {
			log.Printf("[beh] error: %v", err)
		}
	}()

	// Redis ↔ bus bridge for dashboard communication.
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		go runRedisBridge(ctx, redisURL, b)
	}

	log.Println("[robotos] all services started")
	<-ctx.Done()
	log.Println("[robotos] shutting down")
}

// runRedisBridge bridges dashboard ↔ navigation via Redis:
//
//	nav:goal     (Redis) → robot/goal         (bus)
//	nav:obstacle (Redis) → robot/map/obstacle  (bus)
//	robot/state/navigation (bus) → nav:state  (Redis)
func runRedisBridge(ctx context.Context, redisURL string, b *bus.Bus) {
	opt, err := goredis.ParseURL(redisURL)
	if err != nil {
		log.Printf("[bridge] invalid REDIS_URL: %v", err)
		return
	}
	rdb := goredis.NewClient(opt)
	defer rdb.Close()

	// bus → Redis: publish nav state for the dashboard
	navStateCh := b.Sub("robot/state/navigation", 32)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-navStateCh:
				rdb.Publish(ctx, "nav:state", string(msg.Payload))
			}
		}
	}()

	// Redis → bus: forward dashboard goal/obstacle commands
	sub := rdb.Subscribe(ctx, "nav:goal", "nav:obstacle", "nav:reset", "nav:reset_robot", "nav:maze")
	defer sub.Close()

	log.Println("[bridge] Redis bridge started")

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-sub.Channel():
			if !ok {
				return
			}
			switch msg.Channel {
			case "nav:goal":
				b.Pub("robot/goal", []byte(msg.Payload))
			case "nav:obstacle":
				var cmd struct {
					X       float64 `json:"x"`
					Y       float64 `json:"y"`
					Blocked bool    `json:"blocked"`
				}
				if err := json.Unmarshal([]byte(msg.Payload), &cmd); err != nil {
					continue
				}
				obs, _ := json.Marshal(map[string]any{
					"x": cmd.X, "y": cmd.Y, "blocked": cmd.Blocked,
				})
				b.Pub("robot/map/obstacle", obs)
			case "nav:reset":
				b.Pub("robot/map/reset", []byte("{}"))
			case "nav:reset_robot":
				b.Pub("robot/reset/robot", []byte("{}"))
			case "nav:maze":
				b.Pub("robot/map/batch", []byte(msg.Payload))
			}
		}
	}
}
