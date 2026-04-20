package calendar

import (
	"context"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/domain"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// Service provides calendar business logic and response shaping.
type Service struct {
	repo storage.CalendarRepository
	now  func() time.Time // injectable clock for testing
}

// NewService creates a new calendar service.
func NewService(repo storage.CalendarRepository) *Service {
	return &Service{repo: repo, now: func() time.Time { return time.Now().UTC() }}
}

// NewServiceWithClock creates a calendar service with an injectable clock.
func NewServiceWithClock(repo storage.CalendarRepository, now func() time.Time) *Service {
	return &Service{repo: repo, now: now}
}

// GetCalendar retrieves the full season calendar and computes next-race metadata.
func (s *Service) GetCalendar(ctx context.Context, season int) (*CalendarResponse, error) {
	meetings, err := s.repo.GetMeetingsBySeason(ctx, season)
	if err != nil {
		return nil, err
	}

	now := s.now()
	rounds := make([]RoundDTO, 0, len(meetings))
	var latestDataAsOf time.Time

	// Convert storage meetings to domain meetings for next-race selection.
	domainMeetings := make([]domain.RaceMeeting, 0, len(meetings))
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

		domainMeetings = append(domainMeetings, domain.RaceMeeting{
			Round:            m.Round,
			RaceName:         m.RaceName,
			CircuitName:      m.CircuitName,
			CountryName:      m.CountryName,
			StartDatetimeUTC: m.StartDatetimeUTC,
			EndDatetimeUTC:   m.EndDatetimeUTC,
			Status:           domain.MeetingStatus(m.Status),
			IsCancelled:      m.IsCancelled,
			CancelledLabel:   m.CancelledLabel,
			CancelledReason:  m.CancelledReason,
		})
	}

	// Delegate next-race computation to the domain layer.
	nextResult := domain.SelectNextRace(domainMeetings, now)

	var countdownTarget *time.Time
	if nextResult.Found {
		t := nextResult.CountdownTarget
		countdownTarget = &t
	}

	return &CalendarResponse{
		Year:               season,
		DataAsOfUTC:        latestDataAsOf,
		NextRound:          nextResult.Round,
		CountdownTargetUTC: countdownTarget,
		Rounds:             rounds,
	}, nil
}
