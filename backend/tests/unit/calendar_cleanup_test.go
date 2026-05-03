package unit

import (
	"context"
	"testing"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/api/calendar"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/ingest"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// fakeSessionRepoWithResults is a SessionRepository that returns canned
// session results keyed by round, used to test podium enrichment.
type fakeSessionRepoWithResults struct {
	resultsByRound map[int][]storage.SessionResult
}

func (f *fakeSessionRepoWithResults) UpsertSession(_ context.Context, _ storage.Session) error {
	return nil
}
func (f *fakeSessionRepoWithResults) UpsertSessionResult(_ context.Context, _ storage.SessionResult) error {
	return nil
}
func (f *fakeSessionRepoWithResults) GetSessionsByRound(_ context.Context, _, _ int) ([]storage.Session, error) {
	return nil, nil
}
func (f *fakeSessionRepoWithResults) GetSessionResultsByRound(_ context.Context, _, round int) ([]storage.SessionResult, error) {
	return f.resultsByRound[round], nil
}
func (f *fakeSessionRepoWithResults) GetSessionResultsBySeason(_ context.Context, _ int) ([]storage.SessionResult, error) {
	var all []storage.SessionResult
	for _, rs := range f.resultsByRound {
		all = append(all, rs...)
	}
	return all, nil
}
func (f *fakeSessionRepoWithResults) GetFinalizedSessionKeys(_ context.Context, _ int) (map[int]int, error) {
	return nil, nil
}

// TestIsPreSeasonTesting verifies the predicate matches typical OpenF1
// pre-season testing meeting names without false-positive matches against
// real grands prix.
func TestIsPreSeasonTesting(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"Pre-Season Testing", true},
		{"Pre-Season Test", true},
		{"PRE-SEASON TESTING", true},
		{"Pre Season Testing", true},
		{"Bahrain Pre-Season Testing", true},
		{"Australian Grand Prix", false},
		{"Bahrain Grand Prix", false},
		{"Las Vegas Grand Prix", false},
		{"São Paulo Grand Prix", false},
	}
	for _, tc := range cases {
		if got := ingest.IsPreSeasonTesting(tc.name); got != tc.want {
			t.Errorf("IsPreSeasonTesting(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestGetCalendar_FiltersTestingMeetings verifies that any pre-season
// testing meeting still present in storage is dropped from the calendar
// response so Round 1 in the UI is the first race.
func TestGetCalendar_FiltersTestingMeetings(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)

	meetings := []storage.RaceMeeting{
		// Legacy testing rows that older ingest may have written.
		{
			ID: "2026-99a", Season: 2026, Round: 1,
			RaceName:         "Pre-Season Testing",
			StartDatetimeUTC: time.Date(2026, 2, 26, 8, 0, 0, 0, time.UTC),
			EndDatetimeUTC:   time.Date(2026, 2, 28, 17, 0, 0, 0, time.UTC),
		},
		{
			ID: "2026-01", Season: 2026, Round: 1,
			RaceName:         "Australian Grand Prix",
			StartDatetimeUTC: time.Date(2026, 3, 13, 5, 0, 0, 0, time.UTC),
			EndDatetimeUTC:   time.Date(2026, 3, 16, 5, 0, 0, 0, time.UTC),
		},
		{
			ID: "2026-02", Season: 2026, Round: 2,
			RaceName:         "Chinese Grand Prix",
			StartDatetimeUTC: time.Date(2026, 3, 20, 5, 0, 0, 0, time.UTC),
			EndDatetimeUTC:   time.Date(2026, 3, 23, 9, 0, 0, 0, time.UTC),
		},
	}

	repo := &fakeCalendarRepo{meetings: meetings}
	svc := calendar.NewServiceWithClock(repo, func() time.Time { return now })

	resp, err := svc.GetCalendar(context.Background(), 2026)
	if err != nil {
		t.Fatalf("GetCalendar returned error: %v", err)
	}
	if len(resp.Rounds) != 2 {
		t.Fatalf("expected 2 rounds (testing filtered), got %d", len(resp.Rounds))
	}
	for _, r := range resp.Rounds {
		if r.RaceName == "Pre-Season Testing" {
			t.Errorf("calendar response should not contain pre-season testing, got %q", r.RaceName)
		}
	}
}

// TestNormalizeMeetings_SkipsTestingForRoundNumbering verifies the ingest
// path: when OpenF1 returns testing as the first meeting of the season,
// the first race becomes Round 1 (not Round 3).
func TestNormalizeMeetings_SkipsTestingForRoundNumbering(t *testing.T) {
	raw := []ingest.OpenF1MeetingForTest{
		{MeetingName: "Pre-Season Testing", DateStart: "2026-02-26T08:00:00Z", MeetingKey: 1},
		{MeetingName: "Australian Grand Prix", DateStart: "2026-03-13T05:00:00Z", MeetingKey: 2},
		{MeetingName: "Chinese Grand Prix", DateStart: "2026-03-20T05:00:00Z", MeetingKey: 3},
	}
	meetings := ingest.NormalizeMeetingsForTest(raw, 2026)
	if len(meetings) != 2 {
		t.Fatalf("expected 2 meetings (testing skipped), got %d", len(meetings))
	}
	if meetings[0].Round != 1 || meetings[0].RaceName != "Australian Grand Prix" {
		t.Errorf("Round 1 should be Australian GP, got round=%d name=%q", meetings[0].Round, meetings[0].RaceName)
	}
	if meetings[1].Round != 2 || meetings[1].RaceName != "Chinese Grand Prix" {
		t.Errorf("Round 2 should be Chinese GP, got round=%d name=%q", meetings[1].Round, meetings[1].RaceName)
	}
}

// TestGetCalendar_PodiumEnrichment verifies that completed races are
// enriched with top-3 race finishers + cumulative season points (summed
// from race + sprint session results), while upcoming races are not.
func TestGetCalendar_PodiumEnrichment(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)

	meetings := []storage.RaceMeeting{
		{
			ID: "2026-01", Season: 2026, Round: 1,
			RaceName:         "Australian Grand Prix",
			StartDatetimeUTC: time.Date(2026, 3, 13, 5, 0, 0, 0, time.UTC),
			EndDatetimeUTC:   time.Date(2026, 3, 16, 5, 0, 0, 0, time.UTC),
		},
		{
			ID: "2026-05", Season: 2026, Round: 5,
			RaceName:         "Miami Grand Prix",
			StartDatetimeUTC: time.Date(2026, 5, 4, 19, 0, 0, 0, time.UTC),
			EndDatetimeUTC:   time.Date(2026, 5, 7, 19, 0, 0, 0, time.UTC),
		},
	}

	pts := func(v float64) *float64 { return &v }
	// Round 1 race results, plus a sprint round (round 2) so we exercise the
	// race+sprint cumulative sum.
	results := map[int][]storage.SessionResult{
		1: {
			// Out of order + a position-0 row.
			{Round: 1, SessionType: "race", Position: 0, DriverNumber: 18, DriverName: "Lance Stroll", DriverAcronym: "STR", TeamName: "Aston Martin"},
			{Round: 1, SessionType: "race", Position: 3, DriverNumber: 63, DriverName: "George Russell", DriverAcronym: "RUS", TeamName: "Mercedes", Points: pts(15)},
			{Round: 1, SessionType: "race", Position: 1, DriverNumber: 1, DriverName: "Max Verstappen", DriverAcronym: "VER", TeamName: "Red Bull Racing", Points: pts(25)},
			{Round: 1, SessionType: "race", Position: 2, DriverNumber: 44, DriverName: "Lewis Hamilton", DriverAcronym: "HAM", TeamName: "Ferrari", Points: pts(18)},
			{Round: 1, SessionType: "race", Position: 4, DriverNumber: 4, DriverName: "Lando Norris", DriverAcronym: "NOR", TeamName: "McLaren", Points: pts(12)},
			// Qualifying must NOT contribute to season points.
			{Round: 1, SessionType: "qualifying", Position: 1, DriverNumber: 16, DriverName: "Charles Leclerc", DriverAcronym: "LEC", TeamName: "Ferrari"},
		},
		2: {
			// Sprint contributions — VER wins sprint (8), HAM 2nd (7).
			{Round: 2, SessionType: "sprint", Position: 1, DriverNumber: 1, DriverName: "Max Verstappen", DriverAcronym: "VER", TeamName: "Red Bull Racing", Points: pts(8)},
			{Round: 2, SessionType: "sprint", Position: 2, DriverNumber: 44, DriverName: "Lewis Hamilton", DriverAcronym: "HAM", TeamName: "Ferrari", Points: pts(7)},
		},
	}

	calRepo := &fakeCalendarRepo{meetings: meetings}
	sessRepo := &fakeSessionRepoWithResults{resultsByRound: results}

	svc := calendar.NewServiceWithSessionsAndClock(calRepo, sessRepo, func() time.Time { return now })

	resp, err := svc.GetCalendar(context.Background(), 2026)
	if err != nil {
		t.Fatalf("GetCalendar returned error: %v", err)
	}

	var aus, miami *struct {
		Round  int
		Status string
		Podium int
	}
	for _, r := range resp.Rounds {
		switch r.Round {
		case 1:
			aus = &struct {
				Round  int
				Status string
				Podium int
			}{r.Round, r.Status, len(r.Podium)}
			if r.Status != "completed" {
				t.Errorf("round 1 status = %q, want completed", r.Status)
			}
			if len(r.Podium) != 3 {
				t.Fatalf("round 1 podium length = %d, want 3", len(r.Podium))
			}
			if r.Podium[0].Position != 1 || r.Podium[0].DriverAcronym != "VER" {
				t.Errorf("P1 = %+v, want VER pos=1", r.Podium[0])
			}
			// VER cumulative = 25 (race) + 8 (sprint) = 33
			if r.Podium[0].SeasonPoints != 33 {
				t.Errorf("VER season points = %v, want 33 (25 race + 8 sprint)", r.Podium[0].SeasonPoints)
			}
			if r.Podium[1].Position != 2 || r.Podium[1].DriverAcronym != "HAM" {
				t.Errorf("P2 = %+v, want HAM pos=2", r.Podium[1])
			}
			// HAM cumulative = 18 + 7 = 25
			if r.Podium[1].SeasonPoints != 25 {
				t.Errorf("HAM season points = %v, want 25 (18 race + 7 sprint)", r.Podium[1].SeasonPoints)
			}
			if r.Podium[2].Position != 3 || r.Podium[2].DriverAcronym != "RUS" {
				t.Errorf("P3 = %+v, want RUS pos=3", r.Podium[2])
			}
		case 5:
			miami = &struct {
				Round  int
				Status string
				Podium int
			}{r.Round, r.Status, len(r.Podium)}
			if len(r.Podium) != 0 {
				t.Errorf("upcoming Miami should have empty podium, got %d entries", len(r.Podium))
			}
		}
	}
	if aus == nil {
		t.Fatal("Australian GP missing from response")
	}
	if miami == nil {
		t.Fatal("Miami GP missing from response")
	}
}
