package ingest

import (
	"testing"
)

func TestSummarizeRaceControl_Empty(t *testing.T) {
	summary := SummarizeRaceControl(nil)
	if summary.RedFlagCount != 0 || summary.SafetyCarCount != 0 || summary.VSCCount != 0 {
		t.Errorf("expected zero counts on empty input, got rf=%d sc=%d vsc=%d",
			summary.RedFlagCount, summary.SafetyCarCount, summary.VSCCount)
	}
	if len(summary.NotableEvents) != 0 {
		t.Errorf("expected no notable events, got %d", len(summary.NotableEvents))
	}
}

func TestSummarizeRaceControl_SingleRedFlag(t *testing.T) {
	msgs := []openF1RaceControlMsg{
		{Flag: "RED", LapNumber: 5},
	}
	summary := SummarizeRaceControl(msgs)
	if summary.RedFlagCount != 1 {
		t.Errorf("RedFlagCount = %d, want 1", summary.RedFlagCount)
	}
	if len(summary.NotableEvents) == 0 || summary.NotableEvents[0].EventType != "red_flag" {
		t.Errorf("expected first notable event to be red_flag, got %v", summary.NotableEvents)
	}
	if summary.NotableEvents[0].LapNumber != 5 {
		t.Errorf("NotableEvents[0].LapNumber = %d, want 5", summary.NotableEvents[0].LapNumber)
	}
}

func TestSummarizeRaceControl_SafetyCarDedup_SameLap(t *testing.T) {
	// Two SC messages on the same lap → counted as one activation.
	msgs := []openF1RaceControlMsg{
		{Message: "SAFETY CAR DEPLOYED", LapNumber: 14},
		{Message: "SAFETY CAR DEPLOYED", LapNumber: 14},
	}
	summary := SummarizeRaceControl(msgs)
	if summary.SafetyCarCount != 1 {
		t.Errorf("SafetyCarCount = %d, want 1 (same-lap dedup)", summary.SafetyCarCount)
	}
}

func TestSummarizeRaceControl_SafetyCarTwoDistinctLaps(t *testing.T) {
	msgs := []openF1RaceControlMsg{
		{Message: "SAFETY CAR DEPLOYED", LapNumber: 14},
		{Message: "SAFETY CAR DEPLOYED", LapNumber: 32},
	}
	summary := SummarizeRaceControl(msgs)
	if summary.SafetyCarCount != 2 {
		t.Errorf("SafetyCarCount = %d, want 2 (distinct laps)", summary.SafetyCarCount)
	}
	if len(summary.NotableEvents) != 1 || summary.NotableEvents[0].Count != 2 {
		t.Errorf("expected one notable event with count=2, got %v", summary.NotableEvents)
	}
}

func TestSummarizeRaceControl_VSCIgnoresEndingMessages(t *testing.T) {
	msgs := []openF1RaceControlMsg{
		{Message: "VIRTUAL SAFETY CAR DEPLOYED", LapNumber: 10},
		{Message: "VIRTUAL SAFETY CAR ENDING", LapNumber: 11},
		{Message: "SAFETY CAR ENDING", LapNumber: 14},
	}
	summary := SummarizeRaceControl(msgs)
	if summary.VSCCount != 1 {
		t.Errorf("VSCCount = %d, want 1 (ending messages ignored)", summary.VSCCount)
	}
	if summary.SafetyCarCount != 0 {
		t.Errorf("SafetyCarCount = %d, want 0 (ending messages ignored)", summary.SafetyCarCount)
	}
}

func TestSummarizeRaceControl_PriorityOrder_RedFlagOverSC(t *testing.T) {
	// Red flag and SC present — red flag must be first notable event.
	msgs := []openF1RaceControlMsg{
		{Message: "SAFETY CAR DEPLOYED", LapNumber: 14},
		{Flag: "RED", LapNumber: 22},
	}
	summary := SummarizeRaceControl(msgs)
	if len(summary.NotableEvents) < 2 {
		t.Fatalf("expected at least 2 notable events, got %d", len(summary.NotableEvents))
	}
	if summary.NotableEvents[0].EventType != "red_flag" {
		t.Errorf("first notable event = %q, want red_flag", summary.NotableEvents[0].EventType)
	}
	if summary.NotableEvents[1].EventType != "safety_car" {
		t.Errorf("second notable event = %q, want safety_car", summary.NotableEvents[1].EventType)
	}
}

func TestSummarizeRaceControl_FetchedAtUTCSet(t *testing.T) {
	summary := SummarizeRaceControl(nil)
	if summary.FetchedAtUTC.IsZero() {
		t.Error("FetchedAtUTC should not be zero")
	}
}
