package domain

import (
	"sort"
	"time"
)

// SessionWindow is the minimal session description needed for weekend-progress
// computations. It is decoupled from the storage layer so the domain package
// stays free of storage imports.
type SessionWindow struct {
	SessionType  SessionType
	SessionName  string
	DateStartUTC time.Time
	DateEndUTC   time.Time
}

// ActiveSession holds the session that should be highlighted as the
// "current focus" of an in-progress race weekend.
type ActiveSession struct {
	SessionType SessionType
	SessionName string
	Status      SessionStatus
	DateStart   time.Time
	DateEnd     time.Time
}

// DeriveSessionStatus returns the lifecycle status for a single session given
// the wall clock and its scheduled start/end times.
//
// Rules:
//   - Zero start                -> upcoming (safe default)
//   - Start in the future       -> upcoming
//   - Start <= now < end (or end zero) -> in_progress
//   - End <= now                -> completed
func DeriveSessionStatus(now, dateStart, dateEnd time.Time) SessionStatus {
	if dateStart.IsZero() {
		return SessionStatusUpcoming
	}
	if dateStart.After(now) {
		return SessionStatusUpcoming
	}
	if dateEnd.IsZero() || dateEnd.After(now) {
		return SessionStatusInProgress
	}
	return SessionStatusCompleted
}

// SelectActiveSession picks the session that best represents the "current
// focus" of an in-progress weekend. Selection priority:
//  1. The first session currently in_progress (ordered by start time).
//  2. The next upcoming session by start time.
//  3. The most recent completed session (the weekend is wrapping up).
//
// Returns ok=false if the input slice is empty.
func SelectActiveSession(sessions []SessionWindow, now time.Time) (ActiveSession, bool) {
	if len(sessions) == 0 {
		return ActiveSession{}, false
	}

	ordered := make([]SessionWindow, len(sessions))
	copy(ordered, sessions)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].DateStartUTC.Before(ordered[j].DateStartUTC)
	})

	var lastCompleted *SessionWindow
	for i := range ordered {
		s := ordered[i]
		status := DeriveSessionStatus(now, s.DateStartUTC, s.DateEndUTC)
		switch status {
		case SessionStatusInProgress:
			return ActiveSession{
				SessionType: s.SessionType,
				SessionName: s.SessionName,
				Status:      SessionStatusInProgress,
				DateStart:   s.DateStartUTC,
				DateEnd:     s.DateEndUTC,
			}, true
		case SessionStatusUpcoming:
			return ActiveSession{
				SessionType: s.SessionType,
				SessionName: s.SessionName,
				Status:      SessionStatusUpcoming,
				DateStart:   s.DateStartUTC,
				DateEnd:     s.DateEndUTC,
			}, true
		case SessionStatusCompleted:
			cur := ordered[i]
			lastCompleted = &cur
		}
	}

	if lastCompleted != nil {
		return ActiveSession{
			SessionType: lastCompleted.SessionType,
			SessionName: lastCompleted.SessionName,
			Status:      SessionStatusCompleted,
			DateStart:   lastCompleted.DateStartUTC,
			DateEnd:     lastCompleted.DateEndUTC,
		}, true
	}

	return ActiveSession{}, false
}
