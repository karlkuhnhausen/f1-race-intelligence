package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/api"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/observability"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage/cosmos"
)

func main() {
	addr := envOrDefault("BACKEND_LISTEN_ADDR", ":8080")
	cosmosEndpoint := envOrDefault("COSMOS_ACCOUNT_ENDPOINT", "")

	logger := observability.NewLogger(slog.LevelInfo)

	var calendarRepo storage.CalendarRepository
	var standingsRepo storage.StandingsRepository
	if cosmosEndpoint != "" {
		client, err := cosmos.NewClient(cosmosEndpoint)
		if err != nil {
			log.Fatalf("cosmos client: %v", err)
		}
		calendarRepo = client
		standingsRepo = client
	}

	router := api.NewRouter(calendarRepo, standingsRepo, logger)
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
