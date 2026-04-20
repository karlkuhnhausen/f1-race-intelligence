package calendar

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
)

// Handler serves the calendar API endpoints.
type Handler struct {
	service *Service
	logger  *slog.Logger
}

// NewHandler creates a new calendar handler.
func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// GetCalendar handles GET /api/v1/calendar?year=2026.
func (h *Handler) GetCalendar(w http.ResponseWriter, r *http.Request) {
	yearStr := r.URL.Query().Get("year")
	if yearStr == "" {
		http.Error(w, `{"error":"year query parameter is required"}`, http.StatusBadRequest)
		return
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 1950 || year > 2100 {
		http.Error(w, `{"error":"invalid year parameter"}`, http.StatusBadRequest)
		return
	}

	resp, err := h.service.GetCalendar(r.Context(), year)
	if err != nil {
		h.logger.Error("calendar service error", "error", err, "year", year)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("calendar encode error", "error", err)
	}
}
