// Package standings provides championship data ingestion from OpenF1.
package standings

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

const (
	openF1BaseURL = "https://api.openf1.org/v1"
)

// ChampionshipIngester fetches championship standings, session results, and
// starting grids from OpenF1 and upserts them into Cosmos DB.
type ChampionshipIngester struct {
	repo   storage.ChampionshipRepository
	client *http.Client
	logger *slog.Logger
}

// NewChampionshipIngester creates a new championship data ingester.
func NewChampionshipIngester(repo storage.ChampionshipRepository, logger *slog.Logger) *ChampionshipIngester {
	return &ChampionshipIngester{
		repo:   repo,
		client: &http.Client{Timeout: 30 * time.Second},
		logger: logger,
	}
}

// IngestSession fetches and stores championship data for a single race/sprint session.
func (ci *ChampionshipIngester) IngestSession(ctx context.Context, season, sessionKey, meetingKey int) error {
	start := time.Now()
	ci.logger.Info("championship ingestion starting",
		"season", season,
		"session_key", sessionKey,
		"meeting_key", meetingKey,
	)

	// Fetch championship drivers
	driverSnapshots, err := ci.fetchDriverChampionship(ctx, season, sessionKey, meetingKey)
	if err != nil {
		ci.logger.Error("championship drivers fetch failed", "session_key", sessionKey, "error", err)
		return fmt.Errorf("championship drivers: %w", err)
	}
	if len(driverSnapshots) > 0 {
		if err := ci.repo.UpsertDriverChampionshipSnapshots(ctx, driverSnapshots); err != nil {
			return fmt.Errorf("upsert driver snapshots: %w", err)
		}
	}

	// Fetch championship teams
	teamSnapshots, err := ci.fetchTeamChampionship(ctx, season, sessionKey, meetingKey)
	if err != nil {
		ci.logger.Error("championship teams fetch failed", "session_key", sessionKey, "error", err)
		return fmt.Errorf("championship teams: %w", err)
	}
	if len(teamSnapshots) > 0 {
		if err := ci.repo.UpsertTeamChampionshipSnapshots(ctx, teamSnapshots); err != nil {
			return fmt.Errorf("upsert team snapshots: %w", err)
		}
	}

	// Fetch session results
	results, err := ci.fetchSessionResults(ctx, season, sessionKey, meetingKey)
	if err != nil {
		ci.logger.Error("session results fetch failed", "session_key", sessionKey, "error", err)
		return fmt.Errorf("session results: %w", err)
	}
	if len(results) > 0 {
		if err := ci.repo.UpsertChampionshipSessionResults(ctx, results); err != nil {
			return fmt.Errorf("upsert session results: %w", err)
		}
	}

	// Fetch starting grid
	gridEntries, err := ci.fetchStartingGrid(ctx, season, meetingKey)
	if err != nil {
		ci.logger.Warn("starting grid fetch failed (non-fatal)", "meeting_key", meetingKey, "error", err)
		// Non-fatal: starting grid may not be available for all meetings
	} else if len(gridEntries) > 0 {
		if err := ci.repo.UpsertStartingGridEntries(ctx, gridEntries); err != nil {
			return fmt.Errorf("upsert starting grid: %w", err)
		}
	}

	duration := time.Since(start)
	ci.logger.Info("championship ingestion complete",
		"session_key", sessionKey,
		"meeting_key", meetingKey,
		"drivers", len(driverSnapshots),
		"teams", len(teamSnapshots),
		"results", len(results),
		"grid_entries", len(gridEntries),
		"duration", duration,
	)
	return nil
}

// --- OpenF1 response types ---

type openF1ChampionshipDriver struct {
	DriverNumber    int      `json:"driver_number"`
	MeetingKey      int      `json:"meeting_key"`
	SessionKey      int      `json:"session_key"`
	PositionStart   *int     `json:"position_start"`
	PositionCurrent int      `json:"position_current"`
	PointsStart     *float64 `json:"points_start"`
	PointsCurrent   float64  `json:"points_current"`
}

type openF1ChampionshipTeam struct {
	TeamName        *string  `json:"team_name"`
	MeetingKey      int      `json:"meeting_key"`
	SessionKey      int      `json:"session_key"`
	PositionStart   *int     `json:"position_start"`
	PositionCurrent int      `json:"position_current"`
	PointsStart     *float64 `json:"points_start"`
	PointsCurrent   float64  `json:"points_current"`
}

type openF1SessionResult struct {
	Position     int              `json:"position"`
	DriverNumber int              `json:"driver_number"`
	DNF          bool             `json:"dnf"`
	DNS          bool             `json:"dns"`
	DSQ          bool             `json:"dsq"`
	Points       *float64         `json:"points"`
	NumberOfLaps int              `json:"number_of_laps"`
	GapToLeader  *json.Number     `json:"gap_to_leader"`
	Duration     *float64         `json:"duration"`
}

type openF1StartingGrid struct {
	Position    int      `json:"position"`
	DriverNum   int      `json:"driver_number"`
	LapDuration *float64 `json:"lap_duration"`
}

// --- Fetch and transform methods ---

func (ci *ChampionshipIngester) fetchDriverChampionship(ctx context.Context, season, sessionKey, meetingKey int) ([]storage.DriverChampionshipSnapshot, error) {
	url := fmt.Sprintf("%s/championship_drivers?session_key=%d", openF1BaseURL, sessionKey)
	var raw []openF1ChampionshipDriver
	if err := ci.fetchJSON(ctx, url, &raw); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	snapshots := make([]storage.DriverChampionshipSnapshot, 0, len(raw))
	for _, r := range raw {
		snapshots = append(snapshots, storage.DriverChampionshipSnapshot{
			ID:              fmt.Sprintf("%d-champ-driver-%d-%d", season, sessionKey, r.DriverNumber),
			Type:            "championship_driver",
			Season:          season,
			SessionKey:      sessionKey,
			MeetingKey:      meetingKey,
			DriverNumber:    r.DriverNumber,
			PositionStart:   r.PositionStart,
			PositionCurrent: r.PositionCurrent,
			PointsStart:     r.PointsStart,
			PointsCurrent:   r.PointsCurrent,
			DataAsOfUTC:     now,
			Source:          "openf1",
		})
	}
	return snapshots, nil
}

func (ci *ChampionshipIngester) fetchTeamChampionship(ctx context.Context, season, sessionKey, meetingKey int) ([]storage.TeamChampionshipSnapshot, error) {
	url := fmt.Sprintf("%s/championship_teams?session_key=%d", openF1BaseURL, sessionKey)
	var raw []openF1ChampionshipTeam
	if err := ci.fetchJSON(ctx, url, &raw); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	snapshots := make([]storage.TeamChampionshipSnapshot, 0, len(raw))
	for _, r := range raw {
		teamName := "Unknown"
		if r.TeamName != nil && *r.TeamName != "" {
			teamName = *r.TeamName
		}
		snapshots = append(snapshots, storage.TeamChampionshipSnapshot{
			ID:              fmt.Sprintf("%d-champ-team-%d-%s", season, sessionKey, slugify(teamName)),
			Type:            "championship_team",
			Season:          season,
			SessionKey:      sessionKey,
			MeetingKey:      meetingKey,
			TeamName:        teamName,
			TeamSlug:        slugify(teamName),
			PositionStart:   r.PositionStart,
			PositionCurrent: r.PositionCurrent,
			PointsStart:     r.PointsStart,
			PointsCurrent:   r.PointsCurrent,
			DataAsOfUTC:     now,
			Source:          "openf1",
		})
	}
	return snapshots, nil
}

func (ci *ChampionshipIngester) fetchSessionResults(ctx context.Context, season, sessionKey, meetingKey int) ([]storage.ChampionshipSessionResult, error) {
	url := fmt.Sprintf("%s/session_result?session_key=%d", openF1BaseURL, sessionKey)
	var raw []openF1SessionResult
	if err := ci.fetchJSON(ctx, url, &raw); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	results := make([]storage.ChampionshipSessionResult, 0, len(raw))
	for _, r := range raw {
		points := 0.0
		if r.Points != nil {
			points = *r.Points
		}
		gap := ""
		if r.GapToLeader != nil {
			gap = r.GapToLeader.String()
		}
		duration := 0.0
		if r.Duration != nil {
			duration = *r.Duration
		}
		results = append(results, storage.ChampionshipSessionResult{
			ID:           fmt.Sprintf("%d-result-%d-%d", season, sessionKey, r.DriverNumber),
			Type:         "championship_result",
			Season:       season,
			SessionKey:   sessionKey,
			MeetingKey:   meetingKey,
			DriverNumber: r.DriverNumber,
			Position:     r.Position,
			Points:       points,
			DNF:          r.DNF,
			DNS:          r.DNS,
			DSQ:          r.DSQ,
			NumberOfLaps: r.NumberOfLaps,
			GapToLeader:  gap,
			Duration:     duration,
			DataAsOfUTC:  now,
			Source:       "openf1",
		})
	}
	return results, nil
}

func (ci *ChampionshipIngester) fetchStartingGrid(ctx context.Context, season, meetingKey int) ([]storage.StartingGridEntry, error) {
	url := fmt.Sprintf("%s/starting_grid?meeting_key=%d", openF1BaseURL, meetingKey)
	var raw []openF1StartingGrid
	if err := ci.fetchJSON(ctx, url, &raw); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	entries := make([]storage.StartingGridEntry, 0, len(raw))
	for _, r := range raw {
		lapDur := 0.0
		if r.LapDuration != nil {
			lapDur = *r.LapDuration
		}
		entries = append(entries, storage.StartingGridEntry{
			ID:           fmt.Sprintf("%d-grid-%d-%d", season, meetingKey, r.DriverNum),
			Type:         "starting_grid",
			Season:       season,
			MeetingKey:   meetingKey,
			DriverNumber: r.DriverNum,
			Position:     r.Position,
			LapDuration:  lapDur,
			DataAsOfUTC:  now,
			Source:       "openf1",
		})
	}
	return entries, nil
}

// --- Helpers ---

func (ci *ChampionshipIngester) fetchJSON(ctx context.Context, url string, target interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := ci.client.Do(req)
	if err != nil {
		return fmt.Errorf("request %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("not found: %s", url)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode response from %s: %w", url, err)
	}
	return nil
}

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	lower := strings.ToLower(s)
	slug := nonAlphanumeric.ReplaceAllString(lower, "-")
	return strings.Trim(slug, "-")
}
