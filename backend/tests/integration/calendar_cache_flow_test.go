package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"log/slog"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/api/calendar"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// inMemoryRepo simulates the poll→cache→API flow without Cosmos.
type inMemoryRepo struct {
	meetings map[string]storage.RaceMeeting
}

func newInMemoryRepo() *inMemoryRepo {
	return &inMemoryRepo{meetings: make(map[string]storage.RaceMeeting)}
}

func (r *inMemoryRepo) UpsertMeeting(_ context.Context, m storage.RaceMeeting) error {
	r.meetings[m.ID] = m
	return nil
}

func (r *inMemoryRepo) GetMeetingsBySeason(_ context.Context, season int) ([]storage.RaceMeeting, error) {
	var result []storage.RaceMeeting
	for _, m := range r.meetings {
		if m.Season == season {
			result = append(result, m)
		}
	}
	return result, nil
}

func (r *inMemoryRepo) GetMeetingByID(_ context.Context, _ int, id string) (*storage.RaceMeeting, error) {
	m, ok := r.meetings[id]
	if !ok {
		return nil, nil
	}
	return &m, nil
}

func TestPollCacheAPIFlow(t *testing.T) {
	repo := newInMemoryRepo()
	now := time.Now().UTC()

	// Simulate polling: upsert 5 meetings.
	for i := 1; i <= 5; i++ {
		m := storage.RaceMeeting{
			ID:               fmt.Sprintf("2026-%02d", i),
			Season:           2026,
			Round:            i,
			RaceName:         fmt.Sprintf("GP %d", i),
			CircuitName:      fmt.Sprintf("Circuit %d", i),
			CountryName:      fmt.Sprintf("Country %d", i),
			StartDatetimeUTC: now.Add(time.Duration(i*7) * 24 * time.Hour),
			EndDatetimeUTC:   now.Add(time.Duration(i*7+3) * 24 * time.Hour),
			Status:           "scheduled",
			Source:           "openf1",
			DataAsOfUTC:      now,
			SourceHash:       fmt.Sprintf("hash-%d", i),
		}
		if err := repo.UpsertMeeting(context.Background(), m); err != nil {
			t.Fatalf("upsert: %v", err)
		}
	}

	// Verify via API.
	svc := calendar.NewService(repo)
	handler := calendar.NewHandler(svc, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar?year=2026", nil)
	rec := httptest.NewRecorder()
	handler.GetCalendar(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp calendar.CalendarResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(resp.Rounds) != 5 {
		t.Errorf("expected 5 rounds, got %d", len(resp.Rounds))
	}

	if resp.NextRound == 0 {
		t.Error("expected next_round to be set")
	}

	if resp.CountdownTargetUTC == nil {
		t.Error("expected countdown_target_utc to be set")
	}
}

func TestUpsertIdempotency(t *testing.T) {
	repo := newInMemoryRepo()
	now := time.Now().UTC()

	m := storage.RaceMeeting{
		ID:               "2026-01",
		Season:           2026,
		Round:            1,
		RaceName:         "Australian GP",
		CircuitName:      "Albert Park",
		CountryName:      "Australia",
		StartDatetimeUTC: now.Add(7 * 24 * time.Hour),
		EndDatetimeUTC:   now.Add(10 * 24 * time.Hour),
		Status:           "scheduled",
		Source:           "openf1",
		DataAsOfUTC:      now,
		SourceHash:       "hash-1",
	}

	// Upsert twice.
	_ = repo.UpsertMeeting(context.Background(), m)
	m.DataAsOfUTC = now.Add(5 * time.Minute)
	_ = repo.UpsertMeeting(context.Background(), m)

	meetings, _ := repo.GetMeetingsBySeason(context.Background(), 2026)
	if len(meetings) != 1 {
		t.Errorf("expected 1 meeting after duplicate upsert, got %d", len(meetings))
	}
}
