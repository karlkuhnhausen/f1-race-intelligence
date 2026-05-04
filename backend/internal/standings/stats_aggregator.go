package standings

import (
	"context"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// DriverStats holds aggregated season statistics for a single driver.
type DriverStats struct {
	DriverNumber int
	Wins         int
	Podiums      int
	DNFs         int
	Poles        int
}

// TeamStats holds aggregated season statistics for a constructor team.
type TeamStats struct {
	TeamName string
	Wins     int
	Podiums  int
	DNFs     int
}

// StatsAggregator computes per-driver and per-team stats from session results
// and starting grid data stored in Cosmos DB.
type StatsAggregator struct {
	repo storage.ChampionshipRepository
}

// NewStatsAggregator creates a new stats aggregator.
func NewStatsAggregator(repo storage.ChampionshipRepository) *StatsAggregator {
	return &StatsAggregator{repo: repo}
}

// GetDriverStats computes wins, podiums, DNFs, and poles for all drivers in a season.
func (sa *StatsAggregator) GetDriverStats(ctx context.Context, season int) (map[int]DriverStats, error) {
	results, err := sa.repo.GetChampionshipSessionResults(ctx, season)
	if err != nil {
		return nil, err
	}

	grids, err := sa.repo.GetStartingGridEntries(ctx, season)
	if err != nil {
		return nil, err
	}

	stats := make(map[int]DriverStats)

	for _, r := range results {
		s := stats[r.DriverNumber]
		s.DriverNumber = r.DriverNumber
		if r.Position == 1 && !r.DNF && !r.DSQ {
			s.Wins++
		}
		if r.Position <= 3 && !r.DNF && !r.DSQ {
			s.Podiums++
		}
		if r.DNF {
			s.DNFs++
		}
		stats[r.DriverNumber] = s
	}

	for _, g := range grids {
		s := stats[g.DriverNumber]
		s.DriverNumber = g.DriverNumber
		if g.Position == 1 {
			s.Poles++
		}
		stats[g.DriverNumber] = s
	}

	return stats, nil
}

// GetTeamStats computes combined wins, podiums, and DNFs for all teams in a season.
// Requires a mapping of driver_number → team_name to aggregate correctly.
func (sa *StatsAggregator) GetTeamStats(ctx context.Context, season int, driverTeams map[int]string) (map[string]TeamStats, error) {
	results, err := sa.repo.GetChampionshipSessionResults(ctx, season)
	if err != nil {
		return nil, err
	}

	stats := make(map[string]TeamStats)

	for _, r := range results {
		teamName, ok := driverTeams[r.DriverNumber]
		if !ok {
			continue
		}
		s := stats[teamName]
		s.TeamName = teamName
		if r.Position == 1 && !r.DNF && !r.DSQ {
			s.Wins++
		}
		if r.Position <= 3 && !r.DNF && !r.DSQ {
			s.Podiums++
		}
		if r.DNF {
			s.DNFs++
		}
		stats[teamName] = s
	}

	return stats, nil
}
