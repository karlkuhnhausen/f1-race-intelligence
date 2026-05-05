package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/api/calendar"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// fixedClock returns a function that always returns the given time.
func fixedClock(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

type inMemoryCalendarRepo struct {
	meetings []storage.RaceMeeting
}

func (r *inMemoryCalendarRepo) UpsertMeeting(_ context.Context, m storage.RaceMeeting) error {
	for i, existing := range r.meetings {
		if existing.ID == m.ID {
			r.meetings[i] = m
			return nil
		}
	}
	r.meetings = append(r.meetings, m)
	return nil
}

func (r *inMemoryCalendarRepo) GetMeetingsBySeason(_ context.Context, season int) ([]storage.RaceMeeting, error) {
	var result []storage.RaceMeeting
	for _, m := range r.meetings {
		if m.Season == season {
			result = append(result, m)
		}
	}
	return result, nil
}

func (r *inMemoryCalendarRepo) GetMeetingByID(_ context.Context, _ int, id string) (*storage.RaceMeeting, error) {
	for _, m := range r.meetings {
		if m.ID == id {
			return &m, nil
		}
	}
	return nil, nil
}

func (r *inMemoryCalendarRepo) GetMeetingByMeetingKey(_ context.Context, _ int, meetingKey int) (*storage.RaceMeeting, error) {
	for _, m := range r.meetings {
		if m.MeetingKey == meetingKey {
			return &m, nil
		}
	}
	return nil, nil
}

func (r *inMemoryCalendarRepo) DeleteMeeting(_ context.Context, _ int, id string) error {
	for i, m := range r.meetings {
		if m.ID == id {
			r.meetings = append(r.meetings[:i], r.meetings[i+1:]...)
			return nil
		}
	}
	return nil
}

func seedThreeRounds() *inMemoryCalendarRepo {
	now := time.Now().UTC()
	repo := &inMemoryCalendarRepo{}
	rounds := []struct {
		round     int
		name      string
		start     time.Time
		cancelled bool
		status    string
	}{
		{1, "Australian GP", time.Date(2026, 3, 15, 5, 0, 0, 0, time.UTC), false, "completed"},
		{2, "Miami GP", time.Date(2026, 5, 4, 19, 0, 0, 0, time.UTC), false, "scheduled"},
		{3, "Spanish GP", time.Date(2026, 5, 25, 13, 0, 0, 0, time.UTC), false, "scheduled"},
	}
	for _, r := range rounds {
		err := repo.UpsertMeeting(context.Background(), storage.RaceMeeting{
			ID:               fmt.Sprintf("2026-%02d", r.round),
			Season:           2026,
			Round:            r.round,
			RaceName:         r.name,
			CircuitName:      fmt.Sprintf("Circuit %d", r.round),
			CountryName:      fmt.Sprintf("Country %d", r.round),
			StartDatetimeUTC: r.start,
			EndDatetimeUTC:   r.start.Add(3 * 24 * time.Hour),
			Status:           r.status,
			IsCancelled:      r.cancelled,
			Source:           "test",
			DataAsOfUTC:      now,
			SourceHash:       fmt.Sprintf("hash-%d", r.round),
		})
		if err != nil {
			panic(fmt.Sprintf("UpsertMeeting failed: %v", err))
		}
	}
	return repo
}

func TestCountdownTransition_BeforeMiami(t *testing.T) {
	repo := seedThreeRounds()

	// "Now" is before the Miami GP — next round should be Miami (R2).
	beforeMiami := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)
	svc := calendar.NewServiceWithClock(repo, fixedClock(beforeMiami))
	handler := calendar.NewHandler(svc, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar?year=2026", nil)
	rec := httptest.NewRecorder()
	handler.GetCalendar(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp calendar.CalendarResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.NextRound != 2 {
		t.Errorf("before Miami: expected next_round=2, got %d", resp.NextRound)
	}
	if resp.CountdownTargetUTC == nil {
		t.Fatal("expected countdown_target_utc to be set")
	}
	expected := time.Date(2026, 5, 4, 19, 0, 0, 0, time.UTC)
	if !resp.CountdownTargetUTC.Equal(expected) {
		t.Errorf("expected countdown target %v, got %v", expected, *resp.CountdownTargetUTC)
	}
}

func TestCountdownTransition_AfterMiami(t *testing.T) {
	repo := seedThreeRounds()

	// "Now" is after the entire Miami weekend (Miami ends May 7 19:00 UTC) —
	// next round should transition to Spanish GP (R3).
	afterMiami := time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)
	svc := calendar.NewServiceWithClock(repo, fixedClock(afterMiami))
	handler := calendar.NewHandler(svc, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar?year=2026", nil)
	rec := httptest.NewRecorder()
	handler.GetCalendar(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp calendar.CalendarResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.NextRound != 3 {
		t.Errorf("after Miami: expected next_round=3, got %d", resp.NextRound)
	}
	if resp.CountdownTargetUTC == nil {
		t.Fatal("expected countdown_target_utc to be set")
	}
	expected := time.Date(2026, 5, 25, 13, 0, 0, 0, time.UTC)
	if !resp.CountdownTargetUTC.Equal(expected) {
		t.Errorf("expected countdown target %v, got %v", expected, *resp.CountdownTargetUTC)
	}
}

func TestCountdownTransition_SeasonOver(t *testing.T) {
	repo := seedThreeRounds()

	// "Now" is after all rounds — no next race.
	seasonEnd := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	svc := calendar.NewServiceWithClock(repo, fixedClock(seasonEnd))
	handler := calendar.NewHandler(svc, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar?year=2026", nil)
	rec := httptest.NewRecorder()
	handler.GetCalendar(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp calendar.CalendarResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.NextRound != 0 {
		t.Errorf("season over: expected next_round=0, got %d", resp.NextRound)
	}
	if resp.CountdownTargetUTC != nil {
		t.Errorf("season over: expected countdown_target_utc to be nil, got %v", *resp.CountdownTargetUTC)
	}
}
