package integration

import (
	"testing"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/domain"
)

func TestCancellationOverridesBahrainSaudi(t *testing.T) {
	overrides := domain.CancellationOverrides()

	if len(overrides) != 2 {
		t.Fatalf("expected 2 overrides, got %d", len(overrides))
	}

	expectedNames := map[string]bool{
		"Bahrain Grand Prix":       false,
		"Saudi Arabian Grand Prix": false,
	}

	for _, o := range overrides {
		if o.Season != 2026 {
			t.Errorf("expected season 2026, got %d", o.Season)
		}
		if _, ok := expectedNames[o.RaceName]; !ok {
			t.Errorf("unexpected race name %q", o.RaceName)
		}
		expectedNames[o.RaceName] = true
		if o.Label == "" {
			t.Errorf("%s: label should not be empty", o.RaceName)
		}
		if o.Reason == "" {
			t.Errorf("%s: reason should not be empty", o.RaceName)
		}
	}

	for name, found := range expectedNames {
		if !found {
			t.Errorf("missing override for %s", name)
		}
	}
}

func TestIsCancelledReturnsMatchForBahrain(t *testing.T) {
	override, ok := domain.IsCancelled(2026, "Bahrain Grand Prix")
	if !ok {
		t.Fatal("expected Bahrain Grand Prix to be cancelled")
	}
	if override.RaceName != "Bahrain Grand Prix" {
		t.Errorf("expected Bahrain Grand Prix, got %s", override.RaceName)
	}
}

func TestIsCancelledReturnsFalseForAustralian(t *testing.T) {
	_, ok := domain.IsCancelled(2026, "Australian Grand Prix")
	if ok {
		t.Error("Australian Grand Prix should not be cancelled")
	}
}

func TestIsCancelledReturnsFalseForWrongSeason(t *testing.T) {
	_, ok := domain.IsCancelled(2025, "Bahrain Grand Prix")
	if ok {
		t.Error("2025 Bahrain Grand Prix should not be cancelled")
	}
}

func TestIsCancelledIsCaseInsensitive(t *testing.T) {
	_, ok := domain.IsCancelled(2026, "bahrain grand prix")
	if !ok {
		t.Error("expected case-insensitive match")
	}
}
