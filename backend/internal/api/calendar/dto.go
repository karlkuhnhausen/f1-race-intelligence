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
}

// CalendarResponse is the full API response for GET /api/v1/calendar.
type CalendarResponse struct {
	Year               int        `json:"year"`
	DataAsOfUTC        time.Time  `json:"data_as_of_utc"`
	NextRound          int        `json:"next_round"`
	CountdownTargetUTC *time.Time `json:"countdown_target_utc"`
	Rounds             []RoundDTO `json:"rounds"`
}
