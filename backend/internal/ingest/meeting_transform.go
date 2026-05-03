// Package ingest provides meeting normalization from OpenF1 raw data.
package ingest

import (
	"fmt"
	"strings"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// IsPreSeasonTesting reports whether a meeting/race name refers to pre-season
// testing rather than a championship race. Testing meetings should not be
// counted as rounds — Round 1 is the first race of the season.
func IsPreSeasonTesting(raceName string) bool {
	n := strings.ToLower(raceName)
	return strings.Contains(n, "pre-season") ||
		strings.Contains(n, "pre season") ||
		strings.Contains(n, "preseason") ||
		strings.Contains(n, "testing")
}

// NormalizeMeetings transforms raw OpenF1 meeting data into storage-ready RaceMeeting structs.
// It applies deterministic round numbering and timestamps. Pre-season testing
// meetings are filtered out so the first race of the season is Round 1.
func NormalizeMeetings(raw []openF1Meeting, season int) []storage.RaceMeeting {
	now := time.Now().UTC()
	meetings := make([]storage.RaceMeeting, 0, len(raw))

	round := 0
	for _, r := range raw {
		if IsPreSeasonTesting(r.MeetingName) {
			continue
		}
		round++
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
