package unit

import (
	"testing"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/domain"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/ingest"
)

// --- Race results: rich fields ---

func TestTransformSessionResult_Race_PopulatesPointsTimeAndGap(t *testing.T) {
	rawJSON := `{
		"position": 2,
		"driver_number": 44,
		"number_of_laps": 58,
		"points": 18.0,
		"dnf": false, "dns": false, "dsq": false,
		"duration": 4989.775,
		"gap_to_leader": 2.974,
		"session_key": 11234,
		"meeting_key": 1279
	}`
	got, err := ingest.TestTransformSessionResultJSON(rawJSON, "Lewis HAMILTON", "HAM", "Ferrari", domain.SessionRace, 2026, 3)
	if err != nil {
		t.Fatalf("transform err: %v", err)
	}

	if got.Points == nil || *got.Points != 18.0 {
		t.Errorf("Points = %v, want 18", got.Points)
	}
	if got.RaceTime == nil || *got.RaceTime != 4989.775 {
		t.Errorf("RaceTime = %v, want 4989.775", got.RaceTime)
	}
	if got.GapToLeader == nil || *got.GapToLeader != "+2.974s" {
		t.Errorf("GapToLeader = %v, want '+2.974s'", got.GapToLeader)
	}
	if got.FinishingStatus == nil || *got.FinishingStatus != "Finished" {
		t.Errorf("FinishingStatus = %v, want 'Finished'", got.FinishingStatus)
	}
}

func TestTransformSessionResult_Race_P1HasNilGap(t *testing.T) {
	rawJSON := `{
		"position": 1, "driver_number": 63, "number_of_laps": 58,
		"points": 25.0, "dnf": false, "dns": false, "dsq": false,
		"duration": 4986.801, "gap_to_leader": 0
	}`
	got, err := ingest.TestTransformSessionResultJSON(rawJSON, "George RUSSELL", "RUS", "Mercedes", domain.SessionRace, 2026, 3)
	if err != nil {
		t.Fatalf("transform err: %v", err)
	}

	if got.GapToLeader != nil {
		t.Errorf("expected nil GapToLeader for P1, got %q", *got.GapToLeader)
	}
	if got.RaceTime == nil || *got.RaceTime != 4986.801 {
		t.Errorf("RaceTime = %v, want 4986.801", got.RaceTime)
	}
}

func TestTransformSessionResult_Race_DNFStatus(t *testing.T) {
	rawJSON := `{
		"position": 18, "driver_number": 14, "number_of_laps": 32,
		"points": 0, "dnf": true, "dns": false, "dsq": false,
		"duration": null, "gap_to_leader": null
	}`
	got, err := ingest.TestTransformSessionResultJSON(rawJSON, "Fernando ALONSO", "ALO", "Aston Martin", domain.SessionRace, 2026, 3)
	if err != nil {
		t.Fatalf("transform err: %v", err)
	}

	if got.FinishingStatus == nil || *got.FinishingStatus != "DNF" {
		t.Errorf("FinishingStatus = %v, want 'DNF'", got.FinishingStatus)
	}
	if got.RaceTime != nil {
		t.Errorf("expected nil RaceTime for DNF with null duration, got %v", *got.RaceTime)
	}
	if got.GapToLeader != nil {
		t.Errorf("expected nil GapToLeader for DNF with null gap, got %v", *got.GapToLeader)
	}
}

func TestTransformSessionResult_Race_DSQStatus(t *testing.T) {
	rawJSON := `{
		"position": 20, "driver_number": 10, "number_of_laps": 45,
		"points": 0, "dnf": false, "dns": false, "dsq": true,
		"duration": 5500.0, "gap_to_leader": 100.0
	}`
	got, err := ingest.TestTransformSessionResultJSON(rawJSON, "Pierre GASLY", "GAS", "Alpine", domain.SessionRace, 2026, 3)
	if err != nil {
		t.Fatalf("transform err: %v", err)
	}

	if got.FinishingStatus == nil || *got.FinishingStatus != "DSQ" {
		t.Errorf("FinishingStatus = %v, want 'DSQ'", got.FinishingStatus)
	}
}

func TestTransformSessionResult_Race_DNSStatus(t *testing.T) {
	rawJSON := `{
		"position": 19, "driver_number": 77, "number_of_laps": 0,
		"points": 0, "dnf": false, "dns": true, "dsq": false,
		"duration": null, "gap_to_leader": null
	}`
	got, err := ingest.TestTransformSessionResultJSON(rawJSON, "Valtteri BOTTAS", "BOT", "Sauber", domain.SessionRace, 2026, 3)
	if err != nil {
		t.Fatalf("transform err: %v", err)
	}

	if got.FinishingStatus == nil || *got.FinishingStatus != "DNS" {
		t.Errorf("FinishingStatus = %v, want 'DNS'", got.FinishingStatus)
	}
}

// --- Qualifying results: polymorphic duration array ---

func TestTransformSessionResult_Qualifying_Q3Reached(t *testing.T) {
	rawJSON := `{
		"position": 1, "driver_number": 63, "number_of_laps": 22,
		"dnf": false, "dns": false, "dsq": false,
		"duration": [79.507, 78.934, 78.518],
		"gap_to_leader": [0, 0, 0]
	}`
	got, err := ingest.TestTransformSessionResultJSON(rawJSON, "George RUSSELL", "RUS", "Mercedes", domain.SessionQualifying, 2026, 3)
	if err != nil {
		t.Fatalf("transform err: %v", err)
	}

	if got.Q1Time == nil || *got.Q1Time != 79.507 {
		t.Errorf("Q1Time = %v, want 79.507", got.Q1Time)
	}
	if got.Q2Time == nil || *got.Q2Time != 78.934 {
		t.Errorf("Q2Time = %v, want 78.934", got.Q2Time)
	}
	if got.Q3Time == nil || *got.Q3Time != 78.518 {
		t.Errorf("Q3Time = %v, want 78.518", got.Q3Time)
	}
	if got.FinishingStatus != nil {
		t.Errorf("expected no FinishingStatus on qualifying, got %v", *got.FinishingStatus)
	}
}

func TestTransformSessionResult_Qualifying_Q2Eliminated(t *testing.T) {
	rawJSON := `{
		"position": 11, "driver_number": 44, "number_of_laps": 12,
		"dnf": false, "dns": false, "dsq": false,
		"duration": [80.0, 79.5, null],
		"gap_to_leader": [0.5, 0.6, null]
	}`
	got, err := ingest.TestTransformSessionResultJSON(rawJSON, "Lewis HAMILTON", "HAM", "Ferrari", domain.SessionQualifying, 2026, 3)
	if err != nil {
		t.Fatalf("transform err: %v", err)
	}

	if got.Q1Time == nil || *got.Q1Time != 80.0 {
		t.Errorf("Q1Time = %v, want 80.0", got.Q1Time)
	}
	if got.Q2Time == nil || *got.Q2Time != 79.5 {
		t.Errorf("Q2Time = %v, want 79.5", got.Q2Time)
	}
	if got.Q3Time != nil {
		t.Errorf("expected nil Q3Time for Q2-eliminated driver, got %v", *got.Q3Time)
	}
}

func TestTransformSessionResult_Qualifying_Q1Eliminated(t *testing.T) {
	rawJSON := `{
		"position": 17, "driver_number": 14, "number_of_laps": 10,
		"dnf": false, "dns": false, "dsq": false,
		"duration": [81.969, null, null],
		"gap_to_leader": [2.462, null, null]
	}`
	got, err := ingest.TestTransformSessionResultJSON(rawJSON, "Fernando ALONSO", "ALO", "Aston Martin", domain.SessionQualifying, 2026, 3)
	if err != nil {
		t.Fatalf("transform err: %v", err)
	}

	if got.Q1Time == nil || *got.Q1Time != 81.969 {
		t.Errorf("Q1Time = %v, want 81.969", got.Q1Time)
	}
	if got.Q2Time != nil {
		t.Errorf("expected nil Q2Time for Q1-eliminated driver, got %v", *got.Q2Time)
	}
	if got.Q3Time != nil {
		t.Errorf("expected nil Q3Time for Q1-eliminated driver, got %v", *got.Q3Time)
	}
}

func TestTransformSessionResult_SprintQualifying_Q3Reached(t *testing.T) {
	rawJSON := `{
		"position": 3, "driver_number": 16, "number_of_laps": 18,
		"dnf": false, "dns": false, "dsq": false,
		"duration": [80.1, 79.8, 79.5],
		"gap_to_leader": [0.3, 0.4, 0.5]
	}`
	got, err := ingest.TestTransformSessionResultJSON(rawJSON, "Charles LECLERC", "LEC", "Ferrari", domain.SessionSprintQualifying, 2026, 4)
	if err != nil {
		t.Fatalf("transform err: %v", err)
	}

	if got.Q3Time == nil || *got.Q3Time != 79.5 {
		t.Errorf("Q3Time = %v, want 79.5", got.Q3Time)
	}
}

// --- Practice results ---

func TestTransformSessionResult_Practice_PopulatesBestLapAndGap(t *testing.T) {
	rawJSON := `{
		"position": 2, "driver_number": 44, "number_of_laps": 30,
		"dnf": false, "dns": false, "dsq": false,
		"duration": 80.736, "gap_to_leader": 0.469
	}`
	got, err := ingest.TestTransformSessionResultJSON(rawJSON, "Lewis HAMILTON", "HAM", "Ferrari", domain.SessionPractice1, 2026, 3)
	if err != nil {
		t.Fatalf("transform err: %v", err)
	}

	if got.BestLapTime == nil || *got.BestLapTime != 80.736 {
		t.Errorf("BestLapTime = %v, want 80.736", got.BestLapTime)
	}
	if got.GapToFastest == nil || *got.GapToFastest != 0.469 {
		t.Errorf("GapToFastest = %v, want 0.469", got.GapToFastest)
	}
	if got.FinishingStatus != nil {
		t.Errorf("expected no FinishingStatus on practice, got %v", *got.FinishingStatus)
	}
}

func TestTransformSessionResult_Practice_FastestHasZeroGap(t *testing.T) {
	rawJSON := `{
		"position": 1, "driver_number": 16, "number_of_laps": 33,
		"dnf": false, "dns": false, "dsq": false,
		"duration": 80.267, "gap_to_leader": 0
	}`
	got, err := ingest.TestTransformSessionResultJSON(rawJSON, "Charles LECLERC", "LEC", "Ferrari", domain.SessionPractice1, 2026, 3)
	if err != nil {
		t.Fatalf("transform err: %v", err)
	}

	if got.GapToFastest == nil || *got.GapToFastest != 0 {
		t.Errorf("GapToFastest = %v, want 0", got.GapToFastest)
	}
}

// --- DeriveFastestLap ---

func TestDeriveFastestLap_PicksMinimum(t *testing.T) {
	lapsJSON := `[
		{"driver_number": 1, "lap_number": 1, "lap_duration": 85.5},
		{"driver_number": 1, "lap_number": 2, "lap_duration": 84.2},
		{"driver_number": 44, "lap_number": 1, "lap_duration": 83.9},
		{"driver_number": 16, "lap_number": 1, "lap_duration": 84.5}
	]`
	driver, ok, err := ingest.TestDeriveFastestLap(lapsJSON)
	if err != nil {
		t.Fatalf("derive err: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true with valid lap data")
	}
	if driver != 44 {
		t.Errorf("fastest driver = %d, want 44", driver)
	}
}

func TestDeriveFastestLap_SkipsNullDurations(t *testing.T) {
	lapsJSON := `[
		{"driver_number": 1, "lap_number": 1, "lap_duration": null},
		{"driver_number": 44, "lap_number": 1, "lap_duration": null},
		{"driver_number": 16, "lap_number": 5, "lap_duration": 84.5}
	]`
	driver, ok, err := ingest.TestDeriveFastestLap(lapsJSON)
	if err != nil {
		t.Fatalf("derive err: %v", err)
	}
	if !ok || driver != 16 {
		t.Errorf("got driver=%d ok=%v, want driver=16 ok=true", driver, ok)
	}
}

func TestDeriveFastestLap_EmptyReturnsNotOk(t *testing.T) {
	_, ok, err := ingest.TestDeriveFastestLap(`[]`)
	if err != nil {
		t.Fatalf("derive err: %v", err)
	}
	if ok {
		t.Error("expected ok=false for empty laps")
	}
}

func TestDeriveFastestLap_AllNullsReturnsNotOk(t *testing.T) {
	lapsJSON := `[
		{"driver_number": 1, "lap_number": 1, "lap_duration": null},
		{"driver_number": 44, "lap_number": 1, "lap_duration": null}
	]`
	_, ok, err := ingest.TestDeriveFastestLap(lapsJSON)
	if err != nil {
		t.Fatalf("derive err: %v", err)
	}
	if ok {
		t.Error("expected ok=false when all lap_durations are null")
	}
}
