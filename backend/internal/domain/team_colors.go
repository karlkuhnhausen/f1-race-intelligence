package domain

import "strings"

// TeamColors maps normalized team names to their brand hex colors.
// Mirrors the frontend teamColors.ts mapping.
var TeamColors = map[string]string{
	"mercedes":         "#00d2be",
	"red_bull_racing":  "#3671c6",
	"red_bull":         "#3671c6",
	"ferrari":          "#e8002d",
	"scuderia_ferrari": "#e8002d",
	"mclaren":          "#ff8000",
	"aston_martin":     "#358c75",
	"alpine":           "#ff87bc",
	"haas_f1_team":     "#b6babd",
	"haas":             "#b6babd",
	"williams":         "#64c4ff",
	"racing_bulls":     "#6692ff",
	"cadillac":         "#1e5bc6",
	"audi":             "#00e701",
	"sauber":           "#00e701",
}

// GetTeamColor returns the hex color for a team display name.
// Falls back to an empty string if no match is found.
func GetTeamColor(teamName string) string {
	key := normalizeTeamKey(teamName)
	if c, ok := TeamColors[key]; ok {
		return c
	}
	return ""
}

func normalizeTeamKey(name string) string {
	s := strings.ToLower(name)
	var b strings.Builder
	prevUnderscore := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevUnderscore = false
		} else {
			if !prevUnderscore && b.Len() > 0 {
				b.WriteByte('_')
				prevUnderscore = true
			}
		}
	}
	result := b.String()
	return strings.TrimRight(result, "_")
}
