// Package domain contains core business types and rules.
package domain

import "time"

// MeetingStatus represents the lifecycle state of a race meeting.
type MeetingStatus string

const (
	StatusScheduled MeetingStatus = "scheduled"
	StatusCancelled MeetingStatus = "cancelled"
	StatusCompleted MeetingStatus = "completed"
	StatusUnknown   MeetingStatus = "unknown"
)

// RaceMeeting is the domain representation of one F1 season round.
type RaceMeeting struct {
	Round            int
	RaceName         string
	CircuitName      string
	CountryName      string
	StartDatetimeUTC time.Time
	EndDatetimeUTC   time.Time
	Status           MeetingStatus
	IsCancelled      bool
	CancelledLabel   string
	CancelledReason  string
}

// IsValid returns true if the status value is one of the known enum values.
func (s MeetingStatus) IsValid() bool {
	switch s {
	case StatusScheduled, StatusCancelled, StatusCompleted, StatusUnknown:
		return true
	}
	return false
}
