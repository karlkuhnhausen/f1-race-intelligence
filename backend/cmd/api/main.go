package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/api"
)

func main() {
	addr := envOrDefault("BACKEND_LISTEN_ADDR", ":8080")

	router := api.NewRouter()
	server := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("backend starting on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("backend failed: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
