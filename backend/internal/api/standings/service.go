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

// GetDriverComparison compares two drivers' season stats and progression.
func (s *Service) GetDriverComparison(ctx context.Context, season, driver1, driver2 int) (*DriverComparisonResponse, error) {
	snapshots, err := s.championshipRepo.GetDriverChampionshipSnapshots(ctx, season)
	if err != nil {
		return nil, err
	}

	identities, err := s.resolveDriverIdentities(ctx, season)
	if err != nil {
		return nil, err
	}

	driverStats, err := s.statsAggregator.GetDriverStats(ctx, season)
	if err != nil {
		return nil, err
	}

	// Collect session keys and build per-driver progression.
	sessionKeySet := make(map[int]struct{})
	pointsByDriver := map[int]map[int]int{driver1: {}, driver2: {}}
	var latestPoints [2]int

	for _, snap := range snapshots {
		sessionKeySet[snap.SessionKey] = struct{}{}
		if snap.DriverNumber == driver1 || snap.DriverNumber == driver2 {
			if _, ok := pointsByDriver[snap.DriverNumber]; ok {
				pointsByDriver[snap.DriverNumber][snap.SessionKey] = int(snap.PointsCurrent)
			}
		}
	}

	sessionKeys := make([]int, 0, len(sessionKeySet))
	for sk := range sessionKeySet {
		sessionKeys = append(sessionKeys, sk)
	}
	sort.Ints(sessionKeys)

	rounds := make([]string, len(sessionKeys))
	d1Points := make([]int, len(sessionKeys))
	d2Points := make([]int, len(sessionKeys))
	for i, sk := range sessionKeys {
		rounds[i] = fmt.Sprintf("Round %d", i+1)
		d1Points[i] = pointsByDriver[driver1][sk]
		d2Points[i] = pointsByDriver[driver2][sk]
	}
	if len(sessionKeys) > 0 {
		lastSK := sessionKeys[len(sessionKeys)-1]
		latestPoints[0] = pointsByDriver[driver1][lastSK]
		latestPoints[1] = pointsByDriver[driver2][lastSK]
	}

	id1 := identities[driver1]
	id2 := identities[driver2]
	s1 := driverStats[driver1]
	s2 := driverStats[driver2]

	return &DriverComparisonResponse{
		Year: season,
		Driver1: ComparisonDriverStats{
			DriverNumber: driver1, DriverName: id1.DriverName, TeamName: id1.TeamName, TeamColor: id1.TeamColor,
			Points: latestPoints[0], Wins: s1.Wins, Podiums: s1.Podiums, DNFs: s1.DNFs, Poles: s1.Poles,
		},
		Driver2: ComparisonDriverStats{
			DriverNumber: driver2, DriverName: id2.DriverName, TeamName: id2.TeamName, TeamColor: id2.TeamColor,
			Points: latestPoints[1], Wins: s2.Wins, Podiums: s2.Podiums, DNFs: s2.DNFs, Poles: s2.Poles,
		},
		Deltas: ComparisonDeltas{
			Points: latestPoints[0] - latestPoints[1], Wins: s1.Wins - s2.Wins,
			Podiums: s1.Podiums - s2.Podiums, DNFs: s1.DNFs - s2.DNFs, Poles: s1.Poles - s2.Poles,
		},
		Rounds:        rounds,
		Driver1Points: d1Points,
		Driver2Points: d2Points,
	}, nil
}

// GetConstructorComparison compares two teams' season stats and progression.
func (s *Service) GetConstructorComparison(ctx context.Context, season int, team1, team2 string) (*ConstructorComparisonResponse, error) {
	snapshots, err := s.championshipRepo.GetTeamChampionshipSnapshots(ctx, season)
	if err != nil {
		return nil, err
	}

	identities, err := s.resolveDriverIdentities(ctx, season)
	if err != nil {
		return nil, err
	}
	driverTeams := make(map[int]string)
	for dn, id := range identities {
		driverTeams[dn] = id.TeamName
	}
	teamStats, err := s.statsAggregator.GetTeamStats(ctx, season, driverTeams)
	if err != nil {
		return nil, err
	}

	sessionKeySet := make(map[int]struct{})
	pointsByTeam := map[string]map[int]int{team1: {}, team2: {}}

	for _, snap := range snapshots {
		sessionKeySet[snap.SessionKey] = struct{}{}
		if snap.TeamName == team1 || snap.TeamName == team2 {
			if _, ok := pointsByTeam[snap.TeamName]; ok {
				pointsByTeam[snap.TeamName][snap.SessionKey] = int(snap.PointsCurrent)
			}
		}
	}

	sessionKeys := make([]int, 0, len(sessionKeySet))
	for sk := range sessionKeySet {
		sessionKeys = append(sessionKeys, sk)
	}
	sort.Ints(sessionKeys)

	rounds := make([]string, len(sessionKeys))
	t1Points := make([]int, len(sessionKeys))
	t2Points := make([]int, len(sessionKeys))
	var latestPoints [2]int
	for i, sk := range sessionKeys {
		rounds[i] = fmt.Sprintf("Round %d", i+1)
		t1Points[i] = pointsByTeam[team1][sk]
		t2Points[i] = pointsByTeam[team2][sk]
	}
	if len(sessionKeys) > 0 {
		lastSK := sessionKeys[len(sessionKeys)-1]
		latestPoints[0] = pointsByTeam[team1][lastSK]
		latestPoints[1] = pointsByTeam[team2][lastSK]
	}

	ts1 := teamStats[team1]
	ts2 := teamStats[team2]

	return &ConstructorComparisonResponse{
		Year: season,
		Team1: ComparisonTeamStats{
			TeamName: team1, TeamColor: "", Points: latestPoints[0],
			Wins: ts1.Wins, Podiums: ts1.Podiums, DNFs: ts1.DNFs,
		},
		Team2: ComparisonTeamStats{
			TeamName: team2, TeamColor: "", Points: latestPoints[1],
			Wins: ts2.Wins, Podiums: ts2.Podiums, DNFs: ts2.DNFs,
		},
		Deltas: ComparisonDeltas{
			Points: latestPoints[0] - latestPoints[1], Wins: ts1.Wins - ts2.Wins,
			Podiums: ts1.Podiums - ts2.Podiums, DNFs: ts1.DNFs - ts2.DNFs,
		},
		Rounds:      rounds,
		Team1Points: t1Points,
		Team2Points: t2Points,
	}, nil
}
