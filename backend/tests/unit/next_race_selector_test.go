package unit

import (
	"testing"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/domain"
)

func TestSelectNextRace_SkipsCancelled(t *testing.T) {
	now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	meetings := []domain.RaceMeeting{
		{Round: 1, RaceName: "Bahrain GP", StartDatetimeUTC: time.Date(2026, 4, 5, 15, 0, 0, 0, time.UTC), IsCancelled: true, Status: domain.StatusCancelled},
		{Round: 2, RaceName: "Saudi Arabia GP", StartDatetimeUTC: time.Date(2026, 4, 12, 17, 0, 0, 0, time.UTC), IsCancelled: true, Status: domain.StatusCancelled},
		{Round: 3, RaceName: "Miami GP", StartDatetimeUTC: time.Date(2026, 5, 4, 19, 0, 0, 0, time.UTC), IsCancelled: false, Status: domain.StatusScheduled},
	}

	result := domain.SelectNextRace(meetings, now)
	if !result.Found {
		t.Fatal("expected to find next race")
	}
	if result.Round != 3 {
		t.Errorf("expected round 3, got %d", result.Round)
	}
	if result.CountdownTarget != meetings[2].StartDatetimeUTC {
		t.Errorf("expected countdown target %v, got %v", meetings[2].StartDatetimeUTC, result.CountdownTarget)
	}
}

func TestSelectNextRace_SkipsPastRounds(t *testing.T) {
	now := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	meetings := []domain.RaceMeeting{
		{Round: 1, RaceName: "Australian GP", StartDatetimeUTC: time.Date(2026, 3, 15, 5, 0, 0, 0, time.UTC), Status: domain.StatusCompleted},
		{Round: 2, RaceName: "Chinese GP", StartDatetimeUTC: time.Date(2026, 3, 29, 7, 0, 0, 0, time.UTC), Status: domain.StatusCompleted},
		{Round: 3, RaceName: "Japanese GP", StartDatetimeUTC: time.Date(2026, 4, 12, 6, 0, 0, 0, time.UTC), Status: domain.StatusScheduled},
	}

	result := domain.SelectNextRace(meetings, now)
	if !result.Found {
		t.Fatal("expected to find next race")
	}
	if result.Round != 3 {
		t.Errorf("expected round 3, got %d", result.Round)
	}
}

func TestSelectNextRace_TieBreakLowerRoundWins(t *testing.T) {
	now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	sameTime := time.Date(2026, 5, 10, 14, 0, 0, 0, time.UTC)
	meetings := []domain.RaceMeeting{
		{Round: 5, RaceName: "Sprint A", StartDatetimeUTC: sameTime, Status: domain.StatusScheduled},
		{Round: 6, RaceName: "Sprint B", StartDatetimeUTC: sameTime, Status: domain.StatusScheduled},
	}

	result := domain.SelectNextRace(meetings, now)
	if !result.Found {
		t.Fatal("expected to find next race")
	}
	if result.Round != 5 {
		t.Errorf("tie-break: expected round 5 (lower), got %d", result.Round)
	}
}

func TestSelectNextRace_NoFutureRounds(t *testing.T) {
	now := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	meetings := []domain.RaceMeeting{
		{Round: 1, RaceName: "Australian GP", StartDatetimeUTC: time.Date(2026, 3, 15, 5, 0, 0, 0, time.UTC), Status: domain.StatusCompleted},
		{Round: 2, RaceName: "Chinese GP", StartDatetimeUTC: time.Date(2026, 3, 29, 7, 0, 0, 0, time.UTC), Status: domain.StatusCompleted},
	}

	result := domain.SelectNextRace(meetings, now)
	if result.Found {
		t.Error("expected no next race when all rounds are past")
	}
	if result.Round != 0 {
		t.Errorf("expected round 0 for no result, got %d", result.Round)
	}
}

func TestSelectNextRace_EmptySlice(t *testing.T) {
	now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	result := domain.SelectNextRace(nil, now)
	if result.Found {
		t.Error("expected no next race for empty input")
	}
}

func TestSelectNextRace_StatusCancelledWithoutFlag(t *testing.T) {
	// A round with StatusCancelled but IsCancelled=false should still be skipped.
	now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	meetings := []domain.RaceMeeting{
		{Round: 1, RaceName: "Bahrain GP", StartDatetimeUTC: time.Date(2026, 4, 5, 15, 0, 0, 0, time.UTC), IsCancelled: false, Status: domain.StatusCancelled},
		{Round: 2, RaceName: "Miami GP", StartDatetimeUTC: time.Date(2026, 5, 4, 19, 0, 0, 0, time.UTC), IsCancelled: false, Status: domain.StatusScheduled},
	}

	result := domain.SelectNextRace(meetings, now)
	if !result.Found {
		t.Fatal("expected to find next race")
	}
	if result.Round != 2 {
		t.Errorf("expected round 2, got %d", result.Round)
	}
}
