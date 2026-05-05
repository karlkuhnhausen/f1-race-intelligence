package standings

import (
	"encoding/json"
	"testing"
)

// TestExtractorsWithRealOpenF1Shapes validates that our type-safe extractors
// handle every known shape OpenF1 returns without panicking or losing data.
// Add new JSON samples here when OpenF1 surprises us with a new type.
func TestExtractorsWithRealOpenF1Shapes(t *testing.T) {
	// Real JSON from /v1/session_result — contains:
	// - gap_to_leader: 0 (number for winner)
	// - gap_to_leader: 2.974 (float for time gap)
	// - gap_to_leader: "+1 LAP" (string for lapped driver)
	// - points: 25.0 (float), null
	// - duration: null, 4986.801 (float)
	sampleJSON := `[
		{
			"position": 1,
			"driver_number": 63,
			"number_of_laps": 58,
			"points": 25.0,
			"dnf": false,
			"dns": false,
			"dsq": false,
			"duration": 4986.801,
			"gap_to_leader": 0,
			"meeting_key": 1279,
			"session_key": 11234
		},
		{
			"position": 2,
			"driver_number": 12,
			"number_of_laps": 58,
			"points": 18.0,
			"dnf": false,
			"dns": false,
			"dsq": false,
			"gap_to_leader": 2.974,
			"duration": 4989.775,
			"meeting_key": 1279,
			"session_key": 11234
		},
		{
			"position": 18,
			"driver_number": 99,
			"number_of_laps": 57,
			"points": null,
			"dnf": false,
			"dns": false,
			"dsq": false,
			"gap_to_leader": "+1 LAP",
			"duration": null,
			"meeting_key": 1279,
			"session_key": 11234
		},
		{
			"position": 20,
			"driver_number": 77,
			"number_of_laps": 30,
			"points": null,
			"dnf": true,
			"dns": false,
			"dsq": false,
			"gap_to_leader": null,
			"duration": null,
			"meeting_key": 1279,
			"session_key": 11234
		}
	]`

	var rows []map[string]any
	if err := json.Unmarshal([]byte(sampleJSON), &rows); err != nil {
		t.Fatalf("JSON unmarshal failed (this should never happen with []map[string]any): %v", err)
	}

	tests := []struct {
		name         string
		row          int
		wantPos      int
		wantDriver   int
		wantPoints   float64
		wantDNF      bool
		wantGap      string
		wantDuration float64
		wantLaps     int
	}{
		{
			name: "winner — gap_to_leader is 0 (number)",
			row:  0, wantPos: 1, wantDriver: 63, wantPoints: 25.0,
			wantDNF: false, wantGap: "0", wantDuration: 4986.801, wantLaps: 58,
		},
		{
			name: "P2 — gap_to_leader is float",
			row:  1, wantPos: 2, wantDriver: 12, wantPoints: 18.0,
			wantDNF: false, wantGap: "2.974", wantDuration: 4989.775, wantLaps: 58,
		},
		{
			name: "lapped — gap_to_leader is string +1 LAP",
			row:  2, wantPos: 18, wantDriver: 99, wantPoints: 0,
			wantDNF: false, wantGap: "+1 LAP", wantDuration: 0, wantLaps: 57,
		},
		{
			name: "DNF — gap_to_leader is null, points null",
			row:  3, wantPos: 20, wantDriver: 77, wantPoints: 0,
			wantDNF: true, wantGap: "", wantDuration: 0, wantLaps: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := rows[tt.row]

			if got := getInt(r, "position"); got != tt.wantPos {
				t.Errorf("position = %d, want %d", got, tt.wantPos)
			}
			if got := getInt(r, "driver_number"); got != tt.wantDriver {
				t.Errorf("driver_number = %d, want %d", got, tt.wantDriver)
			}
			if got := getFloat(r, "points"); got != tt.wantPoints {
				t.Errorf("points = %f, want %f", got, tt.wantPoints)
			}
			if got := getBool(r, "dnf"); got != tt.wantDNF {
				t.Errorf("dnf = %v, want %v", got, tt.wantDNF)
			}
			if got := getString(r, "gap_to_leader"); got != tt.wantGap {
				t.Errorf("gap_to_leader = %q, want %q", got, tt.wantGap)
			}
			if got := getFloat(r, "duration"); got != tt.wantDuration {
				t.Errorf("duration = %f, want %f", got, tt.wantDuration)
			}
			if got := getInt(r, "number_of_laps"); got != tt.wantLaps {
				t.Errorf("number_of_laps = %d, want %d", got, tt.wantLaps)
			}
		})
	}
}

// TestExtractorsWithStartingGridShapes validates starting grid decoding.
func TestExtractorsWithStartingGridShapes(t *testing.T) {
	sampleJSON := `[
		{"position": 1, "driver_number": 63, "lap_duration": 78.234, "meeting_key": 1279},
		{"position": 20, "driver_number": 77, "lap_duration": null, "meeting_key": 1279}
	]`

	var rows []map[string]any
	if err := json.Unmarshal([]byte(sampleJSON), &rows); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	// P1 — has lap duration
	if got := getInt(rows[0], "position"); got != 1 {
		t.Errorf("position = %d, want 1", got)
	}
	if got := getFloat(rows[0], "lap_duration"); got != 78.234 {
		t.Errorf("lap_duration = %f, want 78.234", got)
	}

	// P20 — null lap duration
	if got := getFloat(rows[1], "lap_duration"); got != 0 {
		t.Errorf("lap_duration = %f, want 0", got)
	}
}

// TestExtractorsMissingKeys verifies extractors return zero values for missing keys.
func TestExtractorsMissingKeys(t *testing.T) {
	m := map[string]any{}

	if got := getInt(m, "missing"); got != 0 {
		t.Errorf("getInt missing = %d, want 0", got)
	}
	if got := getFloat(m, "missing"); got != 0 {
		t.Errorf("getFloat missing = %f, want 0", got)
	}
	if got := getBool(m, "missing"); got != false {
		t.Errorf("getBool missing = %v, want false", got)
	}
	if got := getString(m, "missing"); got != "" {
		t.Errorf("getString missing = %q, want empty", got)
	}
	if got := getFloatPtr(m, "missing"); got != nil {
		t.Errorf("getFloatPtr missing = %v, want nil", got)
	}
}
