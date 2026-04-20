package domain

import "strings"

// CancellationOverride defines a hardcoded cancellation for a specific race.
type CancellationOverride struct {
	Season   int
	RaceName string // matched case-insensitively against meeting name
	Label    string
	Reason   string
}

// CancellationOverrides returns the known cancellation overrides.
// For 2026: Bahrain Grand Prix and Saudi Arabian Grand Prix are cancelled.
func CancellationOverrides() []CancellationOverride {
	return []CancellationOverride{
		{Season: 2026, RaceName: "Bahrain Grand Prix", Label: "Cancelled", Reason: "Race removed from 2026 calendar"},
		{Season: 2026, RaceName: "Saudi Arabian Grand Prix", Label: "Cancelled", Reason: "Race removed from 2026 calendar"},
	}
}

// IsCancelled checks whether a given season/race name is in the cancellation override list.
func IsCancelled(season int, raceName string) (CancellationOverride, bool) {
	for _, o := range CancellationOverrides() {
		if o.Season == season && strings.EqualFold(o.RaceName, raceName) {
			return o, true
		}
	}
	return CancellationOverride{}, false
}
