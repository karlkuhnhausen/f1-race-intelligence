package domain

import "time"

// NextRaceResult holds the computed next-race metadata.
type NextRaceResult struct {
	Round           int
	CountdownTarget time.Time
	Found           bool
}

// SelectNextRace returns the first future, non-cancelled round from an ordered
// slice of meetings. Tie-break rule: if two rounds share the same start time,
// the lower round number wins (preserving calendar order).
// The now parameter is injected for deterministic testing.
func SelectNextRace(meetings []RaceMeeting, now time.Time) NextRaceResult {
	for _, m := range meetings {
		if m.IsCancelled {
			continue
		}
		if m.Status == StatusCancelled {
			continue
		}
		if !m.StartDatetimeUTC.After(now) {
			continue
		}
		return NextRaceResult{
			Round:           m.Round,
			CountdownTarget: m.StartDatetimeUTC,
			Found:           true,
		}
	}
	return NextRaceResult{}
}
