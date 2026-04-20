package cosmos

import "github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"

// Compile-time interface checks.
var (
	_ storage.CalendarRepository  = (*Client)(nil)
	_ storage.StandingsRepository = (*Client)(nil)
)
