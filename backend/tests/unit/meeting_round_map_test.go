package unit

import (
	"testing"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/ingest"
)

func TestBuildMeetingRoundMap_ExcludesCancelledMeetings(t *testing.T) {
	// Simulate 2026 season: Bahrain (key=100), Saudi (key=200), Australia (key=300), Miami (key=400)
	sessions := []ingest.OpenF1SessionForTest{
		{SessionKey: 1, SessionName: "Race", MeetingKey: 100, DateStart: "2026-03-01T15:00:00+00:00"},
		{SessionKey: 2, SessionName: "Race", MeetingKey: 200, DateStart: "2026-03-15T15:00:00+00:00"},
		{SessionKey: 3, SessionName: "Race", MeetingKey: 300, DateStart: "2026-03-29T15:00:00+00:00"},
		{SessionKey: 4, SessionName: "Sprint", MeetingKey: 400, DateStart: "2026-05-03T15:00:00+00:00"},
		{SessionKey: 5, SessionName: "Race", MeetingKey: 400, DateStart: "2026-05-04T15:00:00+00:00"},
	}

	cancelledKeys := map[int]bool{
		100: true, // Bahrain
		200: true, // Saudi
	}

	result := ingest.BuildMeetingRoundMapForTest(sessions, cancelledKeys)

	// Australia should be Round 1, Miami should be Round 2
	if result[300] != 1 {
		t.Errorf("Australia (key=300): got round %d, want 1", result[300])
	}
	if result[400] != 2 {
		t.Errorf("Miami (key=400): got round %d, want 2", result[400])
	}
	// Cancelled meetings should not appear
	if _, ok := result[100]; ok {
		t.Error("Bahrain (key=100) should not be in round map")
	}
	if _, ok := result[200]; ok {
		t.Error("Saudi (key=200) should not be in round map")
	}
}

func TestBuildMeetingRoundMap_NilCancelledKeys(t *testing.T) {
	// When cancelled keys is nil (fetch failed), all non-testing meetings get rounds
	sessions := []ingest.OpenF1SessionForTest{
		{SessionKey: 1, SessionName: "Race", MeetingKey: 100, DateStart: "2026-03-01T15:00:00+00:00"},
		{SessionKey: 2, SessionName: "Race", MeetingKey: 200, DateStart: "2026-03-15T15:00:00+00:00"},
	}

	result := ingest.BuildMeetingRoundMapForTest(sessions, nil)

	if result[100] != 1 {
		t.Errorf("key=100: got round %d, want 1", result[100])
	}
	if result[200] != 2 {
		t.Errorf("key=200: got round %d, want 2", result[200])
	}
}

func TestBuildMeetingRoundMap_ExcludesTestingAndCancelled(t *testing.T) {
	sessions := []ingest.OpenF1SessionForTest{
		{SessionKey: 1, SessionName: "Day 1", MeetingKey: 50, DateStart: "2026-02-15T10:00:00+00:00"},
		{SessionKey: 2, SessionName: "Race", MeetingKey: 100, DateStart: "2026-03-01T15:00:00+00:00"},
		{SessionKey: 3, SessionName: "Race", MeetingKey: 200, DateStart: "2026-03-15T15:00:00+00:00"},
	}

	cancelledKeys := map[int]bool{
		100: true,
	}

	result := ingest.BuildMeetingRoundMapForTest(sessions, cancelledKeys)

	// Testing meeting (key=50) excluded by testing filter
	// Cancelled meeting (key=100) excluded by cancelled filter
	// Only key=200 remains as Round 1
	if _, ok := result[50]; ok {
		t.Error("Testing meeting (key=50) should not be in round map")
	}
	if _, ok := result[100]; ok {
		t.Error("Cancelled meeting (key=100) should not be in round map")
	}
	if result[200] != 1 {
		t.Errorf("key=200: got round %d, want 1", result[200])
	}
}
