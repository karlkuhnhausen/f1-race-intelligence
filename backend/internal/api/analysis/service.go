package analysis

import (
	"context"
	"log/slog"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/domain"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// Service handles business logic for the session analysis API.
type Service struct {
	repo         storage.AnalysisRepository
	calendarRepo storage.CalendarRepository
	logger       *slog.Logger
}

// NewService creates a new analysis service.
func NewService(repo storage.AnalysisRepository, logger *slog.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

// NewServiceWithCalendar creates an analysis service with calendar access for
// meeting_key-based queries.
func NewServiceWithCalendar(repo storage.AnalysisRepository, calendarRepo storage.CalendarRepository, logger *slog.Logger) *Service {
	return &Service{repo: repo, calendarRepo: calendarRepo, logger: logger}
}

// GetSessionAnalysis retrieves and maps analysis data for a given session.
// Returns nil if no data is available.
func (s *Service) GetSessionAnalysis(ctx context.Context, season, round int, sessionType string) (*SessionAnalysisDTO, error) {
	var data *storage.SessionAnalysisData
	var err error

	// Try meeting_key-based query first if calendar repo is available.
	if s.calendarRepo != nil {
		meetingKey := s.resolveMeetingKey(ctx, season, round)
		if meetingKey != 0 {
			data, err = s.repo.GetSessionAnalysisByMeetingKey(ctx, season, meetingKey, sessionType)
			if err != nil {
				return nil, err
			}
		}
	}

	// Fallback to round-based query.
	if data == nil {
		data, err = s.repo.GetSessionAnalysis(ctx, season, round, sessionType)
		if err != nil {
			return nil, err
		}
	}

	if data == nil {
		return nil, nil
	}

	dto := &SessionAnalysisDTO{
		Year:        season,
		Round:       round,
		SessionType: sessionType,
	}

	// Map positions
	for _, p := range data.Positions {
		laps := make([]PositionLapDTO, len(p.Laps))
		for i, l := range p.Laps {
			laps[i] = PositionLapDTO{Lap: l.LapNumber, Position: l.Position}
		}
		dto.Positions = append(dto.Positions, PositionDriverDTO{
			DriverNumber:  p.DriverNumber,
			DriverName:    p.DriverName,
			DriverAcronym: p.DriverAcronym,
			TeamName:      p.TeamName,
			TeamColor:     p.TeamColor,
			Laps:          laps,
		})
		// Track max laps for total_laps field
		for _, l := range p.Laps {
			if l.LapNumber > dto.TotalLaps {
				dto.TotalLaps = l.LapNumber
			}
		}
	}

	// Map intervals
	// NOTE: Intervals are NOT included in total_laps computation because OpenF1
	// sometimes reports interval data beyond actual race distance.
	for _, iv := range data.Intervals {
		laps := make([]IntervalLapDTO, len(iv.Laps))
		for i, l := range iv.Laps {
			laps[i] = IntervalLapDTO{Lap: l.LapNumber, GapToLeader: l.GapToLeader, Interval: l.Interval}
		}
		dto.Intervals = append(dto.Intervals, IntervalDriverDTO{
			DriverNumber:  iv.DriverNumber,
			DriverAcronym: iv.DriverAcronym,
			TeamName:      iv.TeamName,
			TeamColor:     iv.TeamColor,
			Laps:          laps,
		})
	}

	// Map stints
	for _, s := range data.Stints {
		dto.Stints = append(dto.Stints, StintDTO{
			DriverNumber:   s.DriverNumber,
			DriverAcronym:  s.DriverAcronym,
			TeamName:       s.TeamName,
			StintNumber:    s.StintNumber,
			Compound:       s.Compound,
			LapStart:       s.LapStart,
			LapEnd:         s.LapEnd,
			TireAgeAtStart: s.TireAgeAtStart,
		})
		if s.LapEnd > dto.TotalLaps {
			dto.TotalLaps = s.LapEnd
		}
	}

	// Map pits
	for _, p := range data.Pits {
		dto.Pits = append(dto.Pits, PitDTO{
			DriverNumber:  p.DriverNumber,
			DriverAcronym: p.DriverAcronym,
			TeamName:      p.TeamName,
			Lap:           p.LapNumber,
			PitDuration:   p.PitDuration,
			StopDuration:  p.StopDuration,
		})
		if p.LapNumber > dto.TotalLaps {
			dto.TotalLaps = p.LapNumber
		}
	}

	// Map overtakes
	for _, o := range data.Overtakes {
		dto.Overtakes = append(dto.Overtakes, OvertakeDTO{
			OvertakingDriverNumber: o.OvertakingDriverNumber,
			OvertakingDriverName:   o.OvertakingDriverName,
			OvertakenDriverNumber:  o.OvertakenDriverNumber,
			OvertakenDriverName:    o.OvertakenDriverName,
			Lap:                    o.LapNumber,
			Position:               o.Position,
		})
	}

	return dto, nil
}

func (s *Service) resolveMeetingKey(ctx context.Context, season, round int) int {
	meetings, err := s.calendarRepo.GetMeetingsBySeason(ctx, season)
	if err != nil {
		s.logger.WarnContext(ctx, "failed to get meetings for meeting_key resolution", "error", err)
		return 0
	}
	indexInputs := make([]domain.MeetingForIndex, 0, len(meetings))
	for _, m := range meetings {
		indexInputs = append(indexInputs, domain.MeetingForIndex{
			MeetingKey:       m.MeetingKey,
			RaceName:         m.RaceName,
			StartDatetimeUTC: m.StartDatetimeUTC,
			IsCancelled:      m.IsCancelled,
		})
	}
	idx := domain.BuildMeetingIndex(indexInputs)
	return idx.MeetingKeyForRound(round)
}
