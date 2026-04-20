// Package standings provides the Hyprace standings polling client.
package standings

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

const (
	// DefaultStandingsInterval is the polling cadence for standings data.
	DefaultStandingsInterval = 5 * time.Minute

	// hypraceBaseURL is the Hyprace API base.
	hypraceBaseURL = "https://api.hyprace.com/v1"
)

// hypraceDriverRow is the raw upstream driver standing shape.
type hypraceDriverRow struct {
	Position   int     `json:"position"`
	DriverName string  `json:"driver_name"`
	TeamName   string  `json:"team_name"`
	Points     float64 `json:"points"`
	Wins       int     `json:"wins"`
}

// hypraceConstructorRow is the raw upstream constructor standing shape.
type hypraceConstructorRow struct {
	Position int     `json:"position"`
	TeamName string  `json:"team_name"`
	Points   float64 `json:"points"`
}

// HypraceClient polls the Hyprace API for driver and constructor standings
// and upserts results into the StandingsRepository.
type HypraceClient struct {
	repo     storage.StandingsRepository
	client   *http.Client
	interval time.Duration
	logger   *slog.Logger

	mu       sync.Mutex
	lastPoll time.Time
	lastErr  error
}

// NewHypraceClient creates a new Hyprace standings poller.
func NewHypraceClient(repo storage.StandingsRepository, logger *slog.Logger) *HypraceClient {
	return &HypraceClient{
		repo:     repo,
		client:   &http.Client{Timeout: 30 * time.Second},
		interval: DefaultStandingsInterval,
		logger:   logger,
	}
}

// Start begins the polling loop. It blocks until the context is cancelled.
func (h *HypraceClient) Start(ctx context.Context, season int) {
	h.logger.Info("hyprace poller starting", "season", season, "interval", h.interval)

	// Run immediately on start, then on ticker.
	h.poll(ctx, season)

	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			h.logger.Info("hyprace poller stopped")
			return
		case <-ticker.C:
			h.poll(ctx, season)
		}
	}
}

// poll fetches both driver and constructor standings.
func (h *HypraceClient) poll(ctx context.Context, season int) {
	h.logger.Debug("hyprace poll starting", "season", season)

	dErr := h.pollDrivers(ctx, season)
	cErr := h.pollConstructors(ctx, season)

	now := time.Now().UTC()
	h.mu.Lock()
	h.lastPoll = now
	if dErr != nil {
		h.lastErr = dErr
	} else if cErr != nil {
		h.lastErr = cErr
	} else {
		h.lastErr = nil
	}
	h.mu.Unlock()

	if dErr != nil {
		h.logger.Error("hyprace drivers poll failed", "error", dErr)
	}
	if cErr != nil {
		h.logger.Error("hyprace constructors poll failed", "error", cErr)
	}
	if dErr == nil && cErr == nil {
		h.logger.Info("hyprace poll complete", "season", season)
	}
}

// pollDrivers fetches and upserts driver standings.
func (h *HypraceClient) pollDrivers(ctx context.Context, season int) error {
	url := fmt.Sprintf("%s/standings/drivers?year=%d", hypraceBaseURL, season)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("hyprace: build drivers request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("hyprace: drivers request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("hyprace: drivers unexpected status %d", resp.StatusCode)
	}

	var raw []hypraceDriverRow
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return fmt.Errorf("hyprace: decode drivers: %w", err)
	}

	now := time.Now().UTC()
	rows := make([]storage.DriverStandingRow, 0, len(raw))
	for _, r := range raw {
		rows = append(rows, storage.DriverStandingRow{
			ID:          fmt.Sprintf("%d-driver-%d", season, r.Position),
			Season:      season,
			Position:    r.Position,
			DriverName:  r.DriverName,
			TeamName:    r.TeamName,
			Points:      r.Points,
			Wins:        r.Wins,
			DataAsOfUTC: now,
			Source:      "hyprace",
		})
	}

	return h.repo.UpsertDriverStandings(ctx, rows)
}

// pollConstructors fetches and upserts constructor standings.
func (h *HypraceClient) pollConstructors(ctx context.Context, season int) error {
	url := fmt.Sprintf("%s/standings/constructors?year=%d", hypraceBaseURL, season)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("hyprace: build constructors request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("hyprace: constructors request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("hyprace: constructors unexpected status %d", resp.StatusCode)
	}

	var raw []hypraceConstructorRow
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return fmt.Errorf("hyprace: decode constructors: %w", err)
	}

	now := time.Now().UTC()
	rows := make([]storage.ConstructorStandingRow, 0, len(raw))
	for _, r := range raw {
		rows = append(rows, storage.ConstructorStandingRow{
			ID:          fmt.Sprintf("%d-constructor-%d", season, r.Position),
			Season:      season,
			Position:    r.Position,
			TeamName:    r.TeamName,
			Points:      r.Points,
			DataAsOfUTC: now,
			Source:      "hyprace",
		})
	}

	return h.repo.UpsertConstructorStandings(ctx, rows)
}

// LastPoll returns the time of the last successful poll and any error from the most recent attempt.
func (h *HypraceClient) LastPoll() (time.Time, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.lastPoll, h.lastErr
}
