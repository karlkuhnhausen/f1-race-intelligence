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
	m.drivers = append(m.drivers, rows...)
	return nil
}

func (m *mockStandingsRepo) GetDriverStandings(_ context.Context, season int) ([]storage.DriverStandingRow, error) {
	var result []storage.DriverStandingRow
	for _, r := range m.drivers {
		if r.Season == season {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *mockStandingsRepo) UpsertConstructorStandings(_ context.Context, rows []storage.ConstructorStandingRow) error {
	m.constructors = append(m.constructors, rows...)
	return nil
}

func (m *mockStandingsRepo) GetConstructorStandings(_ context.Context, season int) ([]storage.ConstructorStandingRow, error) {
	var result []storage.ConstructorStandingRow
	for _, r := range m.constructors {
		if r.Season == season {
			result = append(result, r)
		}
	}
	return result, nil
}

func seedDriverStandings() []storage.DriverStandingRow {
	now := time.Now().UTC()
	return []storage.DriverStandingRow{
		{ID: "d-2026-1", Season: 2026, Position: 1, DriverName: "Max Verstappen", TeamName: "Red Bull Racing", Points: 119, Wins: 4, DataAsOfUTC: now, Source: "hyprace"},
		{ID: "d-2026-2", Season: 2026, Position: 2, DriverName: "Lando Norris", TeamName: "McLaren", Points: 98, Wins: 2, DataAsOfUTC: now, Source: "hyprace"},
		{ID: "d-2026-3", Season: 2026, Position: 3, DriverName: "Charles Leclerc", TeamName: "Ferrari", Points: 87, Wins: 1, DataAsOfUTC: now, Source: "hyprace"},
	}
}

func seedConstructorStandings() []storage.ConstructorStandingRow {
	now := time.Now().UTC()
	return []storage.ConstructorStandingRow{
		{ID: "c-2026-1", Season: 2026, Position: 1, TeamName: "Red Bull Racing", Points: 198, DataAsOfUTC: now, Source: "hyprace"},
		{ID: "c-2026-2", Season: 2026, Position: 2, TeamName: "McLaren", Points: 165, DataAsOfUTC: now, Source: "hyprace"},
		{ID: "c-2026-3", Season: 2026, Position: 3, TeamName: "Ferrari", Points: 150, DataAsOfUTC: now, Source: "hyprace"},
	}
}

func TestStandingsDriversContractReturnsRows(t *testing.T) {
	repo := &mockStandingsRepo{
		drivers: seedDriverStandings(),
	}
	svc := standings.NewService(repo)
	handler := standings.NewHandler(svc, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/standings/drivers?year=2026", nil)
	rec := httptest.NewRecorder()

	handler.GetDrivers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp standings.DriversStandingsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.Year != 2026 {
		t.Errorf("expected year 2026, got %d", resp.Year)
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
	repo := &mockStandingsRepo{drivers: seedDriverStandings()}
	svc := standings.NewService(repo)
	handler := standings.NewHandler(svc, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/standings/drivers?year=2026", nil)
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
	repo := &mockStandingsRepo{
		constructors: seedConstructorStandings(),
	}
	svc := standings.NewService(repo)
	handler := standings.NewHandler(svc, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/standings/constructors?year=2026", nil)
	rec := httptest.NewRecorder()

	handler.GetConstructors(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp standings.ConstructorsStandingsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.Year != 2026 {
		t.Errorf("expected year 2026, got %d", resp.Year)
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
	repo := &mockStandingsRepo{constructors: seedConstructorStandings()}
	svc := standings.NewService(repo)
	handler := standings.NewHandler(svc, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/standings/constructors?year=2026", nil)
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

func TestStandingsDriversMissingYear(t *testing.T) {
	repo := &mockStandingsRepo{}
	svc := standings.NewService(repo)
	handler := standings.NewHandler(svc, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/standings/drivers", nil)
	rec := httptest.NewRecorder()

	handler.GetDrivers(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing year, got %d", rec.Code)
	}
}

func TestStandingsConstructorsMissingYear(t *testing.T) {
	repo := &mockStandingsRepo{}
	svc := standings.NewService(repo)
	handler := standings.NewHandler(svc, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/standings/constructors", nil)
	rec := httptest.NewRecorder()

	handler.GetConstructors(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing year, got %d", rec.Code)
	}
}
