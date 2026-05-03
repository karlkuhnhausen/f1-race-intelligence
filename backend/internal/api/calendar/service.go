package calendar

import (
	"context"
	"sort"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/domain"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/ingest"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// Service provides calendar business logic and response shaping.
type Service struct {
	repo          storage.CalendarRepository
	sessionRepo   storage.SessionRepository   // optional; nil disables weekend-active-session enrichment + podium
	standingsRepo storage.StandingsRepository // optional; nil disables podium season-points join
	now           func() time.Time            // injectable clock for testing
}

// NewService creates a new calendar service.
func NewService(repo storage.CalendarRepository) *Service {
	return &Service{repo: repo, now: func() time.Time { return time.Now().UTC() }}
}

// NewServiceWithClock creates a calendar service with an injectable clock.
func NewServiceWithClock(repo storage.CalendarRepository, now func() time.Time) *Service {
	return &Service{repo: repo, now: now}
}

// NewServiceWithSessions creates a calendar service that also surfaces the
// active session for an in-progress race weekend.
func NewServiceWithSessions(repo storage.CalendarRepository, sessionRepo storage.SessionRepository) *Service {
	return &Service{repo: repo, sessionRepo: sessionRepo, now: func() time.Time { return time.Now().UTC() }}
}

// NewServiceWithSessionsAndClock is like NewServiceWithSessions with an
// injectable clock for deterministic testing.
func NewServiceWithSessionsAndClock(repo storage.CalendarRepository, sessionRepo storage.SessionRepository, now func() time.Time) *Service {
	return &Service{repo: repo, sessionRepo: sessionRepo, now: now}
}

// WithStandings attaches a standings repository to enable podium enrichment
// (season-points join for top-3 finishers on completed races). Returns the
// receiver for fluent chaining.
func (s *Service) WithStandings(standingsRepo storage.StandingsRepository) *Service {
	s.standingsRepo = standingsRepo
	return s
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
		// Defensive read-side filter: skip pre-season testing meetings that
		// may have been written by older ingest versions before testing was
		// excluded from round numbering. Round 1 is the first race.
		if ingest.IsPreSeasonTesting(m.RaceName) {
			continue
		}

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

	// Podium enrichment: for completed rounds, attach top-3 race finishers
	// joined with current-season championship points. Best-effort — empty
	// podium silently if repos are not wired or queries fail.
	s.enrichPodiums(ctx, season, rounds)

	// Delegate next-race computation to the domain layer.
	nextResult := domain.SelectNextRace(domainMeetings, now)

	var countdownTarget *time.Time
	if nextResult.Found {
		t := nextResult.CountdownTarget
		countdownTarget = &t
	}

	resp := &CalendarResponse{
		Year:               season,
		DataAsOfUTC:        latestDataAsOf,
		NextRound:          nextResult.Round,
		CountdownTargetUTC: countdownTarget,
		Rounds:             rounds,
	}

	// Race-weekend enrichment: if the next round's weekend window contains
	// `now`, surface the currently-active session so the frontend can render
	// "Race Weekend" context instead of a multi-day countdown to the wrong
	// race. Best-effort — silently skipped if no session repo is wired or if
	// the session lookup fails.
	if nextResult.Found && s.sessionRepo != nil {
		if nextMeeting, ok := findMeeting(domainMeetings, nextResult.Round); ok {
			weekendEnd := nextMeeting.EndDatetimeUTC
			if weekendEnd.IsZero() && !nextMeeting.StartDatetimeUTC.IsZero() {
				weekendEnd = nextMeeting.StartDatetimeUTC.Add(72 * time.Hour)
			}
			weekendStart := nextMeeting.StartDatetimeUTC
			inWeekend := !weekendStart.IsZero() &&
				!now.Before(weekendStart) &&
				(weekendEnd.IsZero() || now.Before(weekendEnd))

			if inWeekend {
				resp.WeekendInProgress = true

				sessions, err := s.sessionRepo.GetSessionsByRound(ctx, season, nextResult.Round)
				if err == nil && len(sessions) > 0 {
					windows := make([]domain.SessionWindow, 0, len(sessions))
					for _, sess := range sessions {
						windows = append(windows, domain.SessionWindow{
							SessionType:  domain.SessionType(sess.SessionType),
							SessionName:  sess.SessionName,
							DateStartUTC: sess.DateStartUTC,
							DateEndUTC:   sess.DateEndUTC,
						})
					}
					if active, ok := domain.SelectActiveSession(windows, now); ok {
						resp.ActiveSession = &ActiveSessionDTO{
							SessionType: string(active.SessionType),
							SessionName: active.SessionName,
							Status:      string(active.Status),
							DateStart:   active.DateStart,
							DateEnd:     active.DateEnd,
						}
						// Prefer a session-scoped countdown target during the
						// weekend: count down to the next session start, or
						// to the active session's end while it is live.
						switch active.Status {
						case domain.SessionStatusUpcoming:
							t := active.DateStart
							resp.CountdownTargetUTC = &t
						case domain.SessionStatusInProgress:
							if !active.DateEnd.IsZero() {
								t := active.DateEnd
								resp.CountdownTargetUTC = &t
							}
						}
					}
				}
			}
		}
	}

	return resp, nil
}

// findMeeting returns the first meeting matching the given round number.
func findMeeting(meetings []domain.RaceMeeting, round int) (domain.RaceMeeting, bool) {
	for _, m := range meetings {
		if m.Round == round {
			return m, true
		}
	}
	return domain.RaceMeeting{}, false
}

// enrichPodiums attaches top-3 race finishers (with current-season points)
// to every completed round in `rounds`. Mutates rounds in-place. Silently
// skips rounds where the session repo is unwired, results aren't yet
// available, or queries fail — podium is a best-effort enhancement.
func (s *Service) enrichPodiums(ctx context.Context, season int, rounds []RoundDTO) {
	if s.sessionRepo == nil {
		return
	}

	// Fetch standings once for the season and index by driver name. Driver
	// name is the only stable join key — DriverStandingRow has no
	// driver_number. If standings aren't available, season points fall
	// back to 0.
	pointsByDriver := map[string]float64{}
	if s.standingsRepo != nil {
		standings, err := s.standingsRepo.GetDriverStandings(ctx, season)
		if err == nil {
			for _, row := range standings {
				pointsByDriver[row.DriverName] = row.Points
			}
		}
	}

	for i := range rounds {
		r := &rounds[i]
		if r.IsCancelled || r.Status != string(domain.StatusCompleted) {
			continue
		}

		results, err := s.sessionRepo.GetSessionResultsByRound(ctx, season, r.Round)
		if err != nil || len(results) == 0 {
			continue
		}

		// Filter to race-type results (race, sprint), then take top 3 by
		// position. Prefer the main "race" session over sprint when both
		// exist for a weekend.
		raceResults := make([]storage.SessionResult, 0, len(results))
		for _, sr := range results {
			if sr.SessionType == "race" && sr.Position >= 1 {
				raceResults = append(raceResults, sr)
			}
		}
		if len(raceResults) == 0 {
			continue
		}

		sort.SliceStable(raceResults, func(a, b int) bool {
			return raceResults[a].Position < raceResults[b].Position
		})

		limit := 3
		if len(raceResults) < limit {
			limit = len(raceResults)
		}
		podium := make([]PodiumEntryDTO, 0, limit)
		for j := 0; j < limit; j++ {
			sr := raceResults[j]
			podium = append(podium, PodiumEntryDTO{
				Position:      sr.Position,
				DriverNumber:  sr.DriverNumber,
				DriverAcronym: sr.DriverAcronym,
				DriverName:    sr.DriverName,
				TeamName:      sr.TeamName,
				SeasonPoints:  pointsByDriver[sr.DriverName],
			})
		}
		r.Podium = podium
	}
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
