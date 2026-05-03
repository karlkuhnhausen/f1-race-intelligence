package contract

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/api/rounds"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// mockSessionRepo implements storage.SessionRepository for testing.
type mockSessionRepo struct {
	sessions []storage.Session
	results  []storage.SessionResult
}

func (m *mockSessionRepo) UpsertSession(_ context.Context, s storage.Session) error {
	m.sessions = append(m.sessions, s)
	return nil
}

func (m *mockSessionRepo) UpsertSessionResult(_ context.Context, r storage.SessionResult) error {
	m.results = append(m.results, r)
	return nil
}

func (m *mockSessionRepo) GetSessionsByRound(_ context.Context, season, round int) ([]storage.Session, error) {
	var out []storage.Session
	for _, s := range m.sessions {
		if s.Season == season && s.Round == round {
			out = append(out, s)
		}
	}
	return out, nil
}

func (m *mockSessionRepo) GetSessionResultsByRound(_ context.Context, season, round int) ([]storage.SessionResult, error) {
	var out []storage.SessionResult
	for _, r := range m.results {
		if r.Season == season && r.Round == round {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *mockSessionRepo) GetSessionResultsBySeason(_ context.Context, season int) ([]storage.SessionResult, error) {
	var out []storage.SessionResult
	for _, r := range m.results {
		if r.Season == season {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *mockSessionRepo) GetFinalizedSessionKeys(_ context.Context, season int) (map[int]int, error) {
	out := make(map[int]int)
	for _, s := range m.sessions {
		if s.Season == season && s.Finalized {
			out[s.SessionKey] = s.SchemaVersion
		}
	}
	return out, nil
}

func seedSessions() (*mockSessionRepo, *mockCalendarRepo) {
	now := time.Now().UTC()

	calRepo := &mockCalendarRepo{
		meetings: []storage.RaceMeeting{
			{
				ID: "2026-01", Season: 2026, Round: 1,
				RaceName: "Australian Grand Prix", CircuitName: "Albert Park",
				CountryName:      "Australia",
				StartDatetimeUTC: time.Date(2026, 3, 15, 5, 0, 0, 0, time.UTC),
				EndDatetimeUTC:   time.Date(2026, 3, 18, 5, 0, 0, 0, time.UTC),
				Status:           "scheduled", Source: "openf1", DataAsOfUTC: now,
			},
		},
	}

	sessRepo := &mockSessionRepo{
		sessions: []storage.Session{
			{
				ID: "2026-01-race", Type: "session", Season: 2026, Round: 1,
				SessionName: "Race", SessionType: "race", Status: "completed",
				DateStartUTC: time.Date(2026, 3, 15, 5, 0, 0, 0, time.UTC),
				DateEndUTC:   time.Date(2026, 3, 15, 7, 0, 0, 0, time.UTC),
				DataAsOfUTC:  now, Source: "openf1",
			},
			{
				ID: "2026-01-qualifying", Type: "session", Season: 2026, Round: 1,
				SessionName: "Qualifying", SessionType: "qualifying", Status: "completed",
				DateStartUTC: time.Date(2026, 3, 14, 6, 0, 0, 0, time.UTC),
				DateEndUTC:   time.Date(2026, 3, 14, 7, 0, 0, 0, time.UTC),
				DataAsOfUTC:  now, Source: "openf1",
			},
		},
		results: []storage.SessionResult{
			{
				ID: "2026-01-race-1", Type: "session_result", Season: 2026, Round: 1,
				SessionType: "race", Position: 1, DriverNumber: 1,
				DriverName: "Max VERSTAPPEN", DriverAcronym: "VER",
				TeamName: "Red Bull Racing", NumberOfLaps: 58,
				DataAsOfUTC: now, Source: "openf1",
			},
			{
				ID: "2026-01-race-44", Type: "session_result", Season: 2026, Round: 1,
				SessionType: "race", Position: 2, DriverNumber: 44,
				DriverName: "Lewis HAMILTON", DriverAcronym: "HAM",
				TeamName: "Ferrari", NumberOfLaps: 58,
				DataAsOfUTC: now, Source: "openf1",
			},
		},
	}

	return sessRepo, calRepo
}

func TestRoundDetailContract(t *testing.T) {
	sessRepo, calRepo := seedSessions()
	svc := rounds.NewService(sessRepo, calRepo)
	handler := rounds.NewHandler(svc, slog.Default())

	t.Run("returns round detail with sessions and results", func(t *testing.T) {
		// Use chi router to parse URL params
		r := chi.NewRouter()
		r.Get("/api/v1/rounds/{round}", handler.GetRoundDetail)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/rounds/1?year=2026", nil)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp rounds.RoundDetailResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}

		if resp.Year != 2026 {
			t.Errorf("year = %d, want 2026", resp.Year)
		}
		if resp.Round != 1 {
			t.Errorf("round = %d, want 1", resp.Round)
		}
		if resp.RaceName != "Australian Grand Prix" {
			t.Errorf("race_name = %q, want Australian Grand Prix", resp.RaceName)
		}
		if len(resp.Sessions) != 2 {
			t.Fatalf("sessions count = %d, want 2", len(resp.Sessions))
		}

		// Find the race session
		var raceSess *rounds.SessionDetailDTO
		for i := range resp.Sessions {
			if resp.Sessions[i].SessionType == "race" {
				raceSess = &resp.Sessions[i]
				break
			}
		}
		if raceSess == nil {
			t.Fatal("no race session found")
		}
		if len(raceSess.Results) != 2 {
			t.Fatalf("race results = %d, want 2", len(raceSess.Results))
		}
		if raceSess.Results[0].DriverAcronym != "VER" {
			t.Errorf("P1 driver = %q, want VER", raceSess.Results[0].DriverAcronym)
		}
		if raceSess.Results[1].DriverAcronym != "HAM" {
			t.Errorf("P2 driver = %q, want HAM", raceSess.Results[1].DriverAcronym)
		}
	})

	t.Run("returns 400 for invalid round", func(t *testing.T) {
		r := chi.NewRouter()
		r.Get("/api/v1/rounds/{round}", handler.GetRoundDetail)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/rounds/abc", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("returns empty sessions for round with no data", func(t *testing.T) {
		r := chi.NewRouter()
		r.Get("/api/v1/rounds/{round}", handler.GetRoundDetail)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/rounds/20?year=2026", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var resp rounds.RoundDetailResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(resp.Sessions) != 0 {
			t.Errorf("sessions = %d, want 0", len(resp.Sessions))
		}
	})
}
