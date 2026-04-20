// Package rounds implements the round detail API service and response shaping.
package rounds

import "time"

// SessionResultDTO is the API response shape for a single driver result.
type SessionResultDTO struct {
	Position      int    `json:"position"`
	DriverNumber  int    `json:"driver_number"`
	DriverName    string `json:"driver_name"`
	DriverAcronym string `json:"driver_acronym"`
	TeamName      string `json:"team_name"`
	NumberOfLaps  int    `json:"number_of_laps"`

	// Race-specific
	FinishingStatus *string  `json:"finishing_status,omitempty"`
	RaceTime        *float64 `json:"race_time,omitempty"`
	GapToLeader     *string  `json:"gap_to_leader,omitempty"`
	Points          *float64 `json:"points,omitempty"`
	FastestLap      *bool    `json:"fastest_lap,omitempty"`

	// Qualifying-specific
	Q1Time *float64 `json:"q1_time,omitempty"`
	Q2Time *float64 `json:"q2_time,omitempty"`
	Q3Time *float64 `json:"q3_time,omitempty"`

	// Practice-specific
	BestLapTime  *float64 `json:"best_lap_time,omitempty"`
	GapToFastest *float64 `json:"gap_to_fastest,omitempty"`
}

// SessionDetailDTO is the API response shape for one session within a round.
type SessionDetailDTO struct {
	SessionName string             `json:"session_name"`
	SessionType string             `json:"session_type"`
	Status      string             `json:"status"`
	DateStart   time.Time          `json:"date_start_utc"`
	DateEnd     time.Time          `json:"date_end_utc"`
	Results     []SessionResultDTO `json:"results"`
}

// RoundDetailResponse is the full API response for GET /api/v1/rounds/{round}.
type RoundDetailResponse struct {
	Year        int                `json:"year"`
	Round       int                `json:"round"`
	RaceName    string             `json:"race_name"`
	CircuitName string             `json:"circuit_name"`
	CountryName string             `json:"country_name"`
	DataAsOfUTC time.Time          `json:"data_as_of_utc"`
	Sessions    []SessionDetailDTO `json:"sessions"`
}
