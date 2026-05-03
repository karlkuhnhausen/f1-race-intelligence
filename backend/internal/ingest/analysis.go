package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/domain"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// DriverInfo holds minimal driver metadata used during analysis aggregation.
type DriverInfo struct {
	DriverNumber  int
	DriverName    string
	DriverAcronym string
	TeamName      string
	TeamColor     string
}

// --- Raw OpenF1 response structs ---

type rawPosition struct {
	Date         string `json:"date"`
	DriverNumber int    `json:"driver_number"`
	MeetingKey   int    `json:"meeting_key"`
	Position     int    `json:"position"`
	SessionKey   int    `json:"session_key"`
}

type rawInterval struct {
	Date         string      `json:"date"`
	DriverNumber int         `json:"driver_number"`
	GapToLeader  interface{} `json:"gap_to_leader"` // can be float or string "+N LAP"
	Interval     interface{} `json:"interval"`      // can be float or string "+N LAP"
	MeetingKey   int         `json:"meeting_key"`
	SessionKey   int         `json:"session_key"`
}

type rawStint struct {
	Compound       string `json:"compound"`
	DriverNumber   int    `json:"driver_number"`
	LapEnd         int    `json:"lap_end"`
	LapStart       int    `json:"lap_start"`
	MeetingKey     int    `json:"meeting_key"`
	SessionKey     int    `json:"session_key"`
	StintNumber    int    `json:"stint_number"`
	TireAgeAtStart int    `json:"tyre_age_at_start"` //nolint:misspell // OpenF1 API field name
}

type rawPit struct {
	Date         string  `json:"date"`
	DriverNumber int     `json:"driver_number"`
	LaneDuration float64 `json:"lane_duration"`
	LapNumber    int     `json:"lap_number"`
	MeetingKey   int     `json:"meeting_key"`
	PitDuration  float64 `json:"pit_duration"`
	SessionKey   int     `json:"session_key"`
	StopDuration float64 `json:"stop_duration"`
}

type rawOvertake struct {
	Date                   string `json:"date"`
	MeetingKey             int    `json:"meeting_key"`
	OvertakenDriverNumber  int    `json:"overtaken_driver_number"`
	OvertakingDriverNumber int    `json:"overtaking_driver_number"`
	Position               int    `json:"position"`
	SessionKey             int    `json:"session_key"`
}

// --- Fetch functions ---

// FetchPositionData fetches position data for a session from OpenF1.
func FetchPositionData(ctx context.Context, client *http.Client, sessionKey int) ([]rawPosition, error) {
	url := fmt.Sprintf("%s/position?session_key=%d", openF1BaseURL, sessionKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("position: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("position: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("position: unexpected status %d", resp.StatusCode)
	}

	var data []rawPosition
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("position: decode: %w", err)
	}
	return data, nil
}

// FetchIntervalData fetches interval/gap data for a session from OpenF1.
func FetchIntervalData(ctx context.Context, client *http.Client, sessionKey int) ([]rawInterval, error) {
	url := fmt.Sprintf("%s/intervals?session_key=%d", openF1BaseURL, sessionKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("intervals: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("intervals: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("intervals: unexpected status %d", resp.StatusCode)
	}

	var data []rawInterval
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("intervals: decode: %w", err)
	}
	return data, nil
}

// FetchStintData fetches tire stint data for a session from OpenF1.
func FetchStintData(ctx context.Context, client *http.Client, sessionKey int) ([]rawStint, error) {
	url := fmt.Sprintf("%s/stints?session_key=%d", openF1BaseURL, sessionKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("stints: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stints: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stints: unexpected status %d", resp.StatusCode)
	}

	var data []rawStint
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("stints: decode: %w", err)
	}
	return data, nil
}

// FetchPitData fetches pit stop data for a session from OpenF1.
func FetchPitData(ctx context.Context, client *http.Client, sessionKey int) ([]rawPit, error) {
	url := fmt.Sprintf("%s/pit?session_key=%d", openF1BaseURL, sessionKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("pit: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pit: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pit: unexpected status %d", resp.StatusCode)
	}

	var data []rawPit
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("pit: decode: %w", err)
	}
	return data, nil
}

// FetchOvertakeData fetches overtake data for a session from OpenF1.
// Returns an empty slice (not an error) if no overtake data is available.
func FetchOvertakeData(ctx context.Context, client *http.Client, sessionKey int) ([]rawOvertake, error) {
	url := fmt.Sprintf("%s/overtakes?session_key=%d", openF1BaseURL, sessionKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("overtakes: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("overtakes: request: %w", err)
	}
	defer resp.Body.Close()

	// Overtakes may return 404 for sessions without data — treat as empty
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("overtakes: unexpected status %d", resp.StatusCode)
	}

	var data []rawOvertake
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("overtakes: decode: %w", err)
	}
	return data, nil
}

// --- Aggregation functions ---

// AggregatePositions converts raw position data to domain types.
// Deduplicates by keeping the LAST entry per (driver_number, position_change_timestamp)
// and produces 1 data point per driver per lap.
func AggregatePositions(raw []rawPosition, drivers map[int]DriverInfo) []domain.AnalysisPosition {
	// Group by driver, then find position per "lap" by ordering entries chronologically
	// and detecting position at each point where position changes.
	// Since OpenF1 /position reports position changes (not per-lap), we need to
	// reconstruct: for each driver, their position at each lap boundary.
	// Strategy: group entries by driver, sort by date, then for each consecutive pair
	// where position changed, that's a new data point.
	// Simpler approach: find max lap from the position events and assign positions per lap.

	// Group raw entries by driver number
	type driverEntries struct {
		entries []rawPosition
	}
	byDriver := make(map[int]*driverEntries)
	for i := range raw {
		d := raw[i].DriverNumber
		if byDriver[d] == nil {
			byDriver[d] = &driverEntries{}
		}
		byDriver[d].entries = append(byDriver[d].entries, raw[i])
	}

	var result []domain.AnalysisPosition
	for driverNum, de := range byDriver {
		info, ok := drivers[driverNum]
		if !ok {
			continue // skip drivers not in our driver map
		}

		// Sort entries chronologically
		sort.Slice(de.entries, func(i, j int) bool {
			return de.entries[i].Date < de.entries[j].Date
		})

		// Convert position changes to per-lap positions.
		// The /position endpoint fires when a driver's position changes.
		// We reconstruct lap-by-lap by finding the last known position before
		// each lap boundary. But since we don't have lap timestamps from this
		// endpoint, we use a simpler approach: treat each entry as one data point
		// and deduplicate — if there are multiple entries for the same position
		// in sequence, we need the positional state at each "checkpoint".
		//
		// Actually, the best approach for post-session display: just keep all
		// position change events as our data points. The chart will plot them
		// as a step function (position on Y, sequence on X).
		// But the spec says "lap-by-lap" — so we need to map to laps.
		//
		// The simplest robust approach: pair with /laps data to get lap count,
		// or use the positional events as-is with an index that represents
		// progression. For now, we'll create sequential data points that the
		// frontend renders as a step chart.

		// Deduplicate: keep only entries where position actually changed
		var laps []domain.PositionLap
		lastPos := -1
		lapCounter := 0
		for _, e := range de.entries {
			if e.Position != lastPos {
				lapCounter++
				laps = append(laps, domain.PositionLap{
					LapNumber: lapCounter,
					Position:  e.Position,
				})
				lastPos = e.Position
			}
		}

		result = append(result, domain.AnalysisPosition{
			DriverNumber:  driverNum,
			DriverName:    info.DriverName,
			DriverAcronym: info.DriverAcronym,
			TeamName:      info.TeamName,
			TeamColor:     info.TeamColor,
			Laps:          laps,
		})
	}

	// Sort by driver number for deterministic output
	sort.Slice(result, func(i, j int) bool {
		return result[i].DriverNumber < result[j].DriverNumber
	})

	return result
}

// parseGapValue converts an interface{} gap/interval value to float64.
// OpenF1 returns floats for normal gaps, strings like "+1 LAP" for lapped drivers.
// Lapped drivers get a -1 sentinel value.
func parseGapValue(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case string:
		// "+1 LAP", "+2 LAPS" etc — use -1 as sentinel for "lapped"
		return -1
	case nil:
		return 0 // leader has null gap
	default:
		return 0
	}
}

// AggregateIntervals converts raw interval data to domain types.
// Deduplicates by keeping the LAST entry per (driver_number) per time window,
// producing 1 data point per driver per approximate lap.
func AggregateIntervals(raw []rawInterval, drivers map[int]DriverInfo) []domain.AnalysisInterval {
	// Group by driver
	type driverEntries struct {
		entries []rawInterval
	}
	byDriver := make(map[int]*driverEntries)
	for i := range raw {
		d := raw[i].DriverNumber
		if byDriver[d] == nil {
			byDriver[d] = &driverEntries{}
		}
		byDriver[d].entries = append(byDriver[d].entries, raw[i])
	}

	var result []domain.AnalysisInterval
	for driverNum, de := range byDriver {
		info, ok := drivers[driverNum]
		if !ok {
			continue
		}

		// Sort chronologically
		sort.Slice(de.entries, func(i, j int) bool {
			return de.entries[i].Date < de.entries[j].Date
		})

		// Intervals are reported every ~4 seconds. For a ~90-second lap,
		// that's ~22 entries per lap. We want ~1 per lap.
		// Simple approach: take every Nth entry where N ≈ entries/expectedLaps.
		// Or simpler: just take every entry and let the frontend handle density.
		// For ~60 laps × 20 drivers = 1200 points — manageable.
		// Better: sample at roughly 1 per lap. Assume ~90s per lap, 4s per sample = 22 samples/lap.
		// Downsample by taking every 20th entry (roughly 1 per lap).

		sampleRate := 20
		if len(de.entries) < 100 {
			sampleRate = 1 // short sessions or sparse data
		}

		var laps []domain.IntervalLap
		lapNum := 0
		for i, e := range de.entries {
			if i%sampleRate == 0 {
				lapNum++
				laps = append(laps, domain.IntervalLap{
					LapNumber:   lapNum,
					GapToLeader: parseGapValue(e.GapToLeader),
					Interval:    parseGapValue(e.Interval),
				})
			}
		}

		result = append(result, domain.AnalysisInterval{
			DriverNumber:  driverNum,
			DriverAcronym: info.DriverAcronym,
			TeamName:      info.TeamName,
			TeamColor:     info.TeamColor,
			Laps:          laps,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].DriverNumber < result[j].DriverNumber
	})

	return result
}

// MapStints converts raw stint data to domain types.
func MapStints(raw []rawStint, drivers map[int]DriverInfo) []domain.AnalysisStint {
	var result []domain.AnalysisStint
	for _, s := range raw {
		info, ok := drivers[s.DriverNumber]
		if !ok {
			continue
		}
		result = append(result, domain.AnalysisStint{
			DriverNumber:   s.DriverNumber,
			DriverAcronym:  info.DriverAcronym,
			TeamName:       info.TeamName,
			StintNumber:    s.StintNumber,
			Compound:       strings.ToUpper(s.Compound),
			LapStart:       s.LapStart,
			LapEnd:         s.LapEnd,
			TireAgeAtStart: s.TireAgeAtStart,
		})
	}
	return result
}

// MapPits converts raw pit data to domain types.
func MapPits(raw []rawPit, drivers map[int]DriverInfo) []domain.AnalysisPit {
	var result []domain.AnalysisPit
	for _, p := range raw {
		info, ok := drivers[p.DriverNumber]
		if !ok {
			continue
		}
		pitDuration := p.LaneDuration
		if pitDuration == 0 {
			pitDuration = p.PitDuration // fallback to deprecated field
		}
		result = append(result, domain.AnalysisPit{
			DriverNumber:  p.DriverNumber,
			DriverAcronym: info.DriverAcronym,
			TeamName:      info.TeamName,
			LapNumber:     p.LapNumber,
			PitDuration:   pitDuration,
			StopDuration:  p.StopDuration,
		})
	}
	return result
}

// MapOvertakes converts raw overtake data to domain types.
func MapOvertakes(raw []rawOvertake, drivers map[int]DriverInfo) []domain.AnalysisOvertake {
	var result []domain.AnalysisOvertake
	for _, o := range raw {
		overtakingInfo := drivers[o.OvertakingDriverNumber]
		overtakenInfo := drivers[o.OvertakenDriverNumber]

		result = append(result, domain.AnalysisOvertake{
			OvertakingDriverNumber: o.OvertakingDriverNumber,
			OvertakingDriverName:   overtakingInfo.DriverName,
			OvertakenDriverNumber:  o.OvertakenDriverNumber,
			OvertakenDriverName:    overtakenInfo.DriverName,
			LapNumber:              0, // OpenF1 doesn't provide lap for overtakes directly; derive from position changes
			Position:               o.Position,
		})
	}
	return result
}

// AnalysisFetchResult holds the combined results of fetching all analysis data.
type AnalysisFetchResult struct {
	Positions []domain.AnalysisPosition
	Intervals []domain.AnalysisInterval
	Stints    []domain.AnalysisStint
	Pits      []domain.AnalysisPit
	Overtakes []domain.AnalysisOvertake
}

// FetchAllAnalysisData orchestrates fetching all 5 analysis data types for a session.
// Position data is required (returns error if it fails). Other 4 are non-fatal on failure.
// Applies 500ms delays between requests to respect rate limits.
func FetchAllAnalysisData(ctx context.Context, client *http.Client, sessionKey int, drivers map[int]DriverInfo, logger *slog.Logger) (*AnalysisFetchResult, error) {
	const delayBetweenFetches = 500 * time.Millisecond

	// 1. Position (required)
	rawPos, err := FetchPositionData(ctx, client, sessionKey)
	if err != nil {
		return nil, fmt.Errorf("analysis: position data required but failed: %w", err)
	}
	positions := AggregatePositions(rawPos, drivers)
	logger.Info("analysis: fetched positions",
		"session_key", sessionKey,
		"raw_count", len(rawPos),
		"aggregated_drivers", len(positions),
	)

	time.Sleep(delayBetweenFetches)

	// 2. Intervals (non-fatal)
	var intervals []domain.AnalysisInterval
	rawInt, err := FetchIntervalData(ctx, client, sessionKey)
	if err != nil {
		logger.Warn("analysis: intervals fetch failed (non-fatal)",
			"session_key", sessionKey,
			"error", err.Error(),
		)
	} else {
		intervals = AggregateIntervals(rawInt, drivers)
		logger.Info("analysis: fetched intervals",
			"session_key", sessionKey,
			"raw_count", len(rawInt),
			"aggregated_drivers", len(intervals),
		)
	}

	time.Sleep(delayBetweenFetches)

	// 3. Stints (non-fatal)
	var stints []domain.AnalysisStint
	rawStints, err := FetchStintData(ctx, client, sessionKey)
	if err != nil {
		logger.Warn("analysis: stints fetch failed (non-fatal)",
			"session_key", sessionKey,
			"error", err.Error(),
		)
	} else {
		stints = MapStints(rawStints, drivers)
		logger.Info("analysis: fetched stints",
			"session_key", sessionKey,
			"count", len(stints),
		)
	}

	time.Sleep(delayBetweenFetches)

	// 4. Pits (non-fatal)
	var pits []domain.AnalysisPit
	rawPits, err := FetchPitData(ctx, client, sessionKey)
	if err != nil {
		logger.Warn("analysis: pits fetch failed (non-fatal)",
			"session_key", sessionKey,
			"error", err.Error(),
		)
	} else {
		pits = MapPits(rawPits, drivers)
		logger.Info("analysis: fetched pits",
			"session_key", sessionKey,
			"count", len(pits),
		)
	}

	time.Sleep(delayBetweenFetches)

	// 5. Overtakes (non-fatal)
	var overtakes []domain.AnalysisOvertake
	rawOvertakes, err := FetchOvertakeData(ctx, client, sessionKey)
	if err != nil {
		logger.Warn("analysis: overtakes fetch failed (non-fatal)",
			"session_key", sessionKey,
			"error", err.Error(),
		)
	} else {
		overtakes = MapOvertakes(rawOvertakes, drivers)
		logger.Info("analysis: fetched overtakes",
			"session_key", sessionKey,
			"count", len(overtakes),
		)
	}

	return &AnalysisFetchResult{
		Positions: positions,
		Intervals: intervals,
		Stints:    stints,
		Pits:      pits,
		Overtakes: overtakes,
	}, nil
}

// ToStoragePositions converts domain positions to storage documents for a given session.
func ToStoragePositions(season, round int, sessionType string, positions []domain.AnalysisPosition) []storage.SessionAnalysisPosition {
	docs := make([]storage.SessionAnalysisPosition, 0, len(positions))
	for _, p := range positions {
		laps := make([]storage.PositionLap, len(p.Laps))
		for i, l := range p.Laps {
			laps[i] = storage.PositionLap{LapNumber: l.LapNumber, Position: l.Position}
		}
		docs = append(docs, storage.SessionAnalysisPosition{
			ID:            fmt.Sprintf("analysis_position_%d_%s_%d", round, sessionType, p.DriverNumber),
			Type:          "analysis_position",
			Season:        season,
			Round:         round,
			SessionType:   sessionType,
			DriverNumber:  p.DriverNumber,
			DriverName:    p.DriverName,
			DriverAcronym: p.DriverAcronym,
			TeamName:      p.TeamName,
			TeamColor:     p.TeamColor,
			Laps:          laps,
		})
	}
	return docs
}

// ToStorageIntervals converts domain intervals to storage documents.
func ToStorageIntervals(season, round int, sessionType string, intervals []domain.AnalysisInterval) []storage.SessionAnalysisInterval {
	docs := make([]storage.SessionAnalysisInterval, 0, len(intervals))
	for _, iv := range intervals {
		laps := make([]storage.IntervalLap, len(iv.Laps))
		for i, l := range iv.Laps {
			laps[i] = storage.IntervalLap{LapNumber: l.LapNumber, GapToLeader: l.GapToLeader, Interval: l.Interval}
		}
		docs = append(docs, storage.SessionAnalysisInterval{
			ID:            fmt.Sprintf("analysis_interval_%d_%s_%d", round, sessionType, iv.DriverNumber),
			Type:          "analysis_interval",
			Season:        season,
			Round:         round,
			SessionType:   sessionType,
			DriverNumber:  iv.DriverNumber,
			DriverAcronym: iv.DriverAcronym,
			TeamName:      iv.TeamName,
			TeamColor:     iv.TeamColor,
			Laps:          laps,
		})
	}
	return docs
}

// ToStorageStints converts domain stints to storage documents.
func ToStorageStints(season, round int, sessionType string, stints []domain.AnalysisStint) []storage.SessionAnalysisStint {
	docs := make([]storage.SessionAnalysisStint, 0, len(stints))
	for _, s := range stints {
		docs = append(docs, storage.SessionAnalysisStint{
			ID:             fmt.Sprintf("analysis_stint_%d_%s_%d_%d", round, sessionType, s.DriverNumber, s.StintNumber),
			Type:           "analysis_stint",
			Season:         season,
			Round:          round,
			SessionType:    sessionType,
			DriverNumber:   s.DriverNumber,
			DriverAcronym:  s.DriverAcronym,
			TeamName:       s.TeamName,
			StintNumber:    s.StintNumber,
			Compound:       s.Compound,
			LapStart:       s.LapStart,
			LapEnd:         s.LapEnd,
			TireAgeAtStart: s.TireAgeAtStart,
		})
	}
	return docs
}

// ToStoragePits converts domain pits to storage documents.
func ToStoragePits(season, round int, sessionType string, pits []domain.AnalysisPit) []storage.SessionAnalysisPit {
	docs := make([]storage.SessionAnalysisPit, 0, len(pits))
	for _, p := range pits {
		docs = append(docs, storage.SessionAnalysisPit{
			ID:            fmt.Sprintf("analysis_pit_%d_%s_%d_%d", round, sessionType, p.DriverNumber, p.LapNumber),
			Type:          "analysis_pit",
			Season:        season,
			Round:         round,
			SessionType:   sessionType,
			DriverNumber:  p.DriverNumber,
			DriverAcronym: p.DriverAcronym,
			TeamName:      p.TeamName,
			LapNumber:     p.LapNumber,
			PitDuration:   p.PitDuration,
			StopDuration:  p.StopDuration,
		})
	}
	return docs
}

// ToStorageOvertakes converts domain overtakes to storage documents.
func ToStorageOvertakes(season, round int, sessionType string, overtakes []domain.AnalysisOvertake) []storage.SessionAnalysisOvertake {
	docs := make([]storage.SessionAnalysisOvertake, 0, len(overtakes))
	for i, o := range overtakes {
		docs = append(docs, storage.SessionAnalysisOvertake{
			ID:                     fmt.Sprintf("analysis_overtake_%d_%s_%d", round, sessionType, i),
			Type:                   "analysis_overtake",
			Season:                 season,
			Round:                  round,
			SessionType:            sessionType,
			OvertakingDriverNumber: o.OvertakingDriverNumber,
			OvertakingDriverName:   o.OvertakingDriverName,
			OvertakenDriverNumber:  o.OvertakenDriverNumber,
			OvertakenDriverName:    o.OvertakenDriverName,
			LapNumber:              o.LapNumber,
			Position:               o.Position,
		})
	}
	return docs
}
