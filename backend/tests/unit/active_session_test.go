package unit

import (
	"testing"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/domain"
)

func miamiSessions() []domain.SessionWindow {
	// Friday FP1/FP2, Saturday FP3/Qualifying, Sunday Race.
	return []domain.SessionWindow{
		{SessionType: domain.SessionPractice1, SessionName: "Practice 1",
			DateStartUTC: time.Date(2026, 5, 1, 17, 30, 0, 0, time.UTC),
			DateEndUTC:   time.Date(2026, 5, 1, 18, 30, 0, 0, time.UTC)},
		{SessionType: domain.SessionPractice2, SessionName: "Practice 2",
			DateStartUTC: time.Date(2026, 5, 1, 21, 0, 0, 0, time.UTC),
			DateEndUTC:   time.Date(2026, 5, 1, 22, 0, 0, 0, time.UTC)},
		{SessionType: domain.SessionPractice3, SessionName: "Practice 3",
			DateStartUTC: time.Date(2026, 5, 2, 16, 30, 0, 0, time.UTC),
			DateEndUTC:   time.Date(2026, 5, 2, 17, 30, 0, 0, time.UTC)},
		{SessionType: domain.SessionQualifying, SessionName: "Qualifying",
			DateStartUTC: time.Date(2026, 5, 2, 20, 0, 0, 0, time.UTC),
			DateEndUTC:   time.Date(2026, 5, 2, 21, 0, 0, 0, time.UTC)},
		{SessionType: domain.SessionRace, SessionName: "Race",
			DateStartUTC: time.Date(2026, 5, 3, 19, 0, 0, 0, time.UTC),
			DateEndUTC:   time.Date(2026, 5, 3, 21, 0, 0, 0, time.UTC)},
	}
}

func TestSelectActiveSession_PicksInProgress(t *testing.T) {
	now := time.Date(2026, 5, 1, 17, 45, 0, 0, time.UTC) // mid-FP1
	active, ok := domain.SelectActiveSession(miamiSessions(), now)
	if !ok {
		t.Fatal("expected an active session")
	}
	if active.SessionType != domain.SessionPractice1 {
		t.Errorf("expected practice1, got %s", active.SessionType)
	}
	if active.Status != domain.SessionStatusInProgress {
		t.Errorf("expected in_progress, got %s", active.Status)
	}
}

func TestSelectActiveSession_PicksNextUpcoming(t *testing.T) {
	// Saturday morning, between FP3 and Qualifying — FP1/FP2 completed,
	// FP3 not yet started. Should return FP3 as next upcoming.
	now := time.Date(2026, 5, 2, 14, 0, 0, 0, time.UTC)
	active, ok := domain.SelectActiveSession(miamiSessions(), now)
	if !ok {
		t.Fatal("expected an active session")
	}
	if active.SessionType != domain.SessionPractice3 {
		t.Errorf("expected practice3, got %s", active.SessionType)
	}
	if active.Status != domain.SessionStatusUpcoming {
		t.Errorf("expected upcoming, got %s", active.Status)
	}
}

func TestSelectActiveSession_AllCompletedReturnsLast(t *testing.T) {
	// Sunday after the race — every session is completed, return the Race.
	now := time.Date(2026, 5, 3, 22, 0, 0, 0, time.UTC)
	active, ok := domain.SelectActiveSession(miamiSessions(), now)
	if !ok {
		t.Fatal("expected an active session")
	}
	if active.SessionType != domain.SessionRace {
		t.Errorf("expected race, got %s", active.SessionType)
	}
	if active.Status != domain.SessionStatusCompleted {
		t.Errorf("expected completed, got %s", active.Status)
	}
}

func TestSelectActiveSession_BeforeWeekendReturnsFirstUpcoming(t *testing.T) {
	now := time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC) // before any session
	active, ok := domain.SelectActiveSession(miamiSessions(), now)
	if !ok {
		t.Fatal("expected an active session")
	}
	if active.SessionType != domain.SessionPractice1 {
		t.Errorf("expected practice1 as first upcoming, got %s", active.SessionType)
	}
}

func TestSelectActiveSession_EmptySlice(t *testing.T) {
	_, ok := domain.SelectActiveSession(nil, time.Now())
	if ok {
		t.Error("expected ok=false for empty input")
	}
}
