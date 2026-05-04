package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/api/analysis"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/api/calendar"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/api/rounds"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/api/standings"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/ingest"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

type healthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Timestamp string `json:"timestamp"`
}

func NewRouter(calendarRepo storage.CalendarRepository, standingsRepo storage.StandingsRepository, sessionRepo storage.SessionRepository, analysisRepo storage.AnalysisRepository, championshipRepo storage.ChampionshipRepository, logger *slog.Logger) http.Handler {
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
	calendarSvc := calendar.NewServiceWithSessions(calendarRepo, sessionRepo).WithStandings(standingsRepo)
	calendarHandler := calendar.NewHandler(calendarSvc, logger)

	// Standings API
	standingsSvc := standings.NewService(standingsRepo, championshipRepo, sessionRepo)
	standingsHandler := standings.NewHandler(standingsSvc, logger)

	// Rounds API — wire a real RaceControlHydrator for lazy-on-read gap fill.
	var rcHydrator rounds.RaceControlHydrator
	if sessionRepo != nil {
		rcHydrator = ingest.NewRaceControlHydrator(sessionRepo, logger)
	}
	roundsSvc := rounds.NewServiceWithHydrator(sessionRepo, calendarRepo, rcHydrator, logger)
	roundsHandler := rounds.NewHandler(roundsSvc, logger)

	// Analysis API
	analysisSvc := analysis.NewService(analysisRepo, logger)
	analysisHandler := analysis.NewHandler(analysisSvc, logger)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/calendar", calendarHandler.GetCalendar)
		r.Get("/standings/drivers", standingsHandler.GetDrivers)
		r.Get("/standings/constructors", standingsHandler.GetConstructors)
		r.Get("/standings/drivers/progression", standingsHandler.GetDriversProgression)
		r.Get("/standings/constructors/progression", standingsHandler.GetConstructorsProgression)
		r.Get("/standings/drivers/compare", standingsHandler.GetDriversCompare)
		r.Get("/standings/constructors/compare", standingsHandler.GetConstructorsCompare)
		r.Get("/standings/constructors/{team}/drivers", standingsHandler.GetConstructorDrivers)
		r.Get("/rounds/{round}", roundsHandler.GetRoundDetail)
		r.Get("/rounds/{round}/sessions/{type}/analysis", analysisHandler.GetSessionAnalysis)
	})

	return r
}
