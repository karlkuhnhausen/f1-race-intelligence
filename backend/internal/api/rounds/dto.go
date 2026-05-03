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

// NotableEventDTO is the API representation of a single notable race-control event.
type NotableEventDTO struct {
	// EventType is one of: "red_flag", "safety_car", "vsc", "investigation"
	EventType string `json:"event_type"`
	// LapNumber is the lap on which the first activation occurred (0 for pre-race).
	LapNumber int `json:"lap_number"`
	// Count is the number of distinct activations of this event type.
	Count int `json:"count"`
}

// SessionRecapDTO is the pre-computed summary payload for one session.
// Fields present depend on session type. Absent fields are omitted from JSON.
type SessionRecapDTO struct {
	// --- Race / Sprint ---
	WinnerName           string   `json:"winner_name,omitempty"`
	WinnerTeam           string   `json:"winner_team,omitempty"`
	GapToP2              string   `json:"gap_to_p2,omitempty"`
	FastestLapHolder     string   `json:"fastest_lap_holder,omitempty"`
	FastestLapTeam       string   `json:"fastest_lap_team,omitempty"`
	FastestLapTimeSeconds *float64 `json:"fastest_lap_time_seconds,omitempty"`
	TotalLaps            int      `json:"total_laps,omitempty"`

	// --- Qualifying / Sprint Qualifying ---
	PoleSitterName string   `json:"pole_sitter_name,omitempty"`
	PoleSitterTeam string   `json:"pole_sitter_team,omitempty"`
	PoleTime       *float64 `json:"pole_time,omitempty"`
	Q1CutoffTime   *float64 `json:"q1_cutoff_time,omitempty"`
	Q2CutoffTime   *float64 `json:"q2_cutoff_time,omitempty"`

	// --- Practice ---
	BestDriverName string   `json:"best_driver_name,omitempty"`
	BestDriverTeam string   `json:"best_driver_team,omitempty"`
	BestLapTime    *float64 `json:"best_lap_time,omitempty"`

	// --- All sessions with race-control data ---
	RedFlagCount   int              `json:"red_flag_count,omitempty"`
	SafetyCarCount int              `json:"safety_car_count,omitempty"`
	VSCCount       int              `json:"vsc_count,omitempty"`
	TopEvent       *NotableEventDTO `json:"top_event,omitempty"`
}

// SessionDetailDTO is the API response shape for one session within a round.
type SessionDetailDTO struct {
	SessionName  string             `json:"session_name"`
	SessionType  string             `json:"session_type"`
	Status       string             `json:"status"`
	DateStart    time.Time          `json:"date_start_utc"`
	DateEnd      time.Time          `json:"date_end_utc"`
	Results      []SessionResultDTO `json:"results"`
	RecapSummary *SessionRecapDTO   `json:"recap_summary,omitempty"`
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
