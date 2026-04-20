package rounds

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// Handler serves the round detail API endpoints.
type Handler struct {
	service *Service
	logger  *slog.Logger
}

// NewHandler creates a new rounds handler.
func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// GetRoundDetail handles GET /api/v1/rounds/{round}?year=2026.
func (h *Handler) GetRoundDetail(w http.ResponseWriter, r *http.Request) {
	roundStr := chi.URLParam(r, "round")
	round, err := strconv.Atoi(roundStr)
	if err != nil || round < 1 || round > 30 {
		http.Error(w, `{"error":"invalid round parameter"}`, http.StatusBadRequest)
		return
	}

	yearStr := r.URL.Query().Get("year")
	if yearStr == "" {
		yearStr = strconv.Itoa(time.Now().Year())
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 1950 || year > 2100 {
		http.Error(w, `{"error":"invalid year parameter"}`, http.StatusBadRequest)
		return
	}

	resp, err := h.service.GetRoundDetail(r.Context(), year, round)
	if err != nil {
		h.logger.Error("rounds service error", "error", err, "year", year, "round", round)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("rounds encode error", "error", err)
	}
}
