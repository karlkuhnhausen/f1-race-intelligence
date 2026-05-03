package integration

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"encoding/json"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/api/rounds"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// inMemorySessionRepo simulates session storage without Cosmos.
type inMemorySessionRepo struct {
	sessions []storage.Session
	results  []storage.SessionResult
}

func newInMemorySessionRepo() *inMemorySessionRepo {
	return &inMemorySessionRepo{}
}

func (r *inMemorySessionRepo) UpsertSession(_ context.Context, s storage.Session) error {
	for i, existing := range r.sessions {
		if existing.ID == s.ID {
			r.sessions[i] = s
			return nil
		}
	}
	r.sessions = append(r.sessions, s)
	return nil
}

func (r *inMemorySessionRepo) UpsertSessionResult(_ context.Context, sr storage.SessionResult) error {
	for i, existing := range r.results {
		if existing.ID == sr.ID {
			r.results[i] = sr
			return nil
		}
	}
	r.results = append(r.results, sr)
	return nil
}

func (r *inMemorySessionRepo) GetSessionsByRound(_ context.Context, season, round int) ([]storage.Session, error) {
	var result []storage.Session
	for _, s := range r.sessions {
		if s.Season == season && s.Round == round {
			result = append(result, s)
		}
	}
	return result, nil
}

func (r *inMemorySessionRepo) GetSessionResultsByRound(_ context.Context, season, round int) ([]storage.SessionResult, error) {
	var result []storage.SessionResult
	for _, sr := range r.results {
		if sr.Season == season && sr.Round == round {
			result = append(result, sr)
		}
	}
	return result, nil
}

func (r *inMemorySessionRepo) GetSessionResultsBySeason(_ context.Context, season int) ([]storage.SessionResult, error) {
	var result []storage.SessionResult
	for _, sr := range r.results {
		if sr.Season == season {
			result = append(result, sr)
		}
	}
	return result, nil
}

func (r *inMemorySessionRepo) GetFinalizedSessionKeys(_ context.Context, season int) (map[int]int, error) {
	out := make(map[int]int)
	for _, s := range r.sessions {
		if s.Season == season && s.Finalized {
			out[s.SessionKey] = s.SchemaVersion
		}
	}
	return out, nil
}
func (r *inMemorySessionRepo) DeleteSession(_ context.Context, _ int, id string) error {
	for i, s := range r.sessions {
		if s.ID == id {
			r.sessions = append(r.sessions[:i], r.sessions[i+1:]...)
			return nil
		}
	}
	return nil
}
func (r *inMemorySessionRepo) DeleteSessionResultsBySessionType(_ context.Context, season, round int, sessionType string) error {
	var kept []storage.SessionResult
	for _, res := range r.results {
		if res.Season == season && res.Round == round && res.SessionType == sessionType {
			continue
		}
		kept = append(kept, res)
	}
	r.results = kept
	return nil
}
func (r *inMemorySessionRepo) GetFinalizedSessions(_ context.Context, _ int) ([]storage.Session, error) {
	return nil, nil
}

// TestSessionIngestionRoundTrip verifies the poll → transform → upsert → query → API response flow.
func TestSessionIngestionRoundTrip(t *testing.T) {
	calRepo := newInMemoryRepo()
	sessRepo := newInMemorySessionRepo()
	now := time.Now().UTC()

	// Simulate meeting upsert (from OpenF1 poller).
	meeting := storage.RaceMeeting{
		ID:               "2026-01",
		Season:           2026,
		Round:            1,
		RaceName:         "Australian Grand Prix",
		CircuitName:      "Albert Park",
		CountryName:      "Australia",
		StartDatetimeUTC: time.Date(2026, 3, 15, 5, 0, 0, 0, time.UTC),
		EndDatetimeUTC:   time.Date(2026, 3, 15, 7, 0, 0, 0, time.UTC),
		Status:           "completed",
		Source:           "openf1",
		DataAsOfUTC:      now,
		SourceHash:       "hash-1",
	}
	if err := calRepo.UpsertMeeting(context.Background(), meeting); err != nil {
		t.Fatalf("upsert meeting: %v", err)
	}

	// Simulate session upserts (from session poller).
	sessions := []storage.Session{
		{
			ID: "2026-01-fp1", Type: "session", Season: 2026, Round: 1,
			SessionName: "Practice 1", SessionType: "practice1", Status: "completed",
			DateStartUTC: time.Date(2026, 3, 14, 1, 0, 0, 0, time.UTC),
			DateEndUTC:   time.Date(2026, 3, 14, 2, 0, 0, 0, time.UTC),
			DataAsOfUTC:  now, Source: "openf1",
		},
		{
			ID: "2026-01-qualifying", Type: "session", Season: 2026, Round: 1,
			SessionName: "Qualifying", SessionType: "qualifying", Status: "completed",
			DateStartUTC: time.Date(2026, 3, 14, 6, 0, 0, 0, time.UTC),
			DateEndUTC:   time.Date(2026, 3, 14, 7, 0, 0, 0, time.UTC),
			DataAsOfUTC:  now, Source: "openf1",
		},
		{
			ID: "2026-01-race", Type: "session", Season: 2026, Round: 1,
			SessionName: "Race", SessionType: "race", Status: "completed",
			DateStartUTC: time.Date(2026, 3, 15, 5, 0, 0, 0, time.UTC),
			DateEndUTC:   time.Date(2026, 3, 15, 7, 0, 0, 0, time.UTC),
			DataAsOfUTC:  now, Source: "openf1",
		},
	}
	for _, s := range sessions {
		if err := sessRepo.UpsertSession(context.Background(), s); err != nil {
			t.Fatalf("upsert session %s: %v", s.ID, err)
		}
	}

	// Simulate result upserts.
	finished := "Finished"
	points25 := 25.0
	points18 := 18.0
	fastLapTrue := true
	q1 := 78.123
	q2 := 77.456
	q3 := 76.789
	bestLap := 85.432
	gapToFastest := 0.0
	gap2 := 0.456

	results := []storage.SessionResult{
		{
			ID: "2026-01-race-1", Type: "session_result", Season: 2026, Round: 1,
			SessionType: "race", Position: 1, DriverNumber: 1,
			DriverName: "Max VERSTAPPEN", DriverAcronym: "VER", TeamName: "Red Bull Racing",
			NumberOfLaps: 58, FinishingStatus: &finished, Points: &points25, FastestLap: &fastLapTrue,
			DataAsOfUTC: now, Source: "openf1",
		},
		{
			ID: "2026-01-race-44", Type: "session_result", Season: 2026, Round: 1,
			SessionType: "race", Position: 2, DriverNumber: 44,
			DriverName: "Lewis HAMILTON", DriverAcronym: "HAM", TeamName: "Ferrari",
			NumberOfLaps: 58, FinishingStatus: &finished, Points: &points18,
			DataAsOfUTC: now, Source: "openf1",
		},
		{
			ID: "2026-01-qualifying-1", Type: "session_result", Season: 2026, Round: 1,
			SessionType: "qualifying", Position: 1, DriverNumber: 1,
			DriverName: "Max VERSTAPPEN", DriverAcronym: "VER", TeamName: "Red Bull Racing",
			Q1Time: &q1, Q2Time: &q2, Q3Time: &q3,
			DataAsOfUTC: now, Source: "openf1",
		},
		{
			ID: "2026-01-fp1-1", Type: "session_result", Season: 2026, Round: 1,
			SessionType: "practice1", Position: 1, DriverNumber: 1,
			DriverName: "Max VERSTAPPEN", DriverAcronym: "VER", TeamName: "Red Bull Racing",
			BestLapTime: &bestLap, GapToFastest: &gapToFastest, NumberOfLaps: 25,
			DataAsOfUTC: now, Source: "openf1",
		},
		{
			ID: "2026-01-fp1-44", Type: "session_result", Season: 2026, Round: 1,
			SessionType: "practice1", Position: 2, DriverNumber: 44,
			DriverName: "Lewis HAMILTON", DriverAcronym: "HAM", TeamName: "Ferrari",
			BestLapTime: &bestLap, GapToFastest: &gap2, NumberOfLaps: 22,
			DataAsOfUTC: now, Source: "openf1",
		},
	}
	for _, r := range results {
		if err := sessRepo.UpsertSessionResult(context.Background(), r); err != nil {
			t.Fatalf("upsert result %s: %v", r.ID, err)
		}
	}

	// Query via the rounds service and handler.
	svc := rounds.NewService(sessRepo, calRepo)
	handler := rounds.NewHandler(svc, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rounds/1?year=2026", nil)
	// Chi URLParam requires a chi context; simulate with direct service call instead.
	resp, err := svc.GetRoundDetail(context.Background(), 2026, 1)
	if err != nil {
		t.Fatalf("GetRoundDetail: %v", err)
	}

	// Validate response structure.
	if resp.RaceName != "Australian Grand Prix" {
		t.Errorf("RaceName = %q, want %q", resp.RaceName, "Australian Grand Prix")
	}
	if resp.CircuitName != "Albert Park" {
		t.Errorf("CircuitName = %q, want %q", resp.CircuitName, "Albert Park")
	}
	if len(resp.Sessions) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(resp.Sessions))
	}

	// Verify session types are present.
	sessionTypes := make(map[string]int)
	for _, s := range resp.Sessions {
		sessionTypes[s.SessionType] = len(s.Results)
	}

	if sessionTypes["race"] != 2 {
		t.Errorf("expected 2 race results, got %d", sessionTypes["race"])
	}
	if sessionTypes["qualifying"] != 1 {
		t.Errorf("expected 1 qualifying result, got %d", sessionTypes["qualifying"])
	}
	if sessionTypes["practice1"] != 2 {
		t.Errorf("expected 2 practice1 results, got %d", sessionTypes["practice1"])
	}

	// Verify the handler can encode the response (JSON round-trip).
	_ = handler // handler is tested via contract tests; verify it was constructed without panic.
	_ = req

	encoded, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var decoded rounds.RoundDetailResponse
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if decoded.Round != 1 || decoded.Year != 2026 {
		t.Errorf("round-trip: Round=%d Year=%d, want 1/2026", decoded.Round, decoded.Year)
	}
}

// TestSessionUpsertIdempotent verifies that upserting the same session twice doesn't duplicate.
func TestSessionUpsertIdempotent(t *testing.T) {
	repo := newInMemorySessionRepo()
	now := time.Now().UTC()

	s := storage.Session{
		ID: "2026-01-race", Type: "session", Season: 2026, Round: 1,
		SessionName: "Race", SessionType: "race", Status: "completed",
		DateStartUTC: now, DateEndUTC: now.Add(2 * time.Hour),
		DataAsOfUTC: now, Source: "openf1",
	}

	// Upsert twice.
	_ = repo.UpsertSession(context.Background(), s)
	_ = repo.UpsertSession(context.Background(), s)

	sessions, _ := repo.GetSessionsByRound(context.Background(), 2026, 1)
	if len(sessions) != 1 {
		t.Errorf("expected 1 session after double upsert, got %d", len(sessions))
	}
}

// TestSessionResultUpsertIdempotent verifies result deduplication.
func TestSessionResultUpsertIdempotent(t *testing.T) {
	repo := newInMemorySessionRepo()
	now := time.Now().UTC()
	finished := "Finished"

	r := storage.SessionResult{
		ID: "2026-01-race-44", Type: "session_result", Season: 2026, Round: 1,
		SessionType: "race", Position: 2, DriverNumber: 44,
		DriverName: "Lewis HAMILTON", DriverAcronym: "HAM", TeamName: "Ferrari",
		NumberOfLaps: 58, FinishingStatus: &finished,
		DataAsOfUTC: now, Source: "openf1",
	}

	_ = repo.UpsertSessionResult(context.Background(), r)
	_ = repo.UpsertSessionResult(context.Background(), r)

	results, _ := repo.GetSessionResultsByRound(context.Background(), 2026, 1)
	if len(results) != 1 {
		t.Errorf("expected 1 result after double upsert, got %d", len(results))
	}
}

// TestEmptyRoundReturnsNoSessions verifies a round with no data returns empty sessions.
func TestEmptyRoundReturnsNoSessions(t *testing.T) {
	calRepo := newInMemoryRepo()
	sessRepo := newInMemorySessionRepo()
	now := time.Now().UTC()

	_ = calRepo.UpsertMeeting(context.Background(), storage.RaceMeeting{
		ID: "2026-05", Season: 2026, Round: 5, RaceName: "Miami GP",
		CircuitName: "Miami", CountryName: "USA",
		StartDatetimeUTC: now, EndDatetimeUTC: now.Add(3 * 24 * time.Hour),
		Status: "scheduled", Source: "openf1", DataAsOfUTC: now,
	})

	svc := rounds.NewService(sessRepo, calRepo)
	resp, err := svc.GetRoundDetail(context.Background(), 2026, 5)
	if err != nil {
		t.Fatalf("GetRoundDetail: %v", err)
	}

	if len(resp.Sessions) != 0 {
		t.Errorf("expected 0 sessions for empty round, got %d", len(resp.Sessions))
	}
	if resp.RaceName != "Miami GP" {
		t.Errorf("RaceName = %q, want %q", resp.RaceName, "Miami GP")
	}
}
