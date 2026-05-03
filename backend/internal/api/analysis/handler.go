package analysis

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// Handler serves the session analysis API endpoints.
type Handler struct {
	service *Service
	logger  *slog.Logger
}

// NewHandler creates a new analysis handler.
func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// GetSessionAnalysis handles GET /api/v1/rounds/{round}/sessions/{type}/analysis?year=2026.
func (h *Handler) GetSessionAnalysis(w http.ResponseWriter, r *http.Request) {
	roundStr := chi.URLParam(r, "round")
	round, err := strconv.Atoi(roundStr)
	if err != nil || round < 1 || round > 30 {
		http.Error(w, `{"error":"invalid round parameter"}`, http.StatusBadRequest)
		return
	}

	sessionType := strings.ToLower(chi.URLParam(r, "type"))
	if sessionType != "race" && sessionType != "sprint" {
		http.Error(w, `{"error":"analysis is only available for race and sprint sessions"}`, http.StatusBadRequest)
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

	dto, err := h.service.GetSessionAnalysis(r.Context(), year, round, sessionType)
	if err != nil {
		h.logger.Error("analysis service error", "error", err, "year", year, "round", round, "type", sessionType)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	if dto == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"analysis_not_available","message":"Analysis data has not been ingested for this session yet. Data becomes available approximately 2 hours after session end."}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(dto); err != nil {
		h.logger.Error("analysis encode error", "error", err)
	}
}
