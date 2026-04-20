package domain

// CancellationOverride defines a hardcoded cancellation for a specific round.
type CancellationOverride struct {
	Season int
	Round  int
	Label  string
	Reason string
}

// CancellationOverrides returns the known cancellation overrides.
// For 2026: R4 Bahrain and R5 Saudi Arabia are cancelled.
func CancellationOverrides() []CancellationOverride {
	return []CancellationOverride{
		{Season: 2026, Round: 4, Label: "Cancelled", Reason: "Race removed from 2026 calendar"},
		{Season: 2026, Round: 5, Label: "Cancelled", Reason: "Race removed from 2026 calendar"},
	}
}

// IsCancelled checks whether a given season/round is in the cancellation override list.
func IsCancelled(season, round int) (CancellationOverride, bool) {
	for _, o := range CancellationOverrides() {
		if o.Season == season && o.Round == round {
			return o, true
		}
	}
	return CancellationOverride{}, false
}
