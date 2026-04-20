package ingest

import (
	"fmt"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/domain"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// openF1Session is the raw upstream session shape from OpenF1.
type openF1Session struct {
	SessionKey  int    `json:"session_key"`
	SessionName string `json:"session_name"`
	MeetingKey  int    `json:"meeting_key"`
	DateStart   string `json:"date_start"`
	DateEnd     string `json:"date_end"`
	Year        int    `json:"year"`
}

// openF1Position is the raw upstream position shape from OpenF1.
type openF1Position struct {
	DriverNumber int    `json:"driver_number"`
	Position     int    `json:"position"`
	SessionKey   int    `json:"session_key"`
	MeetingKey   int    `json:"meeting_key"`
	Date         string `json:"date"`
}

// openF1Driver is the raw upstream driver shape from OpenF1.
type openF1Driver struct {
	DriverNumber int    `json:"driver_number"`
	FullName     string `json:"full_name"`
	NameAcronym  string `json:"name_acronym"`
	TeamName     string `json:"team_name"`
	SessionKey   int    `json:"session_key"`
}

// openF1Lap is the raw upstream lap shape from OpenF1.
type openF1Lap struct {
	DriverNumber int     `json:"driver_number"`
	LapDuration  float64 `json:"lap_duration"`
	LapNumber    int     `json:"lap_number"`
	SessionKey   int     `json:"session_key"`
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
		Status:       "completed",
		DateStartUTC: dateStart,
		DateEndUTC:   dateEnd,
		DataAsOfUTC:  time.Now().UTC(),
		Source:       "openf1",
	}
}

// TransformSessionResult converts raw OpenF1 position + driver data to storage SessionResult.
func TransformSessionResult(
	pos openF1Position,
	driver *openF1Driver,
	sessionType domain.SessionType,
	season, round, totalLaps int,
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
		ID:           fmt.Sprintf("%d-%02d-%s-%d", season, round, slug, pos.DriverNumber),
		Type:         "session_result",
		Season:       season,
		Round:        round,
		SessionKey:   pos.SessionKey,
		SessionType:  string(sessionType),
		Position:     pos.Position,
		DriverNumber: pos.DriverNumber,
		DriverName:   driverName,
		DriverAcronym: driverAcronym,
		TeamName:     teamName,
		NumberOfLaps: totalLaps,
		DataAsOfUTC:  time.Now().UTC(),
		Source:       "openf1",
	}

	// For race/sprint types, set finishing status
	if domain.IsRaceType(sessionType) {
		status := string(domain.FinishStatusFinished)
		r.FinishingStatus = &status
	}

	return r
}
