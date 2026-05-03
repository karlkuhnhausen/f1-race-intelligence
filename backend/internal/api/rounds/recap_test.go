package rounds

import (
	"testing"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// helpers

func boolPtr(b bool) *bool      { return &b }
func f64Ptr(f float64) *float64 { return &f }
func strPtr(s string) *string   { return &s }

func completedSess(sessionType string) storage.Session {
	now := time.Now().UTC()
	return storage.Session{
		ID:           "sess-001",
		SessionType:  sessionType,
		DateStartUTC: now.Add(-3 * time.Hour),
		DateEndUTC:   now.Add(-1 * time.Hour),
	}
}

// --- deriveRecapSummary: race path ---

func TestDeriveRecapSummary_Race_WinnerFields(t *testing.T) {
	sess := completedSess("race")
	results := []SessionResultDTO{
		{Position: 1, DriverName: "Max Verstappen", TeamName: "Red Bull Racing", NumberOfLaps: 57},
		{Position: 2, DriverName: "Lando Norris", TeamName: "McLaren", GapToLeader: strPtr("+8.294")},
	}
	recap := deriveRecapSummary(sess, results)
	if recap == nil {
		t.Fatal("expected recap, got nil")
	}
	if recap.WinnerName != "Max Verstappen" {
		t.Errorf("WinnerName = %q, want Max Verstappen", recap.WinnerName)
	}
	if recap.WinnerTeam != "Red Bull Racing" {
		t.Errorf("WinnerTeam = %q, want Red Bull Racing", recap.WinnerTeam)
	}
	if recap.TotalLaps != 57 {
		t.Errorf("TotalLaps = %d, want 57", recap.TotalLaps)
	}
	if recap.GapToP2 != "+8.294" {
		t.Errorf("GapToP2 = %q, want +8.294", recap.GapToP2)
	}
}

func TestDeriveRecapSummary_Race_NoClassifiedFinishers_ReturnsNil(t *testing.T) {
	sess := completedSess("race")
	// No P1 result
	results := []SessionResultDTO{
		{Position: 0, DriverName: "Driver A", TeamName: "Team A"},
	}
	recap := deriveRecapSummary(sess, results)
	if recap != nil {
		t.Errorf("expected nil recap when no P1 finisher, got %+v", recap)
	}
}

func TestDeriveRecapSummary_Race_FastestLapHolder(t *testing.T) {
	sess := completedSess("race")
	sess.FastestLapTimeSeconds = f64Ptr(87.097)
	results := []SessionResultDTO{
		{Position: 1, DriverName: "Max Verstappen", TeamName: "Red Bull Racing", NumberOfLaps: 53},
		{Position: 2, DriverName: "Lando Norris", TeamName: "McLaren", FastestLap: boolPtr(true)},
	}
	recap := deriveRecapSummary(sess, results)
	if recap == nil {
		t.Fatal("expected recap, got nil")
	}
	if recap.FastestLapHolder != "Lando Norris" {
		t.Errorf("FastestLapHolder = %q, want Lando Norris", recap.FastestLapHolder)
	}
	if recap.FastestLapTimeSeconds == nil || *recap.FastestLapTimeSeconds != 87.097 {
		t.Errorf("FastestLapTimeSeconds = %v, want 87.097", recap.FastestLapTimeSeconds)
	}
}

func TestDeriveRecapSummary_Race_RaceControlFieldsFromSummary(t *testing.T) {
	sess := completedSess("race")
	sess.RaceControlSummary = &storage.RaceControlSummary{
		SafetyCarCount: 2,
		NotableEvents: []storage.NotableEvent{
			{EventType: "safety_car", LapNumber: 14, Count: 2},
		},
	}
	results := []SessionResultDTO{
		{Position: 1, DriverName: "Max Verstappen", TeamName: "Red Bull Racing", NumberOfLaps: 53},
	}
	recap := deriveRecapSummary(sess, results)
	if recap == nil {
		t.Fatal("expected recap, got nil")
	}
	if recap.SafetyCarCount != 2 {
		t.Errorf("SafetyCarCount = %d, want 2", recap.SafetyCarCount)
	}
	if recap.TopEvent == nil || recap.TopEvent.EventType != "safety_car" {
		t.Errorf("TopEvent = %v, want safety_car", recap.TopEvent)
	}
}

func TestDeriveRecapSummary_Race_NilRaceControl_NoEventFields(t *testing.T) {
	sess := completedSess("race")
	results := []SessionResultDTO{
		{Position: 1, DriverName: "Max Verstappen", TeamName: "Red Bull Racing", NumberOfLaps: 53},
	}
	recap := deriveRecapSummary(sess, results)
	if recap == nil {
		t.Fatal("expected recap, got nil")
	}
	if recap.TopEvent != nil {
		t.Errorf("expected nil TopEvent when RaceControlSummary is nil, got %+v", recap.TopEvent)
	}
	if recap.SafetyCarCount != 0 || recap.RedFlagCount != 0 {
		t.Errorf("expected zero event counts, got sc=%d rf=%d", recap.SafetyCarCount, recap.RedFlagCount)
	}
}

// --- deriveRecapSummary: qualifying path ---

func TestDeriveRecapSummary_Qualifying_PoleSitterAndCutoffs(t *testing.T) {
	sess := completedSess("qualifying")
	results := []SessionResultDTO{
		// P1 — has Q3 time (pole)
		{Position: 1, DriverName: "Charles Leclerc", TeamName: "Ferrari",
			Q1Time: f64Ptr(88.0), Q2Time: f64Ptr(87.5), Q3Time: f64Ptr(86.983)},
		// P2 — has Q3 time
		{Position: 2, DriverName: "Max Verstappen", TeamName: "Red Bull Racing",
			Q1Time: f64Ptr(88.1), Q2Time: f64Ptr(87.6), Q3Time: f64Ptr(87.115)},
		// P10 — has Q2 but no Q3 (Q2 cutoff)
		{Position: 10, DriverName: "Driver10", TeamName: "Team10",
			Q1Time: f64Ptr(88.8), Q2Time: f64Ptr(87.654)},
		// P15 — has Q1 but no Q2 (Q1 cutoff)
		{Position: 15, DriverName: "Driver15", TeamName: "Team15",
			Q1Time: f64Ptr(89.2)},
	}
	recap := deriveRecapSummary(sess, results)
	if recap == nil {
		t.Fatal("expected recap, got nil")
	}
	if recap.PoleSitterName != "Charles Leclerc" {
		t.Errorf("PoleSitterName = %q, want Charles Leclerc", recap.PoleSitterName)
	}
	if recap.PoleTime == nil || *recap.PoleTime != 86.983 {
		t.Errorf("PoleTime = %v, want 86.983", recap.PoleTime)
	}
	if recap.Q1CutoffTime == nil {
		t.Error("Q1CutoffTime should be set")
	}
	if recap.Q2CutoffTime == nil {
		t.Error("Q2CutoffTime should be set")
	}
}

func TestDeriveRecapSummary_Qualifying_GapToP2Formatted(t *testing.T) {
	sess := completedSess("qualifying")
	results := []SessionResultDTO{
		{Position: 1, DriverName: "Charles Leclerc", TeamName: "Ferrari", Q3Time: f64Ptr(86.983)},
		{Position: 2, DriverName: "Max Verstappen", TeamName: "Red Bull Racing", Q3Time: f64Ptr(87.115)},
	}
	recap := deriveRecapSummary(sess, results)
	if recap == nil {
		t.Fatal("expected recap, got nil")
	}
	// Gap should be "+0.132" (87.115 - 86.983)
	if recap.GapToP2 == "" {
		t.Error("GapToP2 should not be empty when P2 time is available")
	}
}

func TestDeriveRecapSummary_Qualifying_SprintQualifying_NoQ1Q2Cutoffs(t *testing.T) {
	sess := completedSess("sprint_qualifying")
	// Sprint qualifying: results only have Q1 time (single segment)
	results := []SessionResultDTO{
		{Position: 1, DriverName: "Max Verstappen", TeamName: "Red Bull Racing", Q1Time: f64Ptr(86.5)},
		{Position: 2, DriverName: "Lando Norris", TeamName: "McLaren", Q1Time: f64Ptr(86.8)},
	}
	recap := deriveRecapSummary(sess, results)
	if recap == nil {
		t.Fatal("expected recap, got nil")
	}
	if recap.PoleSitterName != "Max Verstappen" {
		t.Errorf("PoleSitterName = %q, want Max Verstappen", recap.PoleSitterName)
	}
	// PoleTime falls back to Q1Time when Q2/Q3 absent.
	if recap.PoleTime == nil || *recap.PoleTime != 86.5 {
		t.Errorf("PoleTime = %v, want 86.5", recap.PoleTime)
	}
	// Q2CutoffTime must be nil — no driver has a Q2 time.
	if recap.Q2CutoffTime != nil {
		t.Errorf("Q2CutoffTime should be nil when no Q2 times present, got %v", recap.Q2CutoffTime)
	}
}

func TestDeriveRecapSummary_Qualifying_RedFlagCount(t *testing.T) {
	sess := completedSess("qualifying")
	sess.RaceControlSummary = &storage.RaceControlSummary{
		RedFlagCount: 2,
		NotableEvents: []storage.NotableEvent{
			{EventType: "red_flag", LapNumber: 3, Count: 2},
		},
	}
	results := []SessionResultDTO{
		{Position: 1, DriverName: "Charles Leclerc", TeamName: "Ferrari", Q3Time: f64Ptr(86.983)},
	}
	recap := deriveRecapSummary(sess, results)
	if recap == nil {
		t.Fatal("expected recap, got nil")
	}
	if recap.RedFlagCount != 2 {
		t.Errorf("RedFlagCount = %d, want 2", recap.RedFlagCount)
	}
}

// --- deriveRecapSummary: practice path ---

func TestDeriveRecapSummary_Practice_BestDriverAndLaps(t *testing.T) {
	sess := completedSess("practice1")
	results := []SessionResultDTO{
		{Position: 1, DriverName: "Lando Norris", TeamName: "McLaren",
			BestLapTime: f64Ptr(88.5), NumberOfLaps: 22},
		{Position: 2, DriverName: "Max Verstappen", TeamName: "Red Bull Racing",
			BestLapTime: f64Ptr(88.8), NumberOfLaps: 18},
	}
	recap := deriveRecapSummary(sess, results)
	if recap == nil {
		t.Fatal("expected recap, got nil")
	}
	if recap.BestDriverName != "Lando Norris" {
		t.Errorf("BestDriverName = %q, want Lando Norris", recap.BestDriverName)
	}
	if recap.BestLapTime == nil || *recap.BestLapTime != 88.5 {
		t.Errorf("BestLapTime = %v, want 88.5", recap.BestLapTime)
	}
	// TotalLaps = sum across all drivers
	if recap.TotalLaps != 40 {
		t.Errorf("TotalLaps = %d, want 40 (22+18)", recap.TotalLaps)
	}
}

// --- deriveTopEvent ---

func TestDeriveTopEvent_NilSummary_ReturnsNil(t *testing.T) {
	if got := deriveTopEvent(nil); got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestDeriveTopEvent_EmptyEvents_ReturnsNil(t *testing.T) {
	rc := &storage.RaceControlSummary{}
	if got := deriveTopEvent(rc); got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestDeriveTopEvent_ReturnsFirstEvent(t *testing.T) {
	rc := &storage.RaceControlSummary{
		NotableEvents: []storage.NotableEvent{
			{EventType: "red_flag", LapNumber: 5, Count: 1},
			{EventType: "safety_car", LapNumber: 14, Count: 2},
		},
	}
	got := deriveTopEvent(rc)
	if got == nil {
		t.Fatal("expected event, got nil")
	}
	if got.EventType != "red_flag" {
		t.Errorf("EventType = %q, want red_flag", got.EventType)
	}
	if got.Count != 1 {
		t.Errorf("Count = %d, want 1", got.Count)
	}
}
