package rounds

import (
	"context"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// Service provides round detail business logic and response shaping.
type Service struct {
	sessionRepo  storage.SessionRepository
	calendarRepo storage.CalendarRepository
}

// NewService creates a new rounds service.
func NewService(sessionRepo storage.SessionRepository, calendarRepo storage.CalendarRepository) *Service {
	return &Service{sessionRepo: sessionRepo, calendarRepo: calendarRepo}
}

// GetRoundDetail retrieves session data and results for a specific round.
func (s *Service) GetRoundDetail(ctx context.Context, season, round int) (*RoundDetailResponse, error) {
	// Get meeting info for the round
	meetings, err := s.calendarRepo.GetMeetingsBySeason(ctx, season)
	if err != nil {
		return nil, err
	}

	var meeting *storage.RaceMeeting
	for i := range meetings {
		if meetings[i].Round == round {
			meeting = &meetings[i]
			break
		}
	}

	raceName := ""
	circuitName := ""
	countryName := ""
	if meeting != nil {
		raceName = meeting.RaceName
		circuitName = meeting.CircuitName
		countryName = meeting.CountryName
	}

	// Get sessions for this round
	sessions, err := s.sessionRepo.GetSessionsByRound(ctx, season, round)
	if err != nil {
		return nil, err
	}

	// Get all results for this round
	results, err := s.sessionRepo.GetSessionResultsByRound(ctx, season, round)
	if err != nil {
		return nil, err
	}

	// Group results by session_type
	resultsByType := make(map[string][]storage.SessionResult)
	for _, r := range results {
		resultsByType[r.SessionType] = append(resultsByType[r.SessionType], r)
	}

	var latestDataAsOf time.Time
	sessionDTOs := make([]SessionDetailDTO, 0, len(sessions))

	for _, sess := range sessions {
		if sess.DataAsOfUTC.After(latestDataAsOf) {
			latestDataAsOf = sess.DataAsOfUTC
		}

		sessResults := resultsByType[sess.SessionType]
		resultDTOs := make([]SessionResultDTO, 0, len(sessResults))
		for _, r := range sessResults {
			resultDTOs = append(resultDTOs, SessionResultDTO{
				Position:        r.Position,
				DriverNumber:    r.DriverNumber,
				DriverName:      r.DriverName,
				DriverAcronym:   r.DriverAcronym,
				TeamName:        r.TeamName,
				NumberOfLaps:    r.NumberOfLaps,
				FinishingStatus: r.FinishingStatus,
				RaceTime:        r.RaceTime,
				GapToLeader:     r.GapToLeader,
				Points:          r.Points,
				FastestLap:      r.FastestLap,
				Q1Time:          r.Q1Time,
				Q2Time:          r.Q2Time,
				Q3Time:          r.Q3Time,
				BestLapTime:     r.BestLapTime,
				GapToFastest:    r.GapToFastest,
			})
		}

		sessionDTOs = append(sessionDTOs, SessionDetailDTO{
			SessionName: sess.SessionName,
			SessionType: sess.SessionType,
			Status:      sess.Status,
			DateStart:   sess.DateStartUTC,
			DateEnd:     sess.DateEndUTC,
			Results:     resultDTOs,
		})
	}

	return &RoundDetailResponse{
		Year:        season,
		Round:       round,
		RaceName:    raceName,
		CircuitName: circuitName,
		CountryName: countryName,
		DataAsOfUTC: latestDataAsOf,
		Sessions:    sessionDTOs,
	}, nil
}
