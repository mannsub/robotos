package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	addr := os.Getenv("DASHBOARD_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	hub := newHub()
	go hub.run()
	go subscribeRedis(redisAddr, hub)

	http.HandleFunc("/ws", hub.serveWS)

	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "./frontend/dist"
	}
	http.Handle("/", http.FileServer(http.Dir(staticDir)))

	log.Printf("[dashboard] listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
