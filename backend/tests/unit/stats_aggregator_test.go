package unit

import (
	"context"
	"testing"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/standings"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// mockChampionshipRepo is an in-memory implementation of ChampionshipRepository for testing.
type mockChampionshipRepo struct {
	driverSnapshots []storage.DriverChampionshipSnapshot
	teamSnapshots   []storage.TeamChampionshipSnapshot
	results         []storage.ChampionshipSessionResult
	grids           []storage.StartingGridEntry
}

func (m *mockChampionshipRepo) UpsertDriverChampionshipSnapshots(_ context.Context, s []storage.DriverChampionshipSnapshot) error {
	m.driverSnapshots = append(m.driverSnapshots, s...)
	return nil
}

func (m *mockChampionshipRepo) GetDriverChampionshipSnapshots(_ context.Context, _ int) ([]storage.DriverChampionshipSnapshot, error) {
	return m.driverSnapshots, nil
}

func (m *mockChampionshipRepo) UpsertTeamChampionshipSnapshots(_ context.Context, s []storage.TeamChampionshipSnapshot) error {
	m.teamSnapshots = append(m.teamSnapshots, s...)
	return nil
}

func (m *mockChampionshipRepo) GetTeamChampionshipSnapshots(_ context.Context, _ int) ([]storage.TeamChampionshipSnapshot, error) {
	return m.teamSnapshots, nil
}

func (m *mockChampionshipRepo) UpsertChampionshipSessionResults(_ context.Context, r []storage.ChampionshipSessionResult) error {
	m.results = append(m.results, r...)
	return nil
}

func (m *mockChampionshipRepo) GetChampionshipSessionResults(_ context.Context, _ int) ([]storage.ChampionshipSessionResult, error) {
	return m.results, nil
}

func (m *mockChampionshipRepo) UpsertStartingGridEntries(_ context.Context, e []storage.StartingGridEntry) error {
	m.grids = append(m.grids, e...)
	return nil
}

func (m *mockChampionshipRepo) GetStartingGridEntries(_ context.Context, _ int) ([]storage.StartingGridEntry, error) {
	return m.grids, nil
}

// --- StatsAggregator tests ---

func TestStatsAggregator_GetDriverStats(t *testing.T) {
	repo := &mockChampionshipRepo{
		results: []storage.ChampionshipSessionResult{
			{DriverNumber: 1, Position: 1, DNF: false, DSQ: false, SessionKey: 100},
			{DriverNumber: 1, Position: 2, DNF: false, DSQ: false, SessionKey: 101},
			{DriverNumber: 1, Position: 5, DNF: true, DSQ: false, SessionKey: 102},
			{DriverNumber: 44, Position: 3, DNF: false, DSQ: false, SessionKey: 100},
			{DriverNumber: 44, Position: 1, DNF: false, DSQ: false, SessionKey: 101},
			{DriverNumber: 44, Position: 10, DNF: false, DSQ: false, SessionKey: 102},
		},
		grids: []storage.StartingGridEntry{
			{DriverNumber: 1, Position: 1, MeetingKey: 200},
			{DriverNumber: 1, Position: 3, MeetingKey: 201},
			{DriverNumber: 44, Position: 2, MeetingKey: 200},
			{DriverNumber: 44, Position: 1, MeetingKey: 201},
		},
	}

	agg := standings.NewStatsAggregator(repo)
	stats, err := agg.GetDriverStats(context.Background(), 2026)
	if err != nil {
		t.Fatalf("GetDriverStats returned error: %v", err)
	}

	// Driver 1: wins=1, podiums=2 (P1+P2), DNFs=1, poles=1
	d1 := stats[1]
	if d1.Wins != 1 {
		t.Errorf("driver 1 wins: got %d, want 1", d1.Wins)
	}
	if d1.Podiums != 2 {
		t.Errorf("driver 1 podiums: got %d, want 2", d1.Podiums)
	}
	if d1.DNFs != 1 {
		t.Errorf("driver 1 dnfs: got %d, want 1", d1.DNFs)
	}
	if d1.Poles != 1 {
		t.Errorf("driver 1 poles: got %d, want 1", d1.Poles)
	}

	// Driver 44: wins=1, podiums=2 (P3+P1), DNFs=0, poles=1
	d44 := stats[44]
	if d44.Wins != 1 {
		t.Errorf("driver 44 wins: got %d, want 1", d44.Wins)
	}
	if d44.Podiums != 2 {
		t.Errorf("driver 44 podiums: got %d, want 2", d44.Podiums)
	}
	if d44.DNFs != 0 {
		t.Errorf("driver 44 dnfs: got %d, want 0", d44.DNFs)
	}
	if d44.Poles != 1 {
		t.Errorf("driver 44 poles: got %d, want 1", d44.Poles)
	}
}

func TestStatsAggregator_GetDriverStats_DNFDoesNotCountAsPodium(t *testing.T) {
	repo := &mockChampionshipRepo{
		results: []storage.ChampionshipSessionResult{
			// Position 1 but DNF — should NOT count as win or podium
			{DriverNumber: 10, Position: 1, DNF: true, DSQ: false, SessionKey: 100},
		},
	}

	agg := standings.NewStatsAggregator(repo)
	stats, err := agg.GetDriverStats(context.Background(), 2026)
	if err != nil {
		t.Fatalf("GetDriverStats returned error: %v", err)
	}

	d10 := stats[10]
	if d10.Wins != 0 {
		t.Errorf("DNF driver wins: got %d, want 0", d10.Wins)
	}
	if d10.Podiums != 0 {
		t.Errorf("DNF driver podiums: got %d, want 0", d10.Podiums)
	}
	if d10.DNFs != 1 {
		t.Errorf("DNF driver dnfs: got %d, want 1", d10.DNFs)
	}
}

func TestStatsAggregator_GetDriverStats_DSQDoesNotCountAsWin(t *testing.T) {
	repo := &mockChampionshipRepo{
		results: []storage.ChampionshipSessionResult{
			{DriverNumber: 5, Position: 1, DNF: false, DSQ: true, SessionKey: 100},
		},
	}

	agg := standings.NewStatsAggregator(repo)
	stats, err := agg.GetDriverStats(context.Background(), 2026)
	if err != nil {
		t.Fatalf("GetDriverStats returned error: %v", err)
	}

	d5 := stats[5]
	if d5.Wins != 0 {
		t.Errorf("DSQ driver wins: got %d, want 0", d5.Wins)
	}
	if d5.Podiums != 0 {
		t.Errorf("DSQ driver podiums: got %d, want 0", d5.Podiums)
	}
}

func TestStatsAggregator_GetTeamStats(t *testing.T) {
	repo := &mockChampionshipRepo{
		results: []storage.ChampionshipSessionResult{
			{DriverNumber: 1, Position: 1, DNF: false, DSQ: false, SessionKey: 100},
			{DriverNumber: 11, Position: 3, DNF: false, DSQ: false, SessionKey: 100},
			{DriverNumber: 1, Position: 5, DNF: true, DSQ: false, SessionKey: 101},
			{DriverNumber: 11, Position: 2, DNF: false, DSQ: false, SessionKey: 101},
		},
	}

	driverTeams := map[int]string{
		1:  "Red Bull Racing",
		11: "Red Bull Racing",
	}

	agg := standings.NewStatsAggregator(repo)
	stats, err := agg.GetTeamStats(context.Background(), 2026, driverTeams)
	if err != nil {
		t.Fatalf("GetTeamStats returned error: %v", err)
	}

	rbr := stats["Red Bull Racing"]
	// Wins: driver 1 P1 in session 100 = 1
	if rbr.Wins != 1 {
		t.Errorf("RBR wins: got %d, want 1", rbr.Wins)
	}
	// Podiums: driver 1 P1 + driver 11 P3 + driver 11 P2 = 3
	if rbr.Podiums != 3 {
		t.Errorf("RBR podiums: got %d, want 3", rbr.Podiums)
	}
	// DNFs: driver 1 DNF in session 101 = 1
	if rbr.DNFs != 1 {
		t.Errorf("RBR dnfs: got %d, want 1", rbr.DNFs)
	}
}

func TestStatsAggregator_EmptyResults(t *testing.T) {
	repo := &mockChampionshipRepo{}

	agg := standings.NewStatsAggregator(repo)
	stats, err := agg.GetDriverStats(context.Background(), 2026)
	if err != nil {
		t.Fatalf("GetDriverStats returned error: %v", err)
	}
	if len(stats) != 0 {
		t.Errorf("expected empty stats, got %d entries", len(stats))
	}
}
