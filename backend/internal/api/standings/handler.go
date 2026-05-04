package standings

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

// Handler serves championship standings endpoints.
type Handler struct {
	service *Service
	logger  *slog.Logger
}

// NewHandler creates a standings handler.
func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func parseYear(r *http.Request) (int, bool) {
	yearStr := r.URL.Query().Get("year")
	if yearStr == "" {
		// Default to current year.
		return time.Now().Year(), true
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 2023 || year > time.Now().Year() {
		return 0, false
	}
	return year, true
}

// GetDrivers handles GET /api/v1/standings/drivers?year=YYYY.
func (h *Handler) GetDrivers(w http.ResponseWriter, r *http.Request) {
	year, ok := parseYear(r)
	if !ok {
		http.Error(w, `{"error":"year must be between 2023 and the current year"}`, http.StatusBadRequest)
		return
	}

	resp, err := h.service.GetDrivers(r.Context(), year)
	if err != nil {
		h.logger.Error("drivers standings error", "error", err, "year", year)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("drivers standings encode error", "error", err)
	}
}

// GetConstructors handles GET /api/v1/standings/constructors?year=YYYY.
func (h *Handler) GetConstructors(w http.ResponseWriter, r *http.Request) {
	year, ok := parseYear(r)
	if !ok {
		http.Error(w, `{"error":"year must be between 2023 and the current year"}`, http.StatusBadRequest)
		return
	}

	resp, err := h.service.GetConstructors(r.Context(), year)
	if err != nil {
		h.logger.Error("constructors standings error", "error", err, "year", year)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("constructors standings encode error", "error", err)
	}
}

// GetDriversProgression handles GET /api/v1/standings/drivers/progression?year=YYYY.
func (h *Handler) GetDriversProgression(w http.ResponseWriter, r *http.Request) {
	year, ok := parseYear(r)
	if !ok {
		http.Error(w, `{"error":"year must be between 2023 and the current year"}`, http.StatusBadRequest)
		return
	}

	resp, err := h.service.GetDriverProgression(r.Context(), year)
	if err != nil {
		h.logger.Error("drivers progression error", "error", err, "year", year)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("drivers progression encode error", "error", err)
	}
}

// GetConstructorsProgression handles GET /api/v1/standings/constructors/progression?year=YYYY.
func (h *Handler) GetConstructorsProgression(w http.ResponseWriter, r *http.Request) {
	year, ok := parseYear(r)
	if !ok {
		http.Error(w, `{"error":"year must be between 2023 and the current year"}`, http.StatusBadRequest)
		return
	}

	resp, err := h.service.GetConstructorProgression(r.Context(), year)
	if err != nil {
		h.logger.Error("constructors progression error", "error", err, "year", year)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("constructors progression encode error", "error", err)
	}
}

// GetDriversCompare handles GET /api/v1/standings/drivers/compare?year=YYYY&driver1=N&driver2=N.
func (h *Handler) GetDriversCompare(w http.ResponseWriter, r *http.Request) {
	year, ok := parseYear(r)
	if !ok {
		http.Error(w, `{"error":"year must be between 2023 and the current year"}`, http.StatusBadRequest)
		return
	}

	d1Str := r.URL.Query().Get("driver1")
	d2Str := r.URL.Query().Get("driver2")
	d1, err1 := strconv.Atoi(d1Str)
	d2, err2 := strconv.Atoi(d2Str)
	if err1 != nil || err2 != nil || d1 == d2 {
		http.Error(w, `{"error":"driver1 and driver2 must be different valid driver numbers"}`, http.StatusBadRequest)
		return
	}

	resp, err := h.service.GetDriverComparison(r.Context(), year, d1, d2)
	if err != nil {
		h.logger.Error("driver comparison error", "error", err, "year", year)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("driver comparison encode error", "error", err)
	}
}

// GetConstructorsCompare handles GET /api/v1/standings/constructors/compare?year=YYYY&team1=X&team2=X.
func (h *Handler) GetConstructorsCompare(w http.ResponseWriter, r *http.Request) {
	year, ok := parseYear(r)
	if !ok {
		http.Error(w, `{"error":"year must be between 2023 and the current year"}`, http.StatusBadRequest)
		return
	}

	team1 := r.URL.Query().Get("team1")
	team2 := r.URL.Query().Get("team2")
	if team1 == "" || team2 == "" || team1 == team2 {
		http.Error(w, `{"error":"team1 and team2 must be different non-empty team names"}`, http.StatusBadRequest)
		return
	}

	resp, err := h.service.GetConstructorComparison(r.Context(), year, team1, team2)
	if err != nil {
		h.logger.Error("constructor comparison error", "error", err, "year", year)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("constructor comparison encode error", "error", err)
	}
}
