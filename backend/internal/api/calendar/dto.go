// Package calendar implements the calendar API service and response shaping.
package calendar

import "time"

// RoundDTO is the API response shape for a single race round.
type RoundDTO struct {
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
	// Podium lists the top 3 finishers for completed races (P1/P2/P3),
	// joined with current-season championship points. Empty/nil for
	// upcoming or in-progress races.
	Podium []PodiumEntryDTO `json:"podium,omitempty"`
}

// PodiumEntryDTO describes a single podium finisher for a completed race.
// SeasonPoints reflects the driver's CURRENT championship total — the same
// number is used across all completed-race rows for a given driver.
type PodiumEntryDTO struct {
	Position      int     `json:"position"`
	DriverNumber  int     `json:"driver_number"`
	DriverAcronym string  `json:"driver_acronym"`
	DriverName    string  `json:"driver_name"`
	TeamName      string  `json:"team_name"`
	SeasonPoints  float64 `json:"season_points"`
}

// ActiveSessionDTO describes the session currently in focus during an
// in-progress race weekend (live or imminently next).
type ActiveSessionDTO struct {
	SessionType string    `json:"session_type"`
	SessionName string    `json:"session_name"`
	Status      string    `json:"status"` // upcoming | in_progress | completed
	DateStart   time.Time `json:"date_start_utc"`
	DateEnd     time.Time `json:"date_end_utc"`
}

// CalendarResponse is the full API response for GET /api/v1/calendar.
type CalendarResponse struct {
	Year               int               `json:"year"`
	DataAsOfUTC        time.Time         `json:"data_as_of_utc"`
	NextRound          int               `json:"next_round"`
	CountdownTargetUTC *time.Time        `json:"countdown_target_utc"`
	WeekendInProgress  bool              `json:"weekend_in_progress"`
	ActiveSession      *ActiveSessionDTO `json:"active_session,omitempty"`
	Rounds             []RoundDTO        `json:"rounds"`
}
