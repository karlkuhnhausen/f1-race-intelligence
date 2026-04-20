package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/api"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/ingest"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/observability"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/standings"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage/cosmos"
)

func main() {
	addr := envOrDefault("BACKEND_LISTEN_ADDR", ":8080")
	cosmosEndpoint := envOrDefault("COSMOS_ACCOUNT_ENDPOINT", "")

	logger := observability.NewLogger(slog.LevelInfo)

	var calendarRepo storage.CalendarRepository
	var standingsRepo storage.StandingsRepository
	var sessionRepo storage.SessionRepository
	if cosmosEndpoint != "" {
		client, err := cosmos.NewClient(cosmosEndpoint)
		if err != nil {
			log.Fatalf("cosmos client: %v", err)
		}
		calendarRepo = client
		standingsRepo = client
		sessionRepo = client
	}

	router := api.NewRouter(calendarRepo, standingsRepo, sessionRepo, logger)

	// Start data pollers if Cosmos is configured.
	if calendarRepo != nil {
		season := time.Now().Year()
		ctx := context.Background()

		calendarPoller := ingest.NewOpenF1Poller(calendarRepo, logger)
		go calendarPoller.Start(ctx, season)

		standingsPoller := standings.NewHypraceClient(standingsRepo, logger)
		go standingsPoller.Start(ctx, season)

		sessionPoller := ingest.NewSessionPoller(sessionRepo, logger)
		go sessionPoller.Start(ctx, season)
	}

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
