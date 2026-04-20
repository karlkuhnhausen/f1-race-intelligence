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
