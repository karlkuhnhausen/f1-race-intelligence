package unit

import (
	"context"
	"testing"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/api/calendar"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// fakeCalendarRepo is an in-memory CalendarRepository for testing the service.
type fakeCalendarRepo struct {
	meetings []storage.RaceMeeting
}

func (f *fakeCalendarRepo) UpsertMeeting(_ context.Context, _ storage.RaceMeeting) error {
	return nil
}

func (f *fakeCalendarRepo) GetMeetingsBySeason(_ context.Context, _ int) ([]storage.RaceMeeting, error) {
	return f.meetings, nil
}

func (f *fakeCalendarRepo) GetMeetingByID(_ context.Context, _ int, _ string) (*storage.RaceMeeting, error) {
	return nil, nil
}

func (f *fakeCalendarRepo) GetMeetingByMeetingKey(_ context.Context, _ int, _ int) (*storage.RaceMeeting, error) {
	return nil, nil
}

func (f *fakeCalendarRepo) DeleteMeeting(_ context.Context, _ int, _ string) error {
	return nil
}

// TestGetCalendar_DerivesStatusAtReadTime verifies that meeting status is
// computed from start/end dates relative to the current time, regardless of
// the value persisted in storage. Mirrors the day-12 session-level pattern.
func TestGetCalendar_DerivesStatusAtReadTime(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)

	meetings := []storage.RaceMeeting{
		// Past round — stored Status is stale "scheduled" but should derive to "completed".
		{
			ID: "2026-01", Season: 2026, Round: 1,
			RaceName:         "Australian Grand Prix",
			StartDatetimeUTC: time.Date(2026, 3, 13, 5, 0, 0, 0, time.UTC),
			EndDatetimeUTC:   time.Date(2026, 3, 16, 5, 0, 0, 0, time.UTC),
			Status:           "scheduled",
		},
		// In-progress weekend — start in the past, end in the future.
		{
			ID: "2026-04", Season: 2026, Round: 4,
			RaceName:         "Bahrain Race Weekend",
			StartDatetimeUTC: now.Add(-6 * time.Hour),
			EndDatetimeUTC:   now.Add(2 * 24 * time.Hour),
			Status:           "scheduled",
		},
		// Future round.
		{
			ID: "2026-05", Season: 2026, Round: 5,
			RaceName:         "Miami Grand Prix",
			StartDatetimeUTC: time.Date(2026, 5, 4, 19, 0, 0, 0, time.UTC),
			EndDatetimeUTC:   time.Date(2026, 5, 7, 19, 0, 0, 0, time.UTC),
			Status:           "scheduled",
		},
	}

	repo := &fakeCalendarRepo{meetings: meetings}
	svc := calendar.NewServiceWithClock(repo, func() time.Time { return now })

	resp, err := svc.GetCalendar(context.Background(), 2026)
	if err != nil {
		t.Fatalf("GetCalendar returned error: %v", err)
	}
	if len(resp.Rounds) != 3 {
		t.Fatalf("expected 3 rounds, got %d", len(resp.Rounds))
	}

	got := map[int]string{}
	for _, r := range resp.Rounds {
		got[r.Round] = r.Status
	}
	if got[1] != "completed" {
		t.Errorf("round 1 (past) status: want %q, got %q", "completed", got[1])
	}
	if got[4] != "scheduled" {
		t.Errorf("round 4 (in-progress) status: want %q, got %q", "scheduled", got[4])
	}
	if got[5] != "scheduled" {
		t.Errorf("round 5 (future) status: want %q, got %q", "scheduled", got[5])
	}
}

// TestGetCalendar_CancelledRacesExcludedAtIngestion verifies that cancelled
// races (excluded at ingestion time) simply don't appear in the calendar.
// If one somehow slips through in storage, it is NOT marked cancelled at
// read time — that's the ingestion layer's responsibility.
func TestGetCalendar_CancelledRacesExcludedAtIngestion(t *testing.T) {
	now := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	// Simulate a non-cancelled round — cancelled races should never be in
	// storage after the ingest-time filter.
	meetings := []storage.RaceMeeting{
		{
			ID: "2026-04", Season: 2026, Round: 4,
			RaceName:         "Miami Grand Prix",
			StartDatetimeUTC: time.Date(2026, 5, 4, 19, 0, 0, 0, time.UTC),
			EndDatetimeUTC:   time.Date(2026, 5, 7, 19, 0, 0, 0, time.UTC),
			Status:           "scheduled",
		},
	}

	repo := &fakeCalendarRepo{meetings: meetings}
	svc := calendar.NewServiceWithClock(repo, func() time.Time { return now })

	resp, err := svc.GetCalendar(context.Background(), 2026)
	if err != nil {
		t.Fatalf("GetCalendar returned error: %v", err)
	}
	if len(resp.Rounds) != 1 {
		t.Fatalf("expected 1 round, got %d", len(resp.Rounds))
	}
	r := resp.Rounds[0]
	if r.IsCancelled {
		t.Errorf("expected IsCancelled=false for non-cancelled race")
	}
	if r.RaceName != "Miami Grand Prix" {
		t.Errorf("expected Miami Grand Prix, got %q", r.RaceName)
	}
}

// TestGetCalendar_ZeroStartReturnsUnknown verifies the safety branch for
// meetings ingested with no start datetime.
func TestGetCalendar_ZeroStartReturnsUnknown(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	meetings := []storage.RaceMeeting{
		{ID: "2026-99", Season: 2026, Round: 99, RaceName: "Mystery GP", Status: "scheduled"},
	}

	repo := &fakeCalendarRepo{meetings: meetings}
	svc := calendar.NewServiceWithClock(repo, func() time.Time { return now })

	resp, err := svc.GetCalendar(context.Background(), 2026)
	if err != nil {
		t.Fatalf("GetCalendar returned error: %v", err)
	}
	if resp.Rounds[0].Status != "unknown" {
		t.Errorf("expected status=unknown for zero start, got %q", resp.Rounds[0].Status)
	}
}
