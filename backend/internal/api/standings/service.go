package standings

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/domain"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/standings"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// Service provides championship standings data.
type Service struct {
	standingsRepo    storage.StandingsRepository
	championshipRepo storage.ChampionshipRepository
	sessionRepo      storage.SessionRepository
	calendarRepo     storage.CalendarRepository
	statsAggregator  *standings.StatsAggregator
}

// NewService creates a standings service.
func NewService(standingsRepo storage.StandingsRepository, championshipRepo storage.ChampionshipRepository, sessionRepo storage.SessionRepository, calendarRepo storage.CalendarRepository) *Service {
	return &Service{
		standingsRepo:    standingsRepo,
		championshipRepo: championshipRepo,
		sessionRepo:      sessionRepo,
		calendarRepo:     calendarRepo,
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
			TeamColor:  domain.GetTeamColor(r.TeamName),
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
			TeamColor: domain.GetTeamColor(snap.TeamName),
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

// buildRoundLabels creates human-readable X-axis labels for progression charts.
// For meetings with only a race session: just the circuit name (e.g., "Melbourne").
// For sprint weekends (both race + sprint): "Circuit Race" / "Circuit Sprint".
func (s *Service) buildRoundLabels(ctx context.Context, season int, sessionKeys []int, now time.Time) []string {
	labels := make([]string, len(sessionKeys))

	// Default labels in case lookup fails.
	for i := range sessionKeys {
		labels[i] = fmt.Sprintf("Round %d", i+1)
	}

	if s.calendarRepo == nil {
		return labels
	}

	// Fetch completed sessions to map session_key → (meeting_key, session_type).
	sessions, err := s.sessionRepo.GetCompletedRaceSessions(ctx, season, now)
	if err != nil {
		return labels
	}

	type sessionInfo struct {
		meetingKey  int
		sessionType string
	}
	sessionMap := make(map[int]sessionInfo, len(sessions))
	// Track which meetings have multiple scoring sessions (sprint weekends).
	meetingSessions := make(map[int]int) // meeting_key → count of scoring sessions
	for _, sess := range sessions {
		sessionMap[sess.SessionKey] = sessionInfo{
			meetingKey:  sess.MeetingKey,
			sessionType: sess.SessionType,
		}
		meetingSessions[sess.MeetingKey]++
	}

	// Fetch meetings to get circuit names.
	meetings, err := s.calendarRepo.GetMeetingsBySeason(ctx, season)
	if err != nil {
		return labels
	}
	circuitByMeeting := make(map[int]string, len(meetings))
	for _, m := range meetings {
		// Use country name as the short label (e.g., "Australia", "China", "USA").
		// Fall back to circuit name if country is empty.
		name := m.CountryName
		if name == "" {
			name = m.CircuitName
		}
		// Shorten common long names.
		name = shortenCircuitLabel(name)
		circuitByMeeting[m.MeetingKey] = name
	}

	// Build labels.
	for i, sk := range sessionKeys {
		info, ok := sessionMap[sk]
		if !ok {
			continue
		}
		circuit, ok := circuitByMeeting[info.meetingKey]
		if !ok {
			continue
		}
		if meetingSessions[info.meetingKey] > 1 {
			// Sprint weekend — disambiguate.
			switch strings.ToLower(info.sessionType) {
			case "sprint":
				labels[i] = circuit + " Sprint"
			default:
				labels[i] = circuit + " Race"
			}
		} else {
			// Non-sprint weekend — just the circuit name.
			labels[i] = circuit
		}
	}

	return labels
}

// shortenCircuitLabel trims common verbose location names to short forms.
func shortenCircuitLabel(name string) string {
	replacements := map[string]string{
		"United Arab Emirates": "Abu Dhabi",
		"United States":        "USA",
		"United Kingdom":       "Great Britain",
		"Saudi Arabia":         "Saudi Arabia",
	}
	if short, ok := replacements[name]; ok {
		return short
	}
	return name
}

// GetDriverProgression returns per-round cumulative points for each driver.
func (s *Service) GetDriverProgression(ctx context.Context, season int) (*DriversProgressionResponse, error) {
	snapshots, err := s.championshipRepo.GetDriverChampionshipSnapshots(ctx, season)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	// Only include snapshots for sessions that have actually ended (time-based).
	// This prevents phantom rounds from appearing in the progression chart
	// without depending on the finalized flag (which requires schema_version
	// alignment that may not hold for older sessions).
	completedKeys, err := s.sessionRepo.GetCompletedRaceSessionKeys(ctx, season, now)
	if err != nil {
		return nil, err
	}
	filtered := make([]storage.DriverChampionshipSnapshot, 0, len(snapshots))
	for _, snap := range snapshots {
		if _, ok := completedKeys[snap.SessionKey]; ok {
			filtered = append(filtered, snap)
		}
	}
	snapshots = filtered

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

	rounds := s.buildRoundLabels(ctx, season, sessionKeys, now)
	skIndex := make(map[int]int, len(sessionKeys))
	for i, sk := range sessionKeys {
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

	now := time.Now().UTC()

	// Only include snapshots for sessions that have actually ended (time-based).
	completedKeys, err := s.sessionRepo.GetCompletedRaceSessionKeys(ctx, season, now)
	if err != nil {
		return nil, err
	}
	filtered := make([]storage.TeamChampionshipSnapshot, 0, len(snapshots))
	for _, snap := range snapshots {
		if _, ok := completedKeys[snap.SessionKey]; ok {
			filtered = append(filtered, snap)
		}
	}
	snapshots = filtered

	sessionKeySet := make(map[int]struct{})
	for _, snap := range snapshots {
		sessionKeySet[snap.SessionKey] = struct{}{}
	}
	sessionKeys := make([]int, 0, len(sessionKeySet))
	for sk := range sessionKeySet {
		sessionKeys = append(sessionKeys, sk)
	}
	sort.Ints(sessionKeys)

	rounds := s.buildRoundLabels(ctx, season, sessionKeys, now)
	skIndex := make(map[int]int, len(sessionKeys))
	for i, sk := range sessionKeys {
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
			TeamColor:     domain.GetTeamColor(td.teamName),
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
			TeamName: team1, TeamColor: domain.GetTeamColor(team1), Points: latestPoints[0],
			Wins: ts1.Wins, Podiums: ts1.Podiums, DNFs: ts1.DNFs,
		},
		Team2: ComparisonTeamStats{
			TeamName: team2, TeamColor: domain.GetTeamColor(team2), Points: latestPoints[1],
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

// GetConstructorDriverBreakdown returns the point breakdown for a team's drivers.
func (s *Service) GetConstructorDriverBreakdown(ctx context.Context, season int, teamName string) (*ConstructorBreakdownResponse, error) {
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

	// Find drivers belonging to this team.
	teamDrivers := make(map[int]struct{})
	for driverNum, id := range identities {
		if id.TeamName == teamName {
			teamDrivers[driverNum] = struct{}{}
		}
	}

	// Get latest snapshot per driver (for this team only).
	latestByDriver := make(map[int]storage.DriverChampionshipSnapshot)
	for _, snap := range snapshots {
		if _, ok := teamDrivers[snap.DriverNumber]; !ok {
			continue
		}
		existing, exists := latestByDriver[snap.DriverNumber]
		if !exists || snap.SessionKey > existing.SessionKey {
			latestByDriver[snap.DriverNumber] = snap
		}
	}

	// Calculate team total points.
	var teamPoints int
	for _, snap := range latestByDriver {
		teamPoints += int(snap.PointsCurrent)
	}

	// Build entries.
	entries := make([]ConstructorDriverEntry, 0, len(latestByDriver))
	for driverNum, snap := range latestByDriver {
		id := identities[driverNum]
		stats := driverStats[driverNum]
		pct := 0.0
		if teamPoints > 0 {
			pct = float64(int(snap.PointsCurrent)) / float64(teamPoints) * 100
		}
		entries = append(entries, ConstructorDriverEntry{
			DriverNumber:     driverNum,
			DriverName:       id.DriverName,
			Position:         snap.PositionCurrent,
			Points:           int(snap.PointsCurrent),
			Wins:             stats.Wins,
			Podiums:          stats.Podiums,
			PointsPercentage: pct,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Points > entries[j].Points
	})

	return &ConstructorBreakdownResponse{
		Year:       season,
		TeamName:   teamName,
		TeamPoints: teamPoints,
		Drivers:    entries,
	}, nil
}
