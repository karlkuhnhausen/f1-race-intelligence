package rounds

import (
	"context"
	"testing"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

func TestDeriveSessionStatus(t *testing.T) {
	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name      string
		dateStart time.Time
		dateEnd   time.Time
		want      string
	}{
		{
			name:      "future session is upcoming",
			dateStart: now.Add(48 * time.Hour),
			dateEnd:   now.Add(50 * time.Hour),
			want:      statusUpcoming,
		},
		{
			name:      "currently running session is in_progress",
			dateStart: now.Add(-30 * time.Minute),
			dateEnd:   now.Add(30 * time.Minute),
			want:      statusInProgress,
		},
		{
			name:      "past session is completed",
			dateStart: now.Add(-50 * time.Hour),
			dateEnd:   now.Add(-48 * time.Hour),
			want:      statusCompleted,
		},
		{
			name:      "zero start defaults to upcoming",
			dateStart: time.Time{},
			dateEnd:   time.Time{},
			want:      statusUpcoming,
		},
		{
			name:      "started but no end date is in_progress",
			dateStart: now.Add(-1 * time.Hour),
			dateEnd:   time.Time{},
			want:      statusInProgress,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := deriveSessionStatus(now, tc.dateStart, tc.dateEnd)
			if got != tc.want {
				t.Errorf("deriveSessionStatus(%v, %v, %v) = %q, want %q",
					now, tc.dateStart, tc.dateEnd, got, tc.want)
			}
		})
	}
}

// fakeSessionRepo implements storage.SessionRepository with the minimum
// surface needed by GetRoundDetail.
type fakeSessionRepo struct {
	sessions []storage.Session
	results  []storage.SessionResult
}

func (f *fakeSessionRepo) UpsertSession(_ context.Context, _ storage.Session) error {
	return nil
}
func (f *fakeSessionRepo) UpsertSessionResult(_ context.Context, _ storage.SessionResult) error {
	return nil
}
func (f *fakeSessionRepo) GetSessionsByRound(_ context.Context, _, _ int) ([]storage.Session, error) {
	return f.sessions, nil
}
func (f *fakeSessionRepo) GetSessionResultsByRound(_ context.Context, _, _ int) ([]storage.SessionResult, error) {
	return f.results, nil
}
func (f *fakeSessionRepo) GetSessionResultsBySeason(_ context.Context, _ int) ([]storage.SessionResult, error) {
	return f.results, nil
}
func (f *fakeSessionRepo) GetFinalizedSessionKeys(_ context.Context, _ int) (map[int]int, error) {
	return map[int]int{}, nil
}

// fakeCalendarRepo implements storage.CalendarRepository with the minimum
// surface needed by GetRoundDetail.
type fakeCalendarRepo struct {
	meetings []storage.RaceMeeting
}

func (f *fakeCalendarRepo) UpsertMeeting(_ context.Context, _ storage.RaceMeeting) error {
	return nil
}
func (f *fakeCalendarRepo) GetMeetingsBySeason(_ context.Context, _ int) ([]storage.RaceMeeting, error) {
	return f.meetings, nil
}
func (f *fakeCalendarRepo) GetMeetingByID(_ context.Context, _ int, _ string) (*storage.RaceMeeting, error) {
	return nil, nil
}

func TestGetRoundDetail_StatusOverridesStoredValue(t *testing.T) {
	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)

	// Stored status is intentionally wrong ("completed") to verify the
	// service overrides it based on dates.
	sessRepo := &fakeSessionRepo{
		sessions: []storage.Session{
			{
				ID:           "2026-08-fp1",
				Season:       2026,
				Round:        8,
				SessionName:  "Practice 1",
				SessionType:  "practice1",
				Status:       "completed", // stale stored value
				DateStartUTC: now.Add(72 * time.Hour),
				DateEndUTC:   now.Add(73 * time.Hour),
			},
			{
				ID:           "2026-08-race",
				Season:       2026,
				Round:        8,
				SessionName:  "Race",
				SessionType:  "race",
				Status:       "completed", // stale stored value
				DateStartUTC: now.Add(-50 * time.Hour),
				DateEndUTC:   now.Add(-48 * time.Hour),
			},
		},
	}
	calRepo := &fakeCalendarRepo{
		meetings: []storage.RaceMeeting{
			{Season: 2026, Round: 8, RaceName: "Test GP"},
		},
	}

	svc := NewServiceWithClock(sessRepo, calRepo, func() time.Time { return now })
	resp, err := svc.GetRoundDetail(context.Background(), 2026, 8)
	if err != nil {
		t.Fatalf("GetRoundDetail: %v", err)
	}
	if len(resp.Sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(resp.Sessions))
	}

	gotByType := map[string]string{}
	for _, s := range resp.Sessions {
		gotByType[s.SessionType] = s.Status
	}
	if gotByType["practice1"] != statusUpcoming {
		t.Errorf("practice1 status = %q, want %q", gotByType["practice1"], statusUpcoming)
	}
	if gotByType["race"] != statusCompleted {
		t.Errorf("race status = %q, want %q", gotByType["race"], statusCompleted)
	}
}

// TestGetRoundDetail_FiltersAndSortsResults verifies that session results
// are sorted ascending by position, with unclassified rows (position 0,
// typically DNF/DNS/DSQ) sorted to the bottom rather than dropped — the
// frontend renders them in a separate non-classified section.
func TestGetRoundDetail_FiltersAndSortsResults(t *testing.T) {
	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)

	sessRepo := &fakeSessionRepo{
		sessions: []storage.Session{
			{
				ID: "2026-03-race", Season: 2026, Round: 3,
				SessionName: "Race", SessionType: "race",
				DateStartUTC: now.Add(-50 * time.Hour),
				DateEndUTC:   now.Add(-48 * time.Hour),
			},
		},
		results: []storage.SessionResult{
			{SessionType: "race", Position: 0, DriverNumber: 18, DriverName: "Lance Stroll"},
			{SessionType: "race", Position: 3, DriverNumber: 63, DriverName: "George Russell"},
			{SessionType: "race", Position: 1, DriverNumber: 1, DriverName: "Max Verstappen"},
			{SessionType: "race", Position: 2, DriverNumber: 44, DriverName: "Lewis Hamilton"},
		},
	}
	calRepo := &fakeCalendarRepo{
		meetings: []storage.RaceMeeting{{Season: 2026, Round: 3, RaceName: "Australian GP"}},
	}

	svc := NewServiceWithClock(sessRepo, calRepo, func() time.Time { return now })
	resp, err := svc.GetRoundDetail(context.Background(), 2026, 3)
	if err != nil {
		t.Fatalf("GetRoundDetail: %v", err)
	}
	if len(resp.Sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(resp.Sessions))
	}
	results := resp.Sessions[0].Results
	if len(results) != 4 {
		t.Fatalf("expected 4 results (DNF kept), got %d", len(results))
	}
	// First three should be P1..P3 in order.
	for i := 0; i < 3; i++ {
		if results[i].Position != i+1 {
			t.Errorf("results[%d].Position = %d, want %d", i, results[i].Position, i+1)
		}
	}
	if results[0].DriverName != "Max Verstappen" {
		t.Errorf("P1 = %q, want Max Verstappen", results[0].DriverName)
	}
	// Position-0 row sorted to the bottom.
	if results[3].Position != 0 || results[3].DriverName != "Lance Stroll" {
		t.Errorf("last row = pos=%d %q, want pos=0 Lance Stroll",
			results[3].Position, results[3].DriverName)
	}
}
