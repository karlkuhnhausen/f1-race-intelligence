package domain_test

import (
	"testing"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/domain"
)

func TestBuildMeetingIndex_BasicRoundAssignment(t *testing.T) {
	meetings := []domain.MeetingForIndex{
		{MeetingKey: 100, RaceName: "Australian Grand Prix", StartDatetimeUTC: time.Date(2026, 3, 15, 5, 0, 0, 0, time.UTC)},
		{MeetingKey: 200, RaceName: "Chinese Grand Prix", StartDatetimeUTC: time.Date(2026, 3, 29, 7, 0, 0, 0, time.UTC)},
		{MeetingKey: 300, RaceName: "Japanese Grand Prix", StartDatetimeUTC: time.Date(2026, 4, 12, 5, 0, 0, 0, time.UTC)},
	}

	idx := domain.BuildMeetingIndex(meetings)

	if idx.TotalRounds() != 3 {
		t.Fatalf("expected 3 rounds, got %d", idx.TotalRounds())
	}
	if got := idx.MeetingKeyForRound(1); got != 100 {
		t.Errorf("round 1: expected meeting_key 100, got %d", got)
	}
	if got := idx.MeetingKeyForRound(2); got != 200 {
		t.Errorf("round 2: expected meeting_key 200, got %d", got)
	}
	if got := idx.RoundForMeetingKey(300); got != 3 {
		t.Errorf("meeting_key 300: expected round 3, got %d", got)
	}
}

func TestBuildMeetingIndex_ExcludesCancelledMeetings(t *testing.T) {
	meetings := []domain.MeetingForIndex{
		{MeetingKey: 50, RaceName: "Bahrain Grand Prix", StartDatetimeUTC: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), IsCancelled: true},
		{MeetingKey: 100, RaceName: "Australian Grand Prix", StartDatetimeUTC: time.Date(2026, 3, 15, 5, 0, 0, 0, time.UTC)},
		{MeetingKey: 200, RaceName: "Chinese Grand Prix", StartDatetimeUTC: time.Date(2026, 3, 29, 7, 0, 0, 0, time.UTC)},
	}

	idx := domain.BuildMeetingIndex(meetings)

	if idx.TotalRounds() != 2 {
		t.Fatalf("expected 2 rounds (cancelled excluded), got %d", idx.TotalRounds())
	}
	if got := idx.MeetingKeyForRound(1); got != 100 {
		t.Errorf("round 1: expected meeting_key 100, got %d", got)
	}
	// Cancelled meeting should not be in the index.
	if got := idx.RoundForMeetingKey(50); got != 0 {
		t.Errorf("cancelled meeting_key 50: expected round 0, got %d", got)
	}
}

func TestBuildMeetingIndex_ExcludesTestingMeetings(t *testing.T) {
	meetings := []domain.MeetingForIndex{
		{MeetingKey: 10, RaceName: "Pre-Season Testing", StartDatetimeUTC: time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)},
		{MeetingKey: 100, RaceName: "Australian Grand Prix", StartDatetimeUTC: time.Date(2026, 3, 15, 5, 0, 0, 0, time.UTC)},
	}

	idx := domain.BuildMeetingIndex(meetings)

	if idx.TotalRounds() != 1 {
		t.Fatalf("expected 1 round (testing excluded), got %d", idx.TotalRounds())
	}
	if got := idx.MeetingKeyForRound(1); got != 100 {
		t.Errorf("round 1: expected meeting_key 100, got %d", got)
	}
}

func TestBuildMeetingIndex_SortsByDate(t *testing.T) {
	// Provide meetings out of date order.
	meetings := []domain.MeetingForIndex{
		{MeetingKey: 300, RaceName: "Japanese Grand Prix", StartDatetimeUTC: time.Date(2026, 4, 12, 5, 0, 0, 0, time.UTC)},
		{MeetingKey: 100, RaceName: "Australian Grand Prix", StartDatetimeUTC: time.Date(2026, 3, 15, 5, 0, 0, 0, time.UTC)},
		{MeetingKey: 200, RaceName: "Chinese Grand Prix", StartDatetimeUTC: time.Date(2026, 3, 29, 7, 0, 0, 0, time.UTC)},
	}

	idx := domain.BuildMeetingIndex(meetings)

	if got := idx.MeetingKeyForRound(1); got != 100 {
		t.Errorf("round 1: expected meeting_key 100 (earliest date), got %d", got)
	}
	if got := idx.MeetingKeyForRound(3); got != 300 {
		t.Errorf("round 3: expected meeting_key 300 (latest date), got %d", got)
	}
}

func TestMeetingIndex_UnknownRoundReturnsZero(t *testing.T) {
	meetings := []domain.MeetingForIndex{
		{MeetingKey: 100, RaceName: "Australian Grand Prix", StartDatetimeUTC: time.Date(2026, 3, 15, 5, 0, 0, 0, time.UTC)},
	}

	idx := domain.BuildMeetingIndex(meetings)

	if got := idx.MeetingKeyForRound(99); got != 0 {
		t.Errorf("unknown round 99: expected 0, got %d", got)
	}
	if got := idx.RoundForMeetingKey(9999); got != 0 {
		t.Errorf("unknown meeting_key 9999: expected 0, got %d", got)
	}
}
