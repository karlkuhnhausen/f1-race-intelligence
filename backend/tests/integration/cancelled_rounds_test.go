package integration

import (
	"testing"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/domain"
)

func TestCancellationOverridesR4R5(t *testing.T) {
	overrides := domain.CancellationOverrides()

	if len(overrides) != 2 {
		t.Fatalf("expected 2 overrides, got %d", len(overrides))
	}

	expectedRounds := map[int]bool{4: false, 5: false}

	for _, o := range overrides {
		if o.Season != 2026 {
			t.Errorf("expected season 2026, got %d", o.Season)
		}
		if _, ok := expectedRounds[o.Round]; !ok {
			t.Errorf("unexpected round %d", o.Round)
		}
		expectedRounds[o.Round] = true
		if o.Label == "" {
			t.Errorf("round %d: label should not be empty", o.Round)
		}
		if o.Reason == "" {
			t.Errorf("round %d: reason should not be empty", o.Round)
		}
	}

	for round, found := range expectedRounds {
		if !found {
			t.Errorf("missing override for round %d", round)
		}
	}
}

func TestIsCancelledReturnsMatchForR4(t *testing.T) {
	override, ok := domain.IsCancelled(2026, 4)
	if !ok {
		t.Fatal("expected R4 to be cancelled")
	}
	if override.Round != 4 {
		t.Errorf("expected round 4, got %d", override.Round)
	}
}

func TestIsCancelledReturnsFalseForR1(t *testing.T) {
	_, ok := domain.IsCancelled(2026, 1)
	if ok {
		t.Error("R1 should not be cancelled")
	}
}

func TestIsCancelledReturnsFalseForWrongSeason(t *testing.T) {
	_, ok := domain.IsCancelled(2025, 4)
	if ok {
		t.Error("2025 R4 should not be cancelled")
	}
}
