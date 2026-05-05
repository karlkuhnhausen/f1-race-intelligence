package ingest

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/domain"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// openF1Session is the raw upstream session shape from OpenF1 /v1/sessions.
type openF1Session struct {
	SessionKey  int    `json:"session_key"`
	SessionName string `json:"session_name"`
	MeetingKey  int    `json:"meeting_key"`
	DateStart   string `json:"date_start"`
	DateEnd     string `json:"date_end"`
	Year        int    `json:"year"`
	IsCancelled bool   `json:"is_cancelled"`
}

// openF1Driver is the raw upstream driver shape from OpenF1 /v1/drivers.
type openF1Driver struct {
	DriverNumber int    `json:"driver_number"`
	FullName     string `json:"full_name"`
	NameAcronym  string `json:"name_acronym"`
	TeamName     string `json:"team_name"`
	SessionKey   int    `json:"session_key"`
}

// openF1SessionResult is the raw upstream shape from OpenF1 /v1/session_result.
//
// duration and gap_to_leader are polymorphic:
//   - Race/Sprint/Practice: a scalar number (total time / best lap; gap in seconds)
//   - Qualifying/SprintQualifying: an array [Q1, Q2, Q3] with nulls for not-reached
//
// We capture them as RawMessage and decode based on the session type.
type openF1SessionResult struct {
	Position     int             `json:"position"`
	DriverNumber int             `json:"driver_number"`
	NumberOfLaps int             `json:"number_of_laps"`
	Points       *float64        `json:"points,omitempty"`
	DNF          bool            `json:"dnf"`
	DNS          bool            `json:"dns"`
	DSQ          bool            `json:"dsq"`
	Duration     json.RawMessage `json:"duration"`
	GapToLeader  json.RawMessage `json:"gap_to_leader"`
	SessionKey   int             `json:"session_key"`
	MeetingKey   int             `json:"meeting_key"`
}

// openF1Lap is the raw upstream lap shape from OpenF1 /v1/laps.
type openF1Lap struct {
	DriverNumber int      `json:"driver_number"`
	LapNumber    int      `json:"lap_number"`
	LapDuration  *float64 `json:"lap_duration"`
	SessionKey   int      `json:"session_key"`
}

// TransformSession converts an OpenF1 session to our storage Session type.
func TransformSession(raw openF1Session, season, round int) storage.Session {
	sessionType := domain.MapOpenF1SessionType(raw.SessionName)
	slug := domain.SessionTypeSlug(sessionType)

	dateStart, _ := time.Parse(time.RFC3339, raw.DateStart)
	dateEnd, _ := time.Parse(time.RFC3339, raw.DateEnd)

	return storage.Session{
		ID:           fmt.Sprintf("%d-%02d-%s", season, round, slug),
		Type:         "session",
		Season:       season,
		Round:        round,
		MeetingKey:   raw.MeetingKey,
		SessionKey:   raw.SessionKey,
		SessionName:  raw.SessionName,
		SessionType:  string(sessionType),
		Status:       deriveStoredStatus(time.Now().UTC(), dateStart, dateEnd),
		DateStartUTC: dateStart,
		DateEndUTC:   dateEnd,
		DataAsOfUTC:  time.Now().UTC(),
		Source:       "openf1",
		// Finalized is set by the poller after results are successfully
		// fetched and the session has ended (see finalizationBuffer).
		SchemaVersion: SessionSchemaVersion,
	}
}

// deriveStoredStatus returns the lifecycle status to persist for a session
// based on its scheduled times. The rounds API service re-derives status at
// read time (see backend/internal/api/rounds/service.go) so this stored value
// is mainly for diagnostics and direct cache inspection — but writing the
// correct value here prevents stale "completed" entries for future sessions.
func deriveStoredStatus(now, dateStart, dateEnd time.Time) string {
	if dateStart.IsZero() {
		return "upcoming"
	}
	if dateStart.After(now) {
		return "upcoming"
	}
	if dateEnd.IsZero() || dateEnd.After(now) {
		return "in_progress"
	}
	return "completed"
}

// TransformSessionResult converts an OpenF1 session_result + driver into our storage SessionResult.
// Fields populated depend on the session type:
//   - Race/Sprint: FinishingStatus, RaceTime, GapToLeader, Points
//   - Qualifying/SprintQualifying: Q1Time, Q2Time, Q3Time
//   - Practice1/2/3: BestLapTime, GapToFastest
func TransformSessionResult(
	raw openF1SessionResult,
	driver *openF1Driver,
	sessionType domain.SessionType,
	season, round int,
) storage.SessionResult {
	slug := domain.SessionTypeSlug(sessionType)

	driverName := ""
	driverAcronym := ""
	teamName := ""
	if driver != nil {
		driverName = driver.FullName
		driverAcronym = driver.NameAcronym
		teamName = driver.TeamName
	}

	r := storage.SessionResult{
		ID:            fmt.Sprintf("%d-%02d-%s-%d", season, round, slug, raw.DriverNumber),
		Type:          "session_result",
		Season:        season,
		Round:         round,
		MeetingKey:    raw.MeetingKey,
		SessionKey:    raw.SessionKey,
		SessionType:   string(sessionType),
		Position:      raw.Position,
		DriverNumber:  raw.DriverNumber,
		DriverName:    driverName,
		DriverAcronym: driverAcronym,
		TeamName:      teamName,
		NumberOfLaps:  raw.NumberOfLaps,
		DataAsOfUTC:   time.Now().UTC(),
		Source:        "openf1",
	}

	switch {
	case domain.IsRaceType(sessionType):
		populateRaceFields(&r, raw)
	case domain.IsQualifyingType(sessionType):
		populateQualifyingFields(&r, raw)
	case domain.IsPracticeType(sessionType):
		populatePracticeFields(&r, raw)
	}

	return r
}

func populateRaceFields(r *storage.SessionResult, raw openF1SessionResult) {
	r.FinishingStatus = derivedFinishingStatus(raw)
	r.Points = raw.Points

	if d, ok := decodeScalar(raw.Duration); ok {
		r.RaceTime = &d
	}
	if g, ok := decodeScalar(raw.GapToLeader); ok {
		r.GapToLeader = formatGapToLeader(r.Position, g)
	}
}

func populateQualifyingFields(r *storage.SessionResult, raw openF1SessionResult) {
	times := decodeArray(raw.Duration)
	if len(times) > 0 {
		r.Q1Time = times[0]
	}
	if len(times) > 1 {
		r.Q2Time = times[1]
	}
	if len(times) > 2 {
		r.Q3Time = times[2]
	}
}

func populatePracticeFields(r *storage.SessionResult, raw openF1SessionResult) {
	if d, ok := decodeScalar(raw.Duration); ok {
		r.BestLapTime = &d
	}
	if g, ok := decodeScalar(raw.GapToLeader); ok {
		r.GapToFastest = &g
	}
}

func derivedFinishingStatus(raw openF1SessionResult) *string {
	var s string
	switch {
	case raw.DNS:
		s = string(domain.FinishStatusDNS)
	case raw.DSQ:
		s = string(domain.FinishStatusDSQ)
	case raw.DNF:
		s = string(domain.FinishStatusDNF)
	default:
		s = string(domain.FinishStatusFinished)
	}
	return &s
}

// formatGapToLeader returns the display string for race gap-to-leader.
// P1 → nil (the table renders RaceTime instead); others → "+X.XXXs".
func formatGapToLeader(position int, gapSeconds float64) *string {
	if position == 1 {
		return nil
	}
	s := fmt.Sprintf("+%.3fs", gapSeconds)
	return &s
}

// decodeScalar decodes a json.RawMessage as a single float64. Returns (0, false)
// if the value is null, missing, or not a number.
func decodeScalar(raw json.RawMessage) (float64, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, false
	}
	var v float64
	if err := json.Unmarshal(raw, &v); err != nil {
		return 0, false
	}
	return v, true
}

// decodeArray decodes a json.RawMessage as []*float64 (nullable elements).
// Returns nil if the value is null, missing, or not an array.
func decodeArray(raw json.RawMessage) []*float64 {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var v []*float64
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil
	}
	return v
}

// DeriveFastestLap returns the driver number with the fastest lap_duration in
// the given laps. Returns (0, false) if no lap times are available.
func DeriveFastestLap(laps []openF1Lap) (int, bool) {
	bestDriver := 0
	bestTime := 0.0
	found := false
	for _, l := range laps {
		if l.LapDuration == nil {
			continue
		}
		if !found || *l.LapDuration < bestTime {
			bestTime = *l.LapDuration
			bestDriver = l.DriverNumber
			found = true
		}
	}
	return bestDriver, found
}
