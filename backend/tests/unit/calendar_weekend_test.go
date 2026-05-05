package unit

import (
	"context"
	"testing"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/api/calendar"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// fakeSessionRepo is an in-memory SessionRepository for testing.
type fakeSessionRepo struct {
	sessions []storage.Session
}

func (f *fakeSessionRepo) UpsertSession(_ context.Context, _ storage.Session) error {
	return nil
}
func (f *fakeSessionRepo) UpsertSessionResult(_ context.Context, _ storage.SessionResult) error {
	return nil
}
func (f *fakeSessionRepo) GetSessionsByRound(_ context.Context, _, _ int) ([]storage.Session, error) {
	return f.sessions, nil
}
func (f *fakeSessionRepo) GetSessionResultsByRound(_ context.Context, _, _ int) ([]storage.SessionResult, error) {
	return nil, nil
}
func (f *fakeSessionRepo) GetSessionResultsBySeason(_ context.Context, _ int) ([]storage.SessionResult, error) {
	return nil, nil
}
func (f *fakeSessionRepo) GetFinalizedSessionKeys(_ context.Context, _ int) (map[int]int, error) {
	return nil, nil
}
func (f *fakeSessionRepo) GetCompletedRaceSessionKeys(_ context.Context, _ int, _ time.Time) (map[int]struct{}, error) {
	return nil, nil
}
func (f *fakeSessionRepo) DeleteSession(_ context.Context, _ int, _ string) error {
	return nil
}
func (f *fakeSessionRepo) DeleteSessionResultsBySessionType(_ context.Context, _, _ int, _ string) error {
	return nil
}
func (f *fakeSessionRepo) GetSessionsByMeetingKey(_ context.Context, _, _ int) ([]storage.Session, error) {
	return nil, nil
}
func (f *fakeSessionRepo) GetSessionResultsByMeetingKey(_ context.Context, _, _ int) ([]storage.SessionResult, error) {
	return nil, nil
}
func (f *fakeSessionRepo) GetFinalizedSessions(_ context.Context, _ int) ([]storage.Session, error) {
	return nil, nil
}

// TestGetCalendar_WeekendInProgress_ReturnsActiveSession verifies that during
// a race weekend, the calendar response surfaces the in-progress session and
// retargets the countdown rather than skipping ahead to the next race.
func TestGetCalendar_WeekendInProgress_ReturnsActiveSession(t *testing.T) {
	miamiStart := time.Date(2026, 5, 1, 19, 0, 0, 0, time.UTC) // Friday
	miamiEnd := miamiStart.Add(72 * time.Hour)
	now := time.Date(2026, 5, 2, 20, 30, 0, 0, time.UTC) // mid-Qualifying (see below)

	meetings := []storage.RaceMeeting{
		{
			ID: "2026-05", Season: 2026, Round: 5,
			RaceName:         "Miami Grand Prix",
			StartDatetimeUTC: miamiStart,
			EndDatetimeUTC:   miamiEnd,
			Status:           "scheduled",
		},
		{
			ID: "2026-06", Season: 2026, Round: 6,
			RaceName:         "Spanish Grand Prix",
			StartDatetimeUTC: time.Date(2026, 5, 22, 13, 0, 0, 0, time.UTC),
			EndDatetimeUTC:   time.Date(2026, 5, 25, 13, 0, 0, 0, time.UTC),
			Status:           "scheduled",
		},
	}

	sessions := []storage.Session{
		{Season: 2026, Round: 5, SessionType: "practice1", SessionName: "Practice 1",
			DateStartUTC: time.Date(2026, 5, 1, 17, 30, 0, 0, time.UTC),
			DateEndUTC:   time.Date(2026, 5, 1, 18, 30, 0, 0, time.UTC)},
		{Season: 2026, Round: 5, SessionType: "qualifying", SessionName: "Qualifying",
			DateStartUTC: time.Date(2026, 5, 2, 20, 0, 0, 0, time.UTC),
			DateEndUTC:   time.Date(2026, 5, 2, 21, 0, 0, 0, time.UTC)},
		{Season: 2026, Round: 5, SessionType: "race", SessionName: "Race",
			DateStartUTC: time.Date(2026, 5, 3, 19, 0, 0, 0, time.UTC),
			DateEndUTC:   time.Date(2026, 5, 3, 21, 0, 0, 0, time.UTC)},
	}

	calRepo := &fakeCalendarRepo{meetings: meetings}
	sessRepo := &fakeSessionRepo{sessions: sessions}
	svc := calendar.NewServiceWithSessionsAndClock(calRepo, sessRepo, func() time.Time { return now })

	resp, err := svc.GetCalendar(context.Background(), 2026)
	if err != nil {
		t.Fatalf("GetCalendar returned error: %v", err)
	}
	if resp.NextRound != 5 {
		t.Errorf("expected next_round=5 (current weekend), got %d", resp.NextRound)
	}
	if !resp.WeekendInProgress {
		t.Errorf("expected weekend_in_progress=true")
	}
	if resp.ActiveSession == nil {
		t.Fatal("expected active_session to be populated")
	}
	if resp.ActiveSession.SessionType != "qualifying" {
		t.Errorf("expected active session=qualifying, got %s", resp.ActiveSession.SessionType)
	}
	if resp.ActiveSession.Status != "in_progress" {
		t.Errorf("expected active session status=in_progress, got %s", resp.ActiveSession.Status)
	}
	// Countdown target should retarget to the active session's end (live).
	if resp.CountdownTargetUTC == nil {
		t.Fatal("expected countdown_target_utc to be set")
	}
	if !resp.CountdownTargetUTC.Equal(sessions[1].DateEndUTC) {
		t.Errorf("expected countdown target to be qualifying end %v, got %v",
			sessions[1].DateEndUTC, *resp.CountdownTargetUTC)
	}
}

// TestGetCalendar_BeforeWeekend_NoEnrichment verifies normal countdown behavior
// when no race weekend is currently in progress.
func TestGetCalendar_BeforeWeekend_NoEnrichment(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	meetings := []storage.RaceMeeting{
		{
			ID: "2026-05", Season: 2026, Round: 5,
			RaceName:         "Miami Grand Prix",
			StartDatetimeUTC: time.Date(2026, 5, 1, 19, 0, 0, 0, time.UTC),
			EndDatetimeUTC:   time.Date(2026, 5, 4, 19, 0, 0, 0, time.UTC),
			Status:           "scheduled",
		},
	}

	calRepo := &fakeCalendarRepo{meetings: meetings}
	sessRepo := &fakeSessionRepo{}
	svc := calendar.NewServiceWithSessionsAndClock(calRepo, sessRepo, func() time.Time { return now })

	resp, err := svc.GetCalendar(context.Background(), 2026)
	if err != nil {
		t.Fatalf("GetCalendar returned error: %v", err)
	}
	if resp.WeekendInProgress {
		t.Error("expected weekend_in_progress=false before the weekend")
	}
	if resp.ActiveSession != nil {
		t.Errorf("expected active_session to be nil, got %+v", resp.ActiveSession)
	}
	if resp.CountdownTargetUTC == nil || !resp.CountdownTargetUTC.Equal(meetings[0].StartDatetimeUTC) {
		t.Errorf("expected countdown target = race start; got %v", resp.CountdownTargetUTC)
	}
}
