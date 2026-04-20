package unit

import (
	"testing"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/domain"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/ingest"
)

// --- MapOpenF1SessionType tests ---

func TestMapOpenF1SessionType_AllKnownTypes(t *testing.T) {
	cases := []struct {
		input    string
		expected domain.SessionType
	}{
		{"Practice 1", domain.SessionPractice1},
		{"Practice 2", domain.SessionPractice2},
		{"Practice 3", domain.SessionPractice3},
		{"Sprint Qualifying", domain.SessionSprintQualifying},
		{"Sprint", domain.SessionSprint},
		{"Qualifying", domain.SessionQualifying},
		{"Race", domain.SessionRace},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := domain.MapOpenF1SessionType(tc.input)
			if got != tc.expected {
				t.Errorf("MapOpenF1SessionType(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestMapOpenF1SessionType_UnknownFallsThrough(t *testing.T) {
	got := domain.MapOpenF1SessionType("Warm Up")
	if got != domain.SessionType("Warm Up") {
		t.Errorf("expected fallthrough for unknown type, got %q", got)
	}
}

// --- SessionTypeSlug tests ---

func TestSessionTypeSlug(t *testing.T) {
	cases := []struct {
		input    domain.SessionType
		expected string
	}{
		{domain.SessionPractice1, "fp1"},
		{domain.SessionPractice2, "fp2"},
		{domain.SessionPractice3, "fp3"},
		{domain.SessionSprintQualifying, "sprint-qualifying"},
		{domain.SessionSprint, "sprint"},
		{domain.SessionQualifying, "qualifying"},
		{domain.SessionRace, "race"},
	}

	for _, tc := range cases {
		t.Run(string(tc.input), func(t *testing.T) {
			got := domain.SessionTypeSlug(tc.input)
			if got != tc.expected {
				t.Errorf("SessionTypeSlug(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

// --- IsRaceType / IsQualifyingType / IsPracticeType ---

func TestIsRaceType(t *testing.T) {
	if !domain.IsRaceType(domain.SessionRace) {
		t.Error("expected Race to be a race type")
	}
	if !domain.IsRaceType(domain.SessionSprint) {
		t.Error("expected Sprint to be a race type")
	}
	if domain.IsRaceType(domain.SessionQualifying) {
		t.Error("expected Qualifying NOT to be a race type")
	}
}

func TestIsQualifyingType(t *testing.T) {
	if !domain.IsQualifyingType(domain.SessionQualifying) {
		t.Error("expected Qualifying to be a qualifying type")
	}
	if !domain.IsQualifyingType(domain.SessionSprintQualifying) {
		t.Error("expected Sprint Qualifying to be a qualifying type")
	}
	if domain.IsQualifyingType(domain.SessionRace) {
		t.Error("expected Race NOT to be a qualifying type")
	}
}

func TestIsPracticeType(t *testing.T) {
	for _, st := range []domain.SessionType{domain.SessionPractice1, domain.SessionPractice2, domain.SessionPractice3} {
		if !domain.IsPracticeType(st) {
			t.Errorf("expected %q to be a practice type", st)
		}
	}
	if domain.IsPracticeType(domain.SessionRace) {
		t.Error("expected Race NOT to be a practice type")
	}
}

// --- TransformSession tests ---

func TestTransformSession_SetsCorrectFields(t *testing.T) {
	got := ingest.TestTransformSession(9001, "Race", 5000, "2026-03-15T05:00:00Z", "2026-03-15T07:00:00Z", 2026, 2026, 1)

	if got.ID != "2026-01-race" {
		t.Errorf("ID = %q, want %q", got.ID, "2026-01-race")
	}
	if got.SessionType != "race" {
		t.Errorf("SessionType = %q, want %q", got.SessionType, "race")
	}
	if got.Season != 2026 {
		t.Errorf("Season = %d, want 2026", got.Season)
	}
	if got.Round != 1 {
		t.Errorf("Round = %d, want 1", got.Round)
	}
	if got.Status != "completed" {
		t.Errorf("Status = %q, want %q", got.Status, "completed")
	}
	if got.Source != "openf1" {
		t.Errorf("Source = %q, want %q", got.Source, "openf1")
	}
}

func TestTransformSession_PracticeSlug(t *testing.T) {
	got := ingest.TestTransformSession(9002, "Practice 1", 5000, "2026-03-14T01:00:00Z", "2026-03-14T02:00:00Z", 2026, 2026, 3)

	if got.ID != "2026-03-fp1" {
		t.Errorf("ID = %q, want %q", got.ID, "2026-03-fp1")
	}
	if got.SessionType != "practice1" {
		t.Errorf("SessionType = %q, want %q", got.SessionType, "practice1")
	}
}

// --- TransformSessionResult tests ---

func TestTransformSessionResult_RaceType_SetsFinishingStatus(t *testing.T) {
	got := ingest.TestTransformSessionResult(44, 1, "Lewis HAMILTON", "HAM", "Ferrari", domain.SessionRace, 2026, 1, 58)

	if got.FinishingStatus == nil || *got.FinishingStatus != "Finished" {
		t.Errorf("FinishingStatus = %v, want 'Finished'", got.FinishingStatus)
	}
	if got.DriverName != "Lewis HAMILTON" {
		t.Errorf("DriverName = %q, want %q", got.DriverName, "Lewis HAMILTON")
	}
	if got.TeamName != "Ferrari" {
		t.Errorf("TeamName = %q, want %q", got.TeamName, "Ferrari")
	}
	if got.Position != 1 {
		t.Errorf("Position = %d, want 1", got.Position)
	}
}

func TestTransformSessionResult_QualifyingType_NoFinishingStatus(t *testing.T) {
	got := ingest.TestTransformSessionResult(1, 1, "Max VERSTAPPEN", "VER", "Red Bull Racing", domain.SessionQualifying, 2026, 1, 0)

	if got.FinishingStatus != nil {
		t.Errorf("expected no FinishingStatus for qualifying, got %v", *got.FinishingStatus)
	}
}

func TestTransformSessionResult_NilDriver_EmptyFields(t *testing.T) {
	got := ingest.TestTransformSessionResult(99, 10, "", "", "", domain.SessionRace, 2026, 1, 58)

	if got.DriverName != "" {
		t.Errorf("expected empty DriverName for nil driver, got %q", got.DriverName)
	}
	if got.DriverAcronym != "" {
		t.Errorf("expected empty DriverAcronym for nil driver, got %q", got.DriverAcronym)
	}
	if got.TeamName != "" {
		t.Errorf("expected empty TeamName for nil driver, got %q", got.TeamName)
	}
}

func TestTransformSessionResult_SprintType_SetsFinishingStatus(t *testing.T) {
	got := ingest.TestTransformSessionResult(16, 2, "Charles LECLERC", "LEC", "Ferrari", domain.SessionSprint, 2026, 1, 20)

	if got.FinishingStatus == nil || *got.FinishingStatus != "Finished" {
		t.Errorf("Sprint FinishingStatus = %v, want 'Finished'", got.FinishingStatus)
	}
}

func TestTransformSessionResult_IDFormat(t *testing.T) {
	got := ingest.TestTransformSessionResult(44, 1, "Lewis HAMILTON", "HAM", "Ferrari", domain.SessionRace, 2026, 3, 58)

	expected := "2026-03-race-44"
	if got.ID != expected {
		t.Errorf("ID = %q, want %q", got.ID, expected)
	}
}
