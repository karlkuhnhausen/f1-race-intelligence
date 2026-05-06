package rounds

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/domain"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// Session lifecycle status values returned to clients. These align with the
// Feature 003 data-model.md status enum.
const (
	statusUpcoming   = "upcoming"
	statusInProgress = "in_progress"
	statusCompleted  = "completed"
)

// RaceControlHydrator fetches and persists race control data on demand.
// Injected into Service; may be nil (lazy fill skipped gracefully).
type RaceControlHydrator interface {
	Hydrate(ctx context.Context, sess storage.Session) (*storage.RaceControlSummary, error)
}

// Service provides round detail business logic and response shaping.
type Service struct {
	sessionRepo  storage.SessionRepository
	calendarRepo storage.CalendarRepository
	rcHydrator   RaceControlHydrator // nil-safe: lazy fill skipped when nil
	now          func() time.Time    // injectable clock for testing
	logger       *slog.Logger
}

// NewService creates a new rounds service with no hydrator.
func NewService(sessionRepo storage.SessionRepository, calendarRepo storage.CalendarRepository) *Service {
	return &Service{
		sessionRepo:  sessionRepo,
		calendarRepo: calendarRepo,
		now:          func() time.Time { return time.Now().UTC() },
		logger:       slog.Default(),
	}
}

// NewServiceWithClock creates a rounds service with an injectable clock,
// primarily for deterministic testing of session status derivation.
func NewServiceWithClock(sessionRepo storage.SessionRepository, calendarRepo storage.CalendarRepository, now func() time.Time) *Service {
	return &Service{sessionRepo: sessionRepo, calendarRepo: calendarRepo, now: now, logger: slog.Default()}
}

// NewServiceWithHydrator creates a rounds service with a RaceControlHydrator for
// lazy-on-read gap fill. Pass nil for hydrator to disable lazy fill (e.g., in tests).
func NewServiceWithHydrator(
	sessionRepo storage.SessionRepository,
	calendarRepo storage.CalendarRepository,
	hydrator RaceControlHydrator,
	logger *slog.Logger,
) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		sessionRepo:  sessionRepo,
		calendarRepo: calendarRepo,
		rcHydrator:   hydrator,
		now:          func() time.Time { return time.Now().UTC() },
		logger:       logger,
	}
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

// deriveTopEvent returns the highest-priority notable event from a RaceControlSummary,
// or nil if no events occurred. Priority: red_flag > safety_car > vsc > investigation.
func deriveTopEvent(rc *storage.RaceControlSummary) *NotableEventDTO {
	if rc == nil || len(rc.NotableEvents) == 0 {
		return nil
	}
	// NotableEvents are already ordered by priority from SummarizeRaceControl.
	e := rc.NotableEvents[0]
	return &NotableEventDTO{
		EventType: e.EventType,
		LapNumber: e.LapNumber,
		Count:     e.Count,
	}
}

// deriveRecapSummary computes the session recap payload from stored session data
// and result DTOs. Returns nil if the session type is unrecognized or results empty.
// Handles nil RaceControlSummary gracefully — event fields are omitted.
func deriveRecapSummary(sess storage.Session, results []SessionResultDTO) *SessionRecapDTO {
	st := domain.SessionType(sess.SessionType)
	switch {
	case domain.IsRaceType(st):
		return deriveRaceRecap(sess, results)
	case domain.IsQualifyingType(st):
		return deriveQualifyingRecap(sess, results)
	case domain.IsPracticeType(st):
		return derivePracticeRecap(sess, results)
	}
	return nil
}

// deriveRaceRecap builds the recap for race and sprint sessions.
func deriveRaceRecap(sess storage.Session, results []SessionResultDTO) *SessionRecapDTO {
	if len(results) == 0 {
		return nil
	}

	var p1, p2 *SessionResultDTO
	for i := range results {
		if results[i].Position == 1 && p1 == nil {
			p1 = &results[i]
		} else if results[i].Position == 2 && p2 == nil {
			p2 = &results[i]
		}
		if p1 != nil && p2 != nil {
			break
		}
	}
	if p1 == nil {
		return nil
	}

	recap := &SessionRecapDTO{
		WinnerName:            p1.DriverName,
		WinnerTeam:            p1.TeamName,
		TotalLaps:             p1.NumberOfLaps,
		FastestLapTimeSeconds: sess.FastestLapTimeSeconds,
	}

	// Gap to P2: use P2's GapToLeader string directly.
	if p2 != nil && p2.GapToLeader != nil {
		recap.GapToP2 = *p2.GapToLeader
	}

	// Fastest lap holder.
	for i := range results {
		if results[i].FastestLap != nil && *results[i].FastestLap {
			recap.FastestLapHolder = results[i].DriverName
			recap.FastestLapTeam = results[i].TeamName
			break
		}
	}

	// Race control fields.
	if sess.RaceControlSummary != nil {
		recap.RedFlagCount = sess.RaceControlSummary.RedFlagCount
		recap.SafetyCarCount = sess.RaceControlSummary.SafetyCarCount
		recap.VSCCount = sess.RaceControlSummary.VSCCount
	}
	recap.TopEvent = deriveTopEvent(sess.RaceControlSummary)

	return recap
}

// deriveQualifyingRecap builds the recap for qualifying and sprint qualifying sessions.
func deriveQualifyingRecap(sess storage.Session, results []SessionResultDTO) *SessionRecapDTO {
	if len(results) == 0 {
		return nil
	}

	var p1, p2 *SessionResultDTO
	for i := range results {
		if results[i].Position == 1 && p1 == nil {
			p1 = &results[i]
		} else if results[i].Position == 2 && p2 == nil {
			p2 = &results[i]
		}
		if p1 != nil && p2 != nil {
			break
		}
	}
	if p1 == nil {
		return nil
	}

	// Pole time: Q3 preferred, then Q2, then Q1.
	var poleTime *float64
	if p1.Q3Time != nil {
		poleTime = p1.Q3Time
	} else if p1.Q2Time != nil {
		poleTime = p1.Q2Time
	} else {
		poleTime = p1.Q1Time
	}

	recap := &SessionRecapDTO{
		PoleSitterName: p1.DriverName,
		PoleSitterTeam: p1.TeamName,
		PoleTime:       poleTime,
	}

	// Gap to P2: formatted time delta in seconds (P2 time − pole time).
	if p2 != nil && poleTime != nil {
		var p2Time *float64
		if p2.Q3Time != nil {
			p2Time = p2.Q3Time
		} else if p2.Q2Time != nil {
			p2Time = p2.Q2Time
		} else {
			p2Time = p2.Q1Time
		}
		if p2Time != nil {
			recap.GapToP2 = fmt.Sprintf("+%.3f", *p2Time-*poleTime)
		}
	}

	// Q1 cutoff: last result with Q1Time but no Q2Time (eliminated in Q1).
	for i := len(results) - 1; i >= 0; i-- {
		if results[i].Q1Time != nil && results[i].Q2Time == nil {
			recap.Q1CutoffTime = results[i].Q1Time
			break
		}
	}

	// Q2 cutoff: last result with Q2Time but no Q3Time (eliminated in Q2).
	for i := len(results) - 1; i >= 0; i-- {
		if results[i].Q2Time != nil && results[i].Q3Time == nil {
			recap.Q2CutoffTime = results[i].Q2Time
			break
		}
	}

	// Race control fields.
	if sess.RaceControlSummary != nil {
		recap.RedFlagCount = sess.RaceControlSummary.RedFlagCount
		recap.SafetyCarCount = sess.RaceControlSummary.SafetyCarCount
		recap.VSCCount = sess.RaceControlSummary.VSCCount
	}
	recap.TopEvent = deriveTopEvent(sess.RaceControlSummary)

	return recap
}

// derivePracticeRecap builds the recap for practice sessions (FP1/FP2/FP3).
func derivePracticeRecap(sess storage.Session, results []SessionResultDTO) *SessionRecapDTO {
	if len(results) == 0 {
		return nil
	}

	// Best driver: result with Position == 1 (results already sorted ascending by position).
	var best *SessionResultDTO
	for i := range results {
		if results[i].Position == 1 {
			best = &results[i]
			break
		}
	}
	if best == nil {
		best = &results[0]
	}

	// Total laps: sum across all drivers.
	totalLaps := 0
	for _, r := range results {
		totalLaps += r.NumberOfLaps
	}

	recap := &SessionRecapDTO{
		BestDriverName: best.DriverName,
		BestDriverTeam: best.TeamName,
		BestLapTime:    best.BestLapTime,
		TotalLaps:      totalLaps,
	}

	// Race control fields.
	if sess.RaceControlSummary != nil {
		recap.RedFlagCount = sess.RaceControlSummary.RedFlagCount
		recap.SafetyCarCount = sess.RaceControlSummary.SafetyCarCount
		recap.VSCCount = sess.RaceControlSummary.VSCCount
	}
	recap.TopEvent = deriveTopEvent(sess.RaceControlSummary)

	return recap
}

// GetRoundDetail retrieves session data and results for a specific round.
func (s *Service) GetRoundDetail(ctx context.Context, season, round int) (*RoundDetailResponse, error) {
	// Get meeting info for the round and build a MeetingIndex.
	meetings, err := s.calendarRepo.GetMeetingsBySeason(ctx, season)
	if err != nil {
		return nil, err
	}

	// Build MeetingIndex for round → meeting_key resolution.
	indexInputs := make([]domain.MeetingForIndex, 0, len(meetings))
	for _, m := range meetings {
		indexInputs = append(indexInputs, domain.MeetingForIndex{
			MeetingKey:       m.MeetingKey,
			RaceName:         m.RaceName,
			StartDatetimeUTC: m.StartDatetimeUTC,
			IsCancelled:      m.IsCancelled,
		})
	}
	meetingIdx := domain.BuildMeetingIndex(indexInputs)

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

	// Resolve round → meeting_key and prefer meeting_key-based queries.
	meetingKey := meetingIdx.MeetingKeyForRound(round)

	var sessions []storage.Session
	var results []storage.SessionResult

	if meetingKey != 0 {
		sessions, err = s.sessionRepo.GetSessionsByMeetingKey(ctx, season, meetingKey)
		if err != nil {
			return nil, err
		}
		results, err = s.sessionRepo.GetSessionResultsByMeetingKey(ctx, season, meetingKey)
		if err != nil {
			return nil, err
		}
	}

	// Fallback to round-based queries if meeting_key query returned nothing
	// (handles pre-migration data that doesn't have meeting_key populated).
	if len(sessions) == 0 {
		sessions, err = s.sessionRepo.GetSessionsByRound(ctx, season, round)
		if err != nil {
			return nil, err
		}
	}
	if len(results) == 0 {
		results, err = s.sessionRepo.GetSessionResultsByRound(ctx, season, round)
		if err != nil {
			return nil, err
		}
	}

	// Deduplicate sessions: the 008 migration can leave old documents with
	// shifted round numbers alongside freshly-polled documents for the same
	// session_type. Keep only the newest document per session_type.
	{
		bestByType := make(map[string]storage.Session, len(sessions))
		for _, sess := range sessions {
			if existing, ok := bestByType[sess.SessionType]; !ok || sess.DataAsOfUTC.After(existing.DataAsOfUTC) {
				bestByType[sess.SessionType] = sess
			}
		}
		sessions = make([]storage.Session, 0, len(bestByType))
		for _, sess := range bestByType {
			sessions = append(sessions, sess)
		}
		sort.Slice(sessions, func(i, j int) bool {
			return sessions[i].DateStartUTC.Before(sessions[j].DateStartUTC)
		})
	}

	// Deduplicate results: if the same (session_type, driver_number) appears
	// more than once, keep only the first occurrence. This guards against
	// duplicate documents in Cosmos caused by the 008 migration re-ingesting
	// data with shifted round numbers while meeting_key stayed constant.
	seen := make(map[string]struct{}, len(results))
	deduped := make([]storage.SessionResult, 0, len(results))
	for _, r := range results {
		key := fmt.Sprintf("%s:%d", r.SessionType, r.DriverNumber)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, r)
	}
	results = deduped

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

		status := deriveSessionStatus(now, sess.DateStartUTC, sess.DateEndUTC)

		// Only include results for sessions that have started. Upcoming
		// sessions may have misattributed data in Cosmos (post-008 migration
		// artifact) which should not be surfaced.
		var resultDTOs []SessionResultDTO
		if status != statusUpcoming {
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
			resultDTOs = make([]SessionResultDTO, 0, len(sessResults))
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
		}

		// T026: lazy fill — if completed, RaceControlSummary missing, and hydrator available,
		// fetch race control data and persist it before building the recap.
		if status == statusCompleted && sess.RaceControlSummary == nil && s.rcHydrator != nil {
			if summary, hydrateErr := s.rcHydrator.Hydrate(ctx, sess); hydrateErr != nil {
				s.logger.Warn("lazy race control fill failed — recap rendered without events",
					"session_id", sess.ID, "error", hydrateErr)
			} else {
				sess.RaceControlSummary = summary
			}
		}

		// Derive recap summary for completed sessions.
		var recapSummary *SessionRecapDTO
		if status == statusCompleted {
			recapSummary = deriveRecapSummary(sess, resultDTOs)
		}

		sessionDTOs = append(sessionDTOs, SessionDetailDTO{
			SessionName:  sess.SessionName,
			SessionType:  sess.SessionType,
			Status:       status,
			DateStart:    sess.DateStartUTC,
			DateEnd:      sess.DateEndUTC,
			Results:      resultDTOs,
			RecapSummary: recapSummary,
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
