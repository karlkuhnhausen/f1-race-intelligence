package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/api/calendar"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

type healthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Timestamp string `json:"timestamp"`
}

func NewRouter(calendarRepo storage.CalendarRepository, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(healthResponse{
			Status:    "ok",
			Service:   "f1-race-intelligence-backend",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
	})

	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	// Calendar API
	calendarSvc := calendar.NewService(calendarRepo)
	calendarHandler := calendar.NewHandler(calendarSvc, logger)
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/calendar", calendarHandler.GetCalendar)
	})

	return r
}
