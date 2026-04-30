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
	// Serve mesh assets over plain HTTP.
	// bunny.stl is embedded in the binary; bunny.glb is read from /meshes volume mount.
	mux.HandleFunc("/meshes/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		name := r.URL.Path[len("/meshes/"):]
		switch name {
		case "bunny.stl":
			w.Header().Set("Content-Type", "model/stl")
			w.Write(bunnySTL)
		case "bunny.glb":
			w.Header().Set("Content-Type", "model/gltf-binary")
			http.ServeFile(w, r, "/meshes/bunny.glb")
		default:
			http.NotFound(w, r)
		}
	})
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
