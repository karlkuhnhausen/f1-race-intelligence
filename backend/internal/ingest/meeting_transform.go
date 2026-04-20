// Package ingest provides meeting normalization from OpenF1 raw data.
package ingest

import (
	"fmt"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// NormalizeMeetings transforms raw OpenF1 meeting data into storage-ready RaceMeeting structs.
// It applies deterministic round numbering and timestamps.
func NormalizeMeetings(raw []openF1Meeting, season int) []storage.RaceMeeting {
	now := time.Now().UTC()
	meetings := make([]storage.RaceMeeting, 0, len(raw))

	for i, r := range raw {
		round := i + 1
		startUTC, _ := time.Parse(time.RFC3339, r.DateStart)

		m := storage.RaceMeeting{
			ID:               fmt.Sprintf("%d-%02d", season, round),
			Season:           season,
			Round:            round,
			RaceName:         r.MeetingName,
			CircuitName:      r.CircuitName,
			CountryName:      r.CountryName,
			StartDatetimeUTC: startUTC,
			EndDatetimeUTC:   startUTC.Add(3 * 24 * time.Hour),
			Status:           "scheduled",
			IsCancelled:      false,
			Source:           "openf1",
			DataAsOfUTC:      now,
			SourceHash:       fmt.Sprintf("%d", r.MeetingKey),
		}
		meetings = append(meetings, m)
	}

	return meetings
}
