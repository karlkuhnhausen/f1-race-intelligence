package standings

import (
	"context"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// Service provides championship standings data.
type Service struct {
	repo storage.StandingsRepository
}

// NewService creates a standings service.
func NewService(repo storage.StandingsRepository) *Service {
	return &Service{repo: repo}
}

// GetDrivers returns the drivers championship for the given season.
func (s *Service) GetDrivers(ctx context.Context, season int) (*DriversStandingsResponse, error) {
	rows, err := s.repo.GetDriverStandings(ctx, season)
	if err != nil {
		return nil, err
	}

	var latestDataAsOf time.Time
	dtos := make([]DriverStandingDTO, 0, len(rows))
	for _, r := range rows {
		dtos = append(dtos, DriverStandingDTO{
			Position:   r.Position,
			DriverName: r.DriverName,
			TeamName:   r.TeamName,
			Points:     r.Points,
			Wins:       r.Wins,
		})
		if r.DataAsOfUTC.After(latestDataAsOf) {
			latestDataAsOf = r.DataAsOfUTC
		}
	}

	return &DriversStandingsResponse{
		Year:        season,
		DataAsOfUTC: latestDataAsOf,
		Rows:        dtos,
	}, nil
}

// GetConstructors returns the constructors championship for the given season.
func (s *Service) GetConstructors(ctx context.Context, season int) (*ConstructorsStandingsResponse, error) {
	rows, err := s.repo.GetConstructorStandings(ctx, season)
	if err != nil {
		return nil, err
	}

	var latestDataAsOf time.Time
	dtos := make([]ConstructorStandingDTO, 0, len(rows))
	for _, r := range rows {
		dtos = append(dtos, ConstructorStandingDTO{
			Position: r.Position,
			TeamName: r.TeamName,
			Points:   r.Points,
		})
		if r.DataAsOfUTC.After(latestDataAsOf) {
			latestDataAsOf = r.DataAsOfUTC
		}
	}

	return &ConstructorsStandingsResponse{
		Year:        season,
		DataAsOfUTC: latestDataAsOf,
		Rows:        dtos,
	}, nil
}
