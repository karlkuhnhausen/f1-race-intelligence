package cosmos

import (
	"context"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// GetCalendarWithFreshness returns all meetings for a season along with the most recent data_as_of_utc.
func (c *Client) GetCalendarWithFreshness(ctx context.Context, season int) ([]storage.RaceMeeting, time.Time, error) {
	meetings, err := c.GetMeetingsBySeason(ctx, season)
	if err != nil {
		return nil, time.Time{}, err
	}

	var latest time.Time
	for _, m := range meetings {
		if m.DataAsOfUTC.After(latest) {
			latest = m.DataAsOfUTC
		}
	}

	return meetings, latest, nil
}
