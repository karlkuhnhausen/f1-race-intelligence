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
		// Apply cancellation overrides.
		if override, ok := domain.IsCancelled(m.Season, m.RaceName); ok {
			m.IsCancelled = true
			m.Status = string(domain.StatusCancelled)
			m.CancelledLabel = override.Label
			m.CancelledReason = override.Reason
		}

		// Derive lifecycle status at read time so past meetings flip to
		// "completed" without requiring an ingest cycle. The stored Status is
		// effectively a cache; dates + the wall clock are the source of truth.
		// Cancellation override (set above) always wins.
		if !m.IsCancelled {
			m.Status = deriveMeetingStatus(now, m.StartDatetimeUTC, m.EndDatetimeUTC)
		}

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

// deriveMeetingStatus computes a meeting's lifecycle status from the current
// time and its scheduled start/end times. Mirrors the day-12 session-level
// pattern: status is a derived value, not a stored fact.
//
// Rules:
//   - zero start time           -> unknown
//   - start is in the future    -> scheduled
//   - end is zero or in future  -> scheduled (race weekend in progress)
//   - else                      -> completed
//
// Cancellations are handled by the caller and take precedence.
func deriveMeetingStatus(now, start, end time.Time) string {
	if start.IsZero() {
		return string(domain.StatusUnknown)
	}
	if start.After(now) {
		return string(domain.StatusScheduled)
	}
	if end.IsZero() || end.After(now) {
		return string(domain.StatusScheduled)
	}
	return string(domain.StatusCompleted)
}
