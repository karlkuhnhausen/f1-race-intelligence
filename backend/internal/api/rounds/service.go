package rounds

import (
	"context"
	"sort"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// Session lifecycle status values returned to clients. These align with the
// Feature 003 data-model.md status enum.
const (
	statusUpcoming   = "upcoming"
	statusInProgress = "in_progress"
	statusCompleted  = "completed"
)

// Service provides round detail business logic and response shaping.
type Service struct {
	sessionRepo  storage.SessionRepository
	calendarRepo storage.CalendarRepository
	now          func() time.Time // injectable clock for testing
}

// NewService creates a new rounds service.
func NewService(sessionRepo storage.SessionRepository, calendarRepo storage.CalendarRepository) *Service {
	return &Service{
		sessionRepo:  sessionRepo,
		calendarRepo: calendarRepo,
		now:          func() time.Time { return time.Now().UTC() },
	}
}

// NewServiceWithClock creates a rounds service with an injectable clock,
// primarily for deterministic testing of session status derivation.
func NewServiceWithClock(sessionRepo storage.SessionRepository, calendarRepo storage.CalendarRepository, now func() time.Time) *Service {
	return &Service{sessionRepo: sessionRepo, calendarRepo: calendarRepo, now: now}
}

// deriveSessionStatus returns the lifecycle status for a session based on its
// scheduled start/end times relative to now. Stored values are ignored because
// historical ingest writes hardcoded the status field (see session_transform.go).
//
// Rules:
//   - dateStart in the future → "upcoming"
//   - dateStart <= now < dateEnd → "in_progress"
//   - dateEnd <= now → "completed"
//   - Zero/missing dates → "upcoming" (safe default — avoids falsely marking
//     unscheduled sessions as completed)
func deriveSessionStatus(now, dateStart, dateEnd time.Time) string {
	if dateStart.IsZero() {
		return statusUpcoming
	}
	if dateStart.After(now) {
		return statusUpcoming
	}
	if dateEnd.IsZero() || dateEnd.After(now) {
		return statusInProgress
	}
	return statusCompleted
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
	now := s.now()

	for _, sess := range sessions {
		if sess.DataAsOfUTC.After(latestDataAsOf) {
			latestDataAsOf = sess.DataAsOfUTC
		}

		sessResults := resultsByType[sess.SessionType]
		// Sort ascending by position. OpenF1 occasionally uses position 0
		// as a placeholder for unclassified entries (DNF/DNS/DSQ); push
		// those to the bottom so the classified field starts at P1. The
		// frontend further splits classified vs non-classified using
		// finishing_status.
		sort.SliceStable(sessResults, func(i, j int) bool {
			pi, pj := sessResults[i].Position, sessResults[j].Position
			zi := pi <= 0
			zj := pj <= 0
			if zi != zj {
				return !zi // non-zero positions first
			}
			return pi < pj
		})
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
			Status:      deriveSessionStatus(now, sess.DateStartUTC, sess.DateEndUTC),
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
