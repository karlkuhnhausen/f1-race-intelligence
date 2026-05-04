package standings

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/standings"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// Service provides championship standings data.
type Service struct {
	standingsRepo    storage.StandingsRepository
	championshipRepo storage.ChampionshipRepository
	sessionRepo      storage.SessionRepository
	statsAggregator  *standings.StatsAggregator
}

// NewService creates a standings service.
func NewService(standingsRepo storage.StandingsRepository, championshipRepo storage.ChampionshipRepository, sessionRepo storage.SessionRepository) *Service {
	return &Service{
		standingsRepo:    standingsRepo,
		championshipRepo: championshipRepo,
		sessionRepo:      sessionRepo,
		statsAggregator:  standings.NewStatsAggregator(championshipRepo),
	}
}

// driverIdentity holds resolved driver metadata from session results.
type driverIdentity struct {
	DriverName string
	TeamName   string
	TeamColor  string
}

// resolveDriverIdentities builds a map of driver_number → identity from session results.
func (s *Service) resolveDriverIdentities(ctx context.Context, season int) (map[int]driverIdentity, error) {
	results, err := s.sessionRepo.GetSessionResultsBySeason(ctx, season)
	if err != nil {
		return nil, err
	}

	identities := make(map[int]driverIdentity)
	for _, r := range results {
		// Use the most recent entry for each driver (results are chronological).
		identities[r.DriverNumber] = driverIdentity{
			DriverName: r.DriverName,
			TeamName:   r.TeamName,
			TeamColor:  "", // team_colour not in SessionResult; will resolve from analysis data
		}
	}
	return identities, nil
}

// GetDrivers returns the drivers championship for the given season.
func (s *Service) GetDrivers(ctx context.Context, season int) (*DriversStandingsResponse, error) {
	snapshots, err := s.championshipRepo.GetDriverChampionshipSnapshots(ctx, season)
	if err != nil {
		return nil, err
	}

	// Get the latest snapshot per driver (highest session_key).
	latestByDriver := make(map[int]storage.DriverChampionshipSnapshot)
	var latestDataAsOf time.Time
	for _, snap := range snapshots {
		existing, exists := latestByDriver[snap.DriverNumber]
		if !exists || snap.SessionKey > existing.SessionKey {
			latestByDriver[snap.DriverNumber] = snap
		}
		if snap.DataAsOfUTC.After(latestDataAsOf) {
			latestDataAsOf = snap.DataAsOfUTC
		}
	}

	// Resolve driver identities.
	identities, err := s.resolveDriverIdentities(ctx, season)
	if err != nil {
		return nil, err
	}

	// Get stats.
	driverStats, err := s.statsAggregator.GetDriverStats(ctx, season)
	if err != nil {
		return nil, err
	}

	// Build DTOs sorted by position.
	dtos := make([]DriverStandingDTO, 0, len(latestByDriver))
	for driverNum, snap := range latestByDriver {
		identity := identities[driverNum]
		stats := driverStats[driverNum]
		dtos = append(dtos, DriverStandingDTO{
			Position:     snap.PositionCurrent,
			DriverNumber: driverNum,
			DriverName:   identity.DriverName,
			TeamName:     identity.TeamName,
			TeamColor:    identity.TeamColor,
			Points:       snap.PointsCurrent,
			Wins:         stats.Wins,
			Podiums:      stats.Podiums,
			DNFs:         stats.DNFs,
			Poles:        stats.Poles,
		})
	}
	sort.Slice(dtos, func(i, j int) bool {
		return dtos[i].Position < dtos[j].Position
	})

	return &DriversStandingsResponse{
		Year:        season,
		DataAsOfUTC: latestDataAsOf,
		Rows:        dtos,
	}, nil
}

// GetConstructors returns the constructors championship for the given season.
func (s *Service) GetConstructors(ctx context.Context, season int) (*ConstructorsStandingsResponse, error) {
	snapshots, err := s.championshipRepo.GetTeamChampionshipSnapshots(ctx, season)
	if err != nil {
		return nil, err
	}

	// Get the latest snapshot per team (highest session_key).
	latestByTeam := make(map[string]storage.TeamChampionshipSnapshot)
	var latestDataAsOf time.Time
	for _, snap := range snapshots {
		existing, exists := latestByTeam[snap.TeamSlug]
		if !exists || snap.SessionKey > existing.SessionKey {
			latestByTeam[snap.TeamSlug] = snap
		}
		if snap.DataAsOfUTC.After(latestDataAsOf) {
			latestDataAsOf = snap.DataAsOfUTC
		}
	}

	// Get driver identities to build driver→team mapping for stats.
	identities, err := s.resolveDriverIdentities(ctx, season)
	if err != nil {
		return nil, err
	}
	driverTeams := make(map[int]string)
	for driverNum, id := range identities {
		driverTeams[driverNum] = id.TeamName
	}

	// Get team stats.
	teamStats, err := s.statsAggregator.GetTeamStats(ctx, season, driverTeams)
	if err != nil {
		return nil, err
	}

	// Build DTOs sorted by position.
	dtos := make([]ConstructorStandingDTO, 0, len(latestByTeam))
	for _, snap := range latestByTeam {
		stats := teamStats[snap.TeamName]
		dtos = append(dtos, ConstructorStandingDTO{
			Position:  snap.PositionCurrent,
			TeamName:  snap.TeamName,
			TeamColor: "", // Will resolve when we have team color data
			Points:    snap.PointsCurrent,
			Wins:      stats.Wins,
			Podiums:   stats.Podiums,
			DNFs:      stats.DNFs,
		})
	}
	sort.Slice(dtos, func(i, j int) bool {
		return dtos[i].Position < dtos[j].Position
	})

	return &ConstructorsStandingsResponse{
		Year:        season,
		DataAsOfUTC: latestDataAsOf,
		Rows:        dtos,
	}, nil
}

// GetDriverProgression returns per-round cumulative points for each driver.
func (s *Service) GetDriverProgression(ctx context.Context, season int) (*DriversProgressionResponse, error) {
	snapshots, err := s.championshipRepo.GetDriverChampionshipSnapshots(ctx, season)
	if err != nil {
		return nil, err
	}

	// Collect unique session keys (each represents a round) and sort them.
	sessionKeySet := make(map[int]struct{})
	for _, snap := range snapshots {
		sessionKeySet[snap.SessionKey] = struct{}{}
	}
	sessionKeys := make([]int, 0, len(sessionKeySet))
	for sk := range sessionKeySet {
		sessionKeys = append(sessionKeys, sk)
	}
	sort.Ints(sessionKeys)

	rounds := make([]string, len(sessionKeys))
	skIndex := make(map[int]int)
	for i, sk := range sessionKeys {
		rounds[i] = fmt.Sprintf("Round %d", i+1)
		skIndex[sk] = i
	}

	identities, err := s.resolveDriverIdentities(ctx, season)
	if err != nil {
		return nil, err
	}

	type driverData struct {
		driverNum int
		points    []int
	}
	byDriver := make(map[int]*driverData)
	for _, snap := range snapshots {
		dd, ok := byDriver[snap.DriverNumber]
		if !ok {
			dd = &driverData{
				driverNum: snap.DriverNumber,
				points:    make([]int, len(sessionKeys)),
			}
			byDriver[snap.DriverNumber] = dd
		}
		dd.points[skIndex[snap.SessionKey]] = int(snap.PointsCurrent)
	}

	entries := make([]DriverProgressionEntry, 0, len(byDriver))
	for driverNum, dd := range byDriver {
		id := identities[driverNum]
		entries = append(entries, DriverProgressionEntry{
			DriverNumber:  driverNum,
			DriverName:    id.DriverName,
			TeamName:      id.TeamName,
			TeamColor:     id.TeamColor,
			PointsByRound: dd.points,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		lastI := entries[i].PointsByRound[len(entries[i].PointsByRound)-1]
		lastJ := entries[j].PointsByRound[len(entries[j].PointsByRound)-1]
		return lastI > lastJ
	})

	return &DriversProgressionResponse{
		Year:    season,
		Rounds:  rounds,
		Drivers: entries,
	}, nil
}

// GetConstructorProgression returns per-round cumulative points for each team.
func (s *Service) GetConstructorProgression(ctx context.Context, season int) (*ConstructorsProgressionResponse, error) {
	snapshots, err := s.championshipRepo.GetTeamChampionshipSnapshots(ctx, season)
	if err != nil {
		return nil, err
	}

	sessionKeySet := make(map[int]struct{})
	for _, snap := range snapshots {
		sessionKeySet[snap.SessionKey] = struct{}{}
	}
	sessionKeys := make([]int, 0, len(sessionKeySet))
	for sk := range sessionKeySet {
		sessionKeys = append(sessionKeys, sk)
	}
	sort.Ints(sessionKeys)

	rounds := make([]string, len(sessionKeys))
	skIndex := make(map[int]int)
	for i, sk := range sessionKeys {
		rounds[i] = fmt.Sprintf("Round %d", i+1)
		skIndex[sk] = i
	}

	type teamData struct {
		teamName string
		points   []int
	}
	byTeam := make(map[string]*teamData)
	for _, snap := range snapshots {
		td, ok := byTeam[snap.TeamSlug]
		if !ok {
			td = &teamData{
				teamName: snap.TeamName,
				points:   make([]int, len(sessionKeys)),
			}
			byTeam[snap.TeamSlug] = td
		}
		td.points[skIndex[snap.SessionKey]] = int(snap.PointsCurrent)
	}

	entries := make([]TeamProgressionEntry, 0, len(byTeam))
	for _, td := range byTeam {
		entries = append(entries, TeamProgressionEntry{
			TeamName:      td.teamName,
			TeamColor:     "",
			PointsByRound: td.points,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		lastI := entries[i].PointsByRound[len(entries[i].PointsByRound)-1]
		lastJ := entries[j].PointsByRound[len(entries[j].PointsByRound)-1]
		return lastI > lastJ
	})

	return &ConstructorsProgressionResponse{
		Year:   season,
		Rounds: rounds,
		Teams:  entries,
	}, nil
}
