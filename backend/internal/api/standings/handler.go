package standings

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
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
		return 0, false
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 1950 || year > 2100 {
		return 0, false
	}
	return year, true
}

// GetDrivers handles GET /api/v1/standings/drivers?year=YYYY.
func (h *Handler) GetDrivers(w http.ResponseWriter, r *http.Request) {
	year, ok := parseYear(r)
	if !ok {
		http.Error(w, `{"error":"year query parameter is required and must be 1950-2100"}`, http.StatusBadRequest)
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
		http.Error(w, `{"error":"year query parameter is required and must be 1950-2100"}`, http.StatusBadRequest)
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
