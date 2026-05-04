package standings

import (
	"context"
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
