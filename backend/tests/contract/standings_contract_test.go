package contract

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"log/slog"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/api/standings"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// mockStandingsRepo implements storage.StandingsRepository for testing.
type mockStandingsRepo struct {
	drivers      []storage.DriverStandingRow
	constructors []storage.ConstructorStandingRow
}

func (m *mockStandingsRepo) UpsertDriverStandings(_ context.Context, rows []storage.DriverStandingRow) error {
	return nil
}

func (m *mockStandingsRepo) GetDriverStandings(_ context.Context, _ int) ([]storage.DriverStandingRow, error) {
	return m.drivers, nil
}

func (m *mockStandingsRepo) UpsertConstructorStandings(_ context.Context, rows []storage.ConstructorStandingRow) error {
	return nil
}

func (m *mockStandingsRepo) GetConstructorStandings(_ context.Context, _ int) ([]storage.ConstructorStandingRow, error) {
	return m.constructors, nil
}

// mockChampionshipRepo implements storage.ChampionshipRepository for testing.
type mockChampionshipRepo struct {
	driverSnapshots []storage.DriverChampionshipSnapshot
	teamSnapshots   []storage.TeamChampionshipSnapshot
	sessionResults  []storage.ChampionshipSessionResult
	startingGrid    []storage.StartingGridEntry
}

func (m *mockChampionshipRepo) UpsertDriverChampionshipSnapshots(_ context.Context, s []storage.DriverChampionshipSnapshot) error {
	return nil
}
func (m *mockChampionshipRepo) GetDriverChampionshipSnapshots(_ context.Context, _ int) ([]storage.DriverChampionshipSnapshot, error) {
	return m.driverSnapshots, nil
}
func (m *mockChampionshipRepo) UpsertTeamChampionshipSnapshots(_ context.Context, s []storage.TeamChampionshipSnapshot) error {
	return nil
}
func (m *mockChampionshipRepo) GetTeamChampionshipSnapshots(_ context.Context, _ int) ([]storage.TeamChampionshipSnapshot, error) {
	return m.teamSnapshots, nil
}
func (m *mockChampionshipRepo) UpsertChampionshipSessionResults(_ context.Context, r []storage.ChampionshipSessionResult) error {
	return nil
}
func (m *mockChampionshipRepo) GetChampionshipSessionResults(_ context.Context, _ int) ([]storage.ChampionshipSessionResult, error) {
	return m.sessionResults, nil
}
func (m *mockChampionshipRepo) UpsertStartingGridEntries(_ context.Context, e []storage.StartingGridEntry) error {
	return nil
}
func (m *mockChampionshipRepo) GetStartingGridEntries(_ context.Context, _ int) ([]storage.StartingGridEntry, error) {
	return m.startingGrid, nil
}

// mockSessionRepoForStandings implements storage.SessionRepository for identity resolution.
type mockSessionRepoForStandings struct {
	results []storage.SessionResult
}

func (m *mockSessionRepoForStandings) UpsertSession(_ context.Context, _ storage.Session) error {
	return nil
}
func (m *mockSessionRepoForStandings) UpsertSessionResult(_ context.Context, _ storage.SessionResult) error {
	return nil
}
func (m *mockSessionRepoForStandings) GetSessionsByRound(_ context.Context, _, _ int) ([]storage.Session, error) {
	return nil, nil
}
func (m *mockSessionRepoForStandings) GetSessionResultsByRound(_ context.Context, _, _ int) ([]storage.SessionResult, error) {
	return nil, nil
}
func (m *mockSessionRepoForStandings) GetSessionResultsBySeason(_ context.Context, _ int) ([]storage.SessionResult, error) {
	return m.results, nil
}
func (m *mockSessionRepoForStandings) GetFinalizedSessionKeys(_ context.Context, _ int) (map[int]int, error) {
	return nil, nil
}
func (m *mockSessionRepoForStandings) GetCompletedRaceSessionKeys(_ context.Context, _ int, _ time.Time) (map[int]struct{}, error) {
	return nil, nil
}
func (m *mockSessionRepoForStandings) GetCompletedRaceSessions(_ context.Context, _ int, _ time.Time) ([]storage.Session, error) {
	return nil, nil
}
func (m *mockSessionRepoForStandings) GetFinalizedSessions(_ context.Context, _ int) ([]storage.Session, error) {
	return nil, nil
}
func (m *mockSessionRepoForStandings) DeleteSession(_ context.Context, _ int, _ string) error {
	return nil
}
func (m *mockSessionRepoForStandings) DeleteSessionResultsBySessionType(_ context.Context, _, _ int, _ string) error {
	return nil
}
func (m *mockSessionRepoForStandings) GetSessionsByMeetingKey(_ context.Context, _, _ int) ([]storage.Session, error) {
	return nil, nil
}
func (m *mockSessionRepoForStandings) GetSessionResultsByMeetingKey(_ context.Context, _, _ int) ([]storage.SessionResult, error) {
	return nil, nil
}

func newTestService() (*standings.Service, *mockChampionshipRepo, *mockSessionRepoForStandings) {
	now := time.Now().UTC()
	champRepo := &mockChampionshipRepo{
		driverSnapshots: []storage.DriverChampionshipSnapshot{
			{Season: 2025, SessionKey: 9000, DriverNumber: 1, PositionCurrent: 1, PointsCurrent: 119, DataAsOfUTC: now},
			{Season: 2025, SessionKey: 9000, DriverNumber: 4, PositionCurrent: 2, PointsCurrent: 98, DataAsOfUTC: now},
			{Season: 2025, SessionKey: 9000, DriverNumber: 16, PositionCurrent: 3, PointsCurrent: 87, DataAsOfUTC: now},
		},
		teamSnapshots: []storage.TeamChampionshipSnapshot{
			{Season: 2025, SessionKey: 9000, TeamSlug: "red-bull-racing", TeamName: "Red Bull Racing", PositionCurrent: 1, PointsCurrent: 198, DataAsOfUTC: now},
			{Season: 2025, SessionKey: 9000, TeamSlug: "mclaren", TeamName: "McLaren", PositionCurrent: 2, PointsCurrent: 165, DataAsOfUTC: now},
			{Season: 2025, SessionKey: 9000, TeamSlug: "ferrari", TeamName: "Ferrari", PositionCurrent: 3, PointsCurrent: 150, DataAsOfUTC: now},
		},
		sessionResults: []storage.ChampionshipSessionResult{},
		startingGrid:   []storage.StartingGridEntry{},
	}
	sessionRepo := &mockSessionRepoForStandings{
		results: []storage.SessionResult{
			{Season: 2025, DriverNumber: 1, DriverName: "Max Verstappen", TeamName: "Red Bull Racing"},
			{Season: 2025, DriverNumber: 4, DriverName: "Lando Norris", TeamName: "McLaren"},
			{Season: 2025, DriverNumber: 16, DriverName: "Charles Leclerc", TeamName: "Ferrari"},
		},
	}
	standingsRepo := &mockStandingsRepo{}
	svc := standings.NewService(standingsRepo, champRepo, sessionRepo, nil)
	return svc, champRepo, sessionRepo
}

func TestStandingsDriversContractReturnsRows(t *testing.T) {
	svc, _, _ := newTestService()
	handler := standings.NewHandler(svc, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/standings/drivers?year=2025", nil)
	rec := httptest.NewRecorder()

	handler.GetDrivers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp standings.DriversStandingsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.Year != 2025 {
		t.Errorf("expected year 2025, got %d", resp.Year)
	}

	if len(resp.Rows) != 3 {
		t.Fatalf("expected 3 driver rows, got %d", len(resp.Rows))
	}

	first := resp.Rows[0]
	if first.Position != 1 {
		t.Errorf("expected position 1, got %d", first.Position)
	}
	if first.DriverName == "" {
		t.Error("driver_name should not be empty")
	}
	if first.TeamName == "" {
		t.Error("team_name should not be empty")
	}
}

func TestStandingsDriversContractRequiredFields(t *testing.T) {
	svc, _, _ := newTestService()
	handler := standings.NewHandler(svc, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/standings/drivers?year=2025", nil)
	rec := httptest.NewRecorder()

	handler.GetDrivers(rec, req)

	var raw map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode: %v", err)
	}

	required := []string{"year", "data_as_of_utc", "rows"}
	for _, field := range required {
		if _, ok := raw[field]; !ok {
			t.Errorf("missing required field: %s", field)
		}
	}

	rows := raw["rows"].([]interface{})
	if len(rows) == 0 {
		t.Fatal("expected at least one row")
	}

	row := rows[0].(map[string]interface{})
	rowFields := []string{"position", "driver_name", "team_name", "points", "wins"}
	for _, field := range rowFields {
		if _, ok := row[field]; !ok {
			t.Errorf("missing required row field: %s", field)
		}
	}
}

func TestStandingsConstructorsContractReturnsRows(t *testing.T) {
	svc, _, _ := newTestService()
	handler := standings.NewHandler(svc, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/standings/constructors?year=2025", nil)
	rec := httptest.NewRecorder()

	handler.GetConstructors(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp standings.ConstructorsStandingsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.Year != 2025 {
		t.Errorf("expected year 2025, got %d", resp.Year)
	}

	if len(resp.Rows) != 3 {
		t.Fatalf("expected 3 constructor rows, got %d", len(resp.Rows))
	}

	first := resp.Rows[0]
	if first.Position != 1 {
		t.Errorf("expected position 1, got %d", first.Position)
	}
	if first.TeamName == "" {
		t.Error("team_name should not be empty")
	}
}

func TestStandingsConstructorsContractRequiredFields(t *testing.T) {
	svc, _, _ := newTestService()
	handler := standings.NewHandler(svc, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/standings/constructors?year=2025", nil)
	rec := httptest.NewRecorder()

	handler.GetConstructors(rec, req)

	var raw map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode: %v", err)
	}

	required := []string{"year", "data_as_of_utc", "rows"}
	for _, field := range required {
		if _, ok := raw[field]; !ok {
			t.Errorf("missing required field: %s", field)
		}
	}

	rows := raw["rows"].([]interface{})
	if len(rows) == 0 {
		t.Fatal("expected at least one row")
	}

	row := rows[0].(map[string]interface{})
	rowFields := []string{"position", "team_name", "points"}
	for _, field := range rowFields {
		if _, ok := row[field]; !ok {
			t.Errorf("missing required row field: %s", field)
		}
	}
}

func TestStandingsDriversDefaultsToCurrentYear(t *testing.T) {
	svc, _, _ := newTestService()
	handler := standings.NewHandler(svc, slog.Default())

	// No year parameter — should default to current year, not 400.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/standings/drivers", nil)
	rec := httptest.NewRecorder()

	handler.GetDrivers(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for missing year (defaults to current), got %d", rec.Code)
	}
}

func TestStandingsDriversInvalidYear(t *testing.T) {
	svc, _, _ := newTestService()
	handler := standings.NewHandler(svc, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/standings/drivers?year=2020", nil)
	rec := httptest.NewRecorder()

	handler.GetDrivers(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for year before 2023, got %d", rec.Code)
	}
}
