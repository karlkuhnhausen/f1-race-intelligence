package calendar

import (
	"context"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// Service provides calendar business logic and response shaping.
type Service struct {
	repo storage.CalendarRepository
}

// NewService creates a new calendar service.
func NewService(repo storage.CalendarRepository) *Service {
	return &Service{repo: repo}
}

// GetCalendar retrieves the full season calendar and computes next-race metadata.
func (s *Service) GetCalendar(ctx context.Context, season int) (*CalendarResponse, error) {
	meetings, err := s.repo.GetMeetingsBySeason(ctx, season)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	rounds := make([]RoundDTO, 0, len(meetings))
	var latestDataAsOf time.Time
	var nextRound int
	var countdownTarget *time.Time

	for _, m := range meetings {
		rounds = append(rounds, RoundDTO{
			Round:            m.Round,
			RaceName:         m.RaceName,
			CircuitName:      m.CircuitName,
			CountryName:      m.CountryName,
			StartDatetimeUTC: m.StartDatetimeUTC,
			EndDatetimeUTC:   m.EndDatetimeUTC,
			Status:           m.Status,
			IsCancelled:      m.IsCancelled,
			CancelledLabel:   m.CancelledLabel,
			CancelledReason:  m.CancelledReason,
		})

		if m.DataAsOfUTC.After(latestDataAsOf) {
			latestDataAsOf = m.DataAsOfUTC
		}

		// Compute next round: first non-cancelled round with start >= now.
		if nextRound == 0 && !m.IsCancelled && m.StartDatetimeUTC.After(now) {
			nextRound = m.Round
			t := m.StartDatetimeUTC
			countdownTarget = &t
		}
	}

	return &CalendarResponse{
		Year:               season,
		DataAsOfUTC:        latestDataAsOf,
		NextRound:          nextRound,
		CountdownTargetUTC: countdownTarget,
		Rounds:             rounds,
	}, nil
}
