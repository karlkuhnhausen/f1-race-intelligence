// Package storage defines the repository interfaces for the F1 Race Intelligence backend.
package storage

import (
	"context"
	"time"
)

// RaceMeeting represents one F1 season round stored in Cosmos DB.
type RaceMeeting struct {
	ID               string    `json:"id"`
	Season           int       `json:"season"`
	Round            int       `json:"round"`
	RaceName         string    `json:"race_name"`
	CircuitName      string    `json:"circuit_name"`
	CountryName      string    `json:"country_name"`
	StartDatetimeUTC time.Time `json:"start_datetime_utc"`
	EndDatetimeUTC   time.Time `json:"end_datetime_utc"`
	Status           string    `json:"status"`
	IsCancelled      bool      `json:"is_cancelled"`
	CancelledLabel   string    `json:"cancelled_label,omitempty"`
	CancelledReason  string    `json:"cancelled_reason,omitempty"`
	Source           string    `json:"source"`
	DataAsOfUTC      time.Time `json:"data_as_of_utc"`
	SourceHash       string    `json:"source_hash"`
}

// DriverStandingRow represents one row in the drivers championship.
type DriverStandingRow struct {
	ID          string    `json:"id"`
	Season      int       `json:"season"`
	Position    int       `json:"position"`
	DriverName  string    `json:"driver_name"`
	TeamName    string    `json:"team_name"`
	Points      float64   `json:"points"`
	Wins        int       `json:"wins"`
	DataAsOfUTC time.Time `json:"data_as_of_utc"`
	Source      string    `json:"source"`
}

// ConstructorStandingRow represents one row in the constructors championship.
type ConstructorStandingRow struct {
	ID          string    `json:"id"`
	Season      int       `json:"season"`
	Position    int       `json:"position"`
	TeamName    string    `json:"team_name"`
	Points      float64   `json:"points"`
	DataAsOfUTC time.Time `json:"data_as_of_utc"`
	Source      string    `json:"source"`
}

// CalendarRepository defines read/write operations for race meetings.
type CalendarRepository interface {
	UpsertMeeting(ctx context.Context, m RaceMeeting) error
	GetMeetingsBySeason(ctx context.Context, season int) ([]RaceMeeting, error)
	GetMeetingByID(ctx context.Context, season int, id string) (*RaceMeeting, error)
}

// StandingsRepository defines read/write operations for championship standings.
type StandingsRepository interface {
	UpsertDriverStandings(ctx context.Context, rows []DriverStandingRow) error
	GetDriverStandings(ctx context.Context, season int) ([]DriverStandingRow, error)
	UpsertConstructorStandings(ctx context.Context, rows []ConstructorStandingRow) error
	GetConstructorStandings(ctx context.Context, season int) ([]ConstructorStandingRow, error)
}

// Session represents one session within a race weekend stored in Cosmos DB.
type Session struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"` // document type discriminator: "session"
	Season       int       `json:"season"`
	Round        int       `json:"round"`
	MeetingKey   int       `json:"meeting_key"`
	SessionKey   int       `json:"session_key"`
	SessionName  string    `json:"session_name"`
	SessionType  string    `json:"session_type"`
	Status       string    `json:"status"`
	DateStartUTC time.Time `json:"date_start_utc"`
	DateEndUTC   time.Time `json:"date_end_utc"`
	DataAsOfUTC  time.Time `json:"data_as_of_utc"`
	Source       string    `json:"source"`

	// Finalized indicates the session is over and its results/drivers/laps
	// have been fully fetched and cached. Once true, the session poller
	// skips re-fetching from OpenF1.
	Finalized bool `json:"finalized,omitempty"`
	// FinalizedAtUTC is the time the session was first marked finalized.
	FinalizedAtUTC *time.Time `json:"finalized_at_utc,omitempty"`
	// SchemaVersion tracks the cached document layout. If the code's
	// current schema version is newer than the cached value, the
	// finalized flag is treated as stale and the session is re-fetched.
	SchemaVersion int `json:"schema_version,omitempty"`
}

// SessionResult represents one driver's result within a session stored in Cosmos DB.
type SessionResult struct {
	ID            string    `json:"id"`
	Type          string    `json:"type"` // document type discriminator: "session_result"
	Season        int       `json:"season"`
	Round         int       `json:"round"`
	SessionKey    int       `json:"session_key"`
	SessionType   string    `json:"session_type"`
	Position      int       `json:"position"`
	DriverNumber  int       `json:"driver_number"`
	DriverName    string    `json:"driver_name"`
	DriverAcronym string    `json:"driver_acronym"`
	TeamName      string    `json:"team_name"`
	NumberOfLaps  int       `json:"number_of_laps"`
	DataAsOfUTC   time.Time `json:"data_as_of_utc"`
	Source        string    `json:"source"`

	// Race-specific fields
	FinishingStatus *string  `json:"finishing_status,omitempty"`
	RaceTime        *float64 `json:"race_time,omitempty"`
	GapToLeader     *string  `json:"gap_to_leader,omitempty"`
	Points          *float64 `json:"points,omitempty"`
	FastestLap      *bool    `json:"fastest_lap,omitempty"`

	// Qualifying-specific fields
	Q1Time *float64 `json:"q1_time,omitempty"`
	Q2Time *float64 `json:"q2_time,omitempty"`
	Q3Time *float64 `json:"q3_time,omitempty"`

	// Practice-specific fields
	BestLapTime  *float64 `json:"best_lap_time,omitempty"`
	GapToFastest *float64 `json:"gap_to_fastest,omitempty"`
}

// SessionRepository defines read/write operations for sessions and session results.
type SessionRepository interface {
	UpsertSession(ctx context.Context, s Session) error
	UpsertSessionResult(ctx context.Context, r SessionResult) error
	GetSessionsByRound(ctx context.Context, season, round int) ([]Session, error)
	GetSessionResultsByRound(ctx context.Context, season, round int) ([]SessionResult, error)
	// GetFinalizedSessionKeys returns the set of session_key values for the
	// season whose cached document has Finalized=true. The poller uses this
	// as a skip-list so it does not re-fetch results/drivers/laps for sessions
	// that already finished and were fully cached.
	GetFinalizedSessionKeys(ctx context.Context, season int) (map[int]int, error)
}
