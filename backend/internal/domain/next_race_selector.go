package domain

import "time"

// NextRaceResult holds the computed next-race metadata.
type NextRaceResult struct {
	Round           int
	CountdownTarget time.Time
	Found           bool
}

// SelectNextRace returns the next non-cancelled round from an ordered slice of
// meetings. A round is considered "next" if its weekend has not yet ended —
// meaning the round currently in progress (race weekend Friday–Sunday) is the
// next round, not the following one. Tie-break rule: if two rounds share the
// same start time, the lower round number wins (preserving calendar order).
// The now parameter is injected for deterministic testing.
//
// Weekend end is computed as EndDatetimeUTC if set, otherwise StartDatetimeUTC
// + 72h as a safe fallback (race weekends span Friday FP1 through Sunday race).
func SelectNextRace(meetings []RaceMeeting, now time.Time) NextRaceResult {
	for _, m := range meetings {
		if m.IsCancelled {
			continue
		}
		if m.Status == StatusCancelled {
			continue
		}
		end := m.EndDatetimeUTC
		if end.IsZero() && !m.StartDatetimeUTC.IsZero() {
			end = m.StartDatetimeUTC.Add(72 * time.Hour)
		}
		if !end.IsZero() && !end.After(now) {
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
