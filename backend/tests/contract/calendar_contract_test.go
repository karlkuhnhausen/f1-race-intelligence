package contract

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

// mockCalendarRepo implements storage.CalendarRepository for testing.
type mockCalendarRepo struct {
	meetings []storage.RaceMeeting
}

func (m *mockCalendarRepo) UpsertMeeting(_ context.Context, meeting storage.RaceMeeting) error {
	m.meetings = append(m.meetings, meeting)
	return nil
}

func (m *mockCalendarRepo) GetMeetingsBySeason(_ context.Context, season int) ([]storage.RaceMeeting, error) {
	var result []storage.RaceMeeting
	for _, mtg := range m.meetings {
		if mtg.Season == season {
			result = append(result, mtg)
		}
	}
	return result, nil
}

func (m *mockCalendarRepo) GetMeetingByID(_ context.Context, _ int, id string) (*storage.RaceMeeting, error) {
	for _, mtg := range m.meetings {
		if mtg.ID == id {
			return &mtg, nil
		}
	}
	return nil, nil
}

func seedMeetings() []storage.RaceMeeting {
	now := time.Now().UTC()
	meetings := make([]storage.RaceMeeting, 0, 24)
	baseDate := time.Date(2026, 3, 15, 14, 0, 0, 0, time.UTC)

	for i := 1; i <= 24; i++ {
		raceDate := baseDate.Add(time.Duration(i*14) * 24 * time.Hour)
		m := storage.RaceMeeting{
			ID:               fmt.Sprintf("2026-%02d", i),
			Season:           2026,
			Round:            i,
			RaceName:         fmt.Sprintf("Round %d Grand Prix", i),
			CircuitName:      fmt.Sprintf("Circuit %d", i),
			CountryName:      fmt.Sprintf("Country %d", i),
			StartDatetimeUTC: raceDate,
			EndDatetimeUTC:   raceDate.Add(3 * 24 * time.Hour),
			Status:           "scheduled",
			IsCancelled:      false,
			Source:           "openf1",
			DataAsOfUTC:      now,
			SourceHash:       fmt.Sprintf("hash-%d", i),
		}
		// Mark R4 and R5 as cancelled per spec.
		if i == 4 || i == 5 {
			m.Status = "cancelled"
			m.IsCancelled = true
			m.CancelledLabel = "Cancelled"
		}
		meetings = append(meetings, m)
	}
	return meetings
}

func TestCalendarContractReturns24Rounds(t *testing.T) {
	repo := &mockCalendarRepo{meetings: seedMeetings()}
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

	if resp.Year != 2026 {
		t.Errorf("expected year 2026, got %d", resp.Year)
	}

	if len(resp.Rounds) != 24 {
		t.Errorf("expected 24 rounds, got %d", len(resp.Rounds))
	}
}

func TestCalendarContractRequiredFields(t *testing.T) {
	repo := &mockCalendarRepo{meetings: seedMeetings()}
	svc := calendar.NewService(repo)
	handler := calendar.NewHandler(svc, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar?year=2026", nil)
	rec := httptest.NewRecorder()

	handler.GetCalendar(rec, req)

	var raw map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode: %v", err)
	}

	required := []string{"year", "data_as_of_utc", "next_round", "countdown_target_utc", "rounds"}
	for _, field := range required {
		if _, ok := raw[field]; !ok {
			t.Errorf("missing required field: %s", field)
		}
	}
}

func TestCalendarContractCancelledRounds(t *testing.T) {
	repo := &mockCalendarRepo{meetings: seedMeetings()}
	svc := calendar.NewService(repo)
	handler := calendar.NewHandler(svc, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar?year=2026", nil)
	rec := httptest.NewRecorder()

	handler.GetCalendar(rec, req)

	var resp calendar.CalendarResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	cancelledCount := 0
	for _, r := range resp.Rounds {
		if r.IsCancelled {
			cancelledCount++
			if r.Status != "cancelled" {
				t.Errorf("round %d: is_cancelled=true but status=%q", r.Round, r.Status)
			}
		}
	}

	if cancelledCount != 2 {
		t.Errorf("expected 2 cancelled rounds (R4, R5), got %d", cancelledCount)
	}
}

func TestCalendarContractMissingYear(t *testing.T) {
	repo := &mockCalendarRepo{}
	svc := calendar.NewService(repo)
	handler := calendar.NewHandler(svc, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar", nil)
	rec := httptest.NewRecorder()

	handler.GetCalendar(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing year, got %d", rec.Code)
	}
}
