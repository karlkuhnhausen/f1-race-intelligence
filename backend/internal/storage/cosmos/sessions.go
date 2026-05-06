package cosmos

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// --- SessionRepository ---

func (c *Client) UpsertSession(ctx context.Context, s storage.Session) error {
	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("cosmos: marshal session: %w", err)
	}

	pk := azcosmos.NewPartitionKeyNumber(float64(s.Season))
	_, err = c.sessions.UpsertItem(ctx, pk, data, nil)
	if err != nil {
		return fmt.Errorf("cosmos: upsert session %s: %w", s.ID, err)
	}
	return nil
}

func (c *Client) UpsertSessionResult(ctx context.Context, r storage.SessionResult) error {
	data, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("cosmos: marshal session result: %w", err)
	}

	pk := azcosmos.NewPartitionKeyNumber(float64(r.Season))
	_, err = c.sessions.UpsertItem(ctx, pk, data, nil)
	if err != nil {
		return fmt.Errorf("cosmos: upsert session result %s: %w", r.ID, err)
	}
	return nil
}

func (c *Client) GetSessionsByRound(ctx context.Context, season, round int) ([]storage.Session, error) {
	pk := azcosmos.NewPartitionKeyNumber(float64(season))
	query := "SELECT * FROM c WHERE c.season = @season AND c.round = @round AND c.type = 'session'"
	queryOpts := &azcosmos.QueryOptions{
		QueryParameters: []azcosmos.QueryParameter{
			{Name: "@season", Value: season},
			{Name: "@round", Value: round},
		},
	}

	pager := c.sessions.NewQueryItemsPager(query, pk, queryOpts)
	var sessions []storage.Session

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("cosmos: query sessions: %w", err)
		}
		for _, item := range resp.Items {
			var s storage.Session
			if err := json.Unmarshal(item, &s); err != nil {
				return nil, fmt.Errorf("cosmos: unmarshal session: %w", err)
			}
			sessions = append(sessions, s)
		}
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].DateStartUTC.Before(sessions[j].DateStartUTC)
	})
	return sessions, nil
}

func (c *Client) GetSessionResultsByRound(ctx context.Context, season, round int) ([]storage.SessionResult, error) {
	pk := azcosmos.NewPartitionKeyNumber(float64(season))
	query := "SELECT * FROM c WHERE c.season = @season AND c.round = @round AND c.type = 'session_result'"
	queryOpts := &azcosmos.QueryOptions{
		QueryParameters: []azcosmos.QueryParameter{
			{Name: "@season", Value: season},
			{Name: "@round", Value: round},
		},
	}

	pager := c.sessions.NewQueryItemsPager(query, pk, queryOpts)
	var results []storage.SessionResult

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("cosmos: query session results: %w", err)
		}
		for _, item := range resp.Items {
			var r storage.SessionResult
			if err := json.Unmarshal(item, &r); err != nil {
				return nil, fmt.Errorf("cosmos: unmarshal session result: %w", err)
			}
			results = append(results, r)
		}
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].SessionType != results[j].SessionType {
			return results[i].SessionType < results[j].SessionType
		}
		return results[i].Position < results[j].Position
	})
	return results, nil
}

func (c *Client) GetSessionsByMeetingKey(ctx context.Context, season, meetingKey int) ([]storage.Session, error) {
	pk := azcosmos.NewPartitionKeyNumber(float64(season))
	query := "SELECT * FROM c WHERE c.season = @season AND c.meeting_key = @meetingKey AND c.type = 'session'"
	queryOpts := &azcosmos.QueryOptions{
		QueryParameters: []azcosmos.QueryParameter{
			{Name: "@season", Value: season},
			{Name: "@meetingKey", Value: meetingKey},
		},
	}

	pager := c.sessions.NewQueryItemsPager(query, pk, queryOpts)
	var sessions []storage.Session

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("cosmos: query sessions by meeting_key: %w", err)
		}
		for _, item := range resp.Items {
			var s storage.Session
			if err := json.Unmarshal(item, &s); err != nil {
				return nil, fmt.Errorf("cosmos: unmarshal session: %w", err)
			}
			sessions = append(sessions, s)
		}
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].DateStartUTC.Before(sessions[j].DateStartUTC)
	})
	return sessions, nil
}

func (c *Client) GetSessionResultsByMeetingKey(ctx context.Context, season, meetingKey int) ([]storage.SessionResult, error) {
	pk := azcosmos.NewPartitionKeyNumber(float64(season))
	query := "SELECT * FROM c WHERE c.season = @season AND c.meeting_key = @meetingKey AND c.type = 'session_result'"
	queryOpts := &azcosmos.QueryOptions{
		QueryParameters: []azcosmos.QueryParameter{
			{Name: "@season", Value: season},
			{Name: "@meetingKey", Value: meetingKey},
		},
	}

	pager := c.sessions.NewQueryItemsPager(query, pk, queryOpts)
	var results []storage.SessionResult

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("cosmos: query session results by meeting_key: %w", err)
		}
		for _, item := range resp.Items {
			var r storage.SessionResult
			if err := json.Unmarshal(item, &r); err != nil {
				return nil, fmt.Errorf("cosmos: unmarshal session result: %w", err)
			}
			results = append(results, r)
		}
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].SessionType != results[j].SessionType {
			return results[i].SessionType < results[j].SessionType
		}
		return results[i].Position < results[j].Position
	})
	return results, nil
}

// GetSessionResultsBySeason returns every cached session result for the
// season (across all rounds and session types). The calendar service uses
// this to compute running championship totals from race+sprint points.
func (c *Client) GetSessionResultsBySeason(ctx context.Context, season int) ([]storage.SessionResult, error) {
	pk := azcosmos.NewPartitionKeyNumber(float64(season))
	query := "SELECT * FROM c WHERE c.season = @season AND c.type = 'session_result'"
	queryOpts := &azcosmos.QueryOptions{
		QueryParameters: []azcosmos.QueryParameter{
			{Name: "@season", Value: season},
		},
	}

	pager := c.sessions.NewQueryItemsPager(query, pk, queryOpts)
	var results []storage.SessionResult

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("cosmos: query season session results: %w", err)
		}
		for _, item := range resp.Items {
			var r storage.SessionResult
			if err := json.Unmarshal(item, &r); err != nil {
				return nil, fmt.Errorf("cosmos: unmarshal session result: %w", err)
			}
			results = append(results, r)
		}
	}
	return results, nil
}

// GetFinalizedSessionKeys returns a map of session_key → schema_version for
// every cached session in the season where finalized=true. The poller uses
// this to skip re-fetching results/drivers/laps for sessions that have
// already been fully cached.
func (c *Client) GetFinalizedSessionKeys(ctx context.Context, season int) (map[int]int, error) {
	pk := azcosmos.NewPartitionKeyNumber(float64(season))
	query := "SELECT c.session_key, c.schema_version FROM c WHERE c.season = @season AND c.type = 'session' AND c.finalized = true"
	queryOpts := &azcosmos.QueryOptions{
		QueryParameters: []azcosmos.QueryParameter{
			{Name: "@season", Value: season},
		},
	}

	pager := c.sessions.NewQueryItemsPager(query, pk, queryOpts)
	out := make(map[int]int)

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("cosmos: query finalized sessions: %w", err)
		}
		for _, item := range resp.Items {
			var row struct {
				SessionKey    int `json:"session_key"`
				SchemaVersion int `json:"schema_version"`
			}
			if err := json.Unmarshal(item, &row); err != nil {
				return nil, fmt.Errorf("cosmos: unmarshal finalized session row: %w", err)
			}
			out[row.SessionKey] = row.SchemaVersion
		}
	}
	return out, nil
}

// GetCompletedRaceSessionKeys returns session_key values for race and sprint
// sessions whose date_end_utc is before the given time. This provides a
// time-based filter for the progression chart that doesn't depend on the
// finalized flag (which requires schema_version alignment).
func (c *Client) GetCompletedRaceSessionKeys(ctx context.Context, season int, now time.Time) (map[int]struct{}, error) {
	pk := azcosmos.NewPartitionKeyNumber(float64(season))
	query := "SELECT c.session_key FROM c WHERE c.season = @season AND c.type = 'session' AND c.session_type IN ('race', 'sprint') AND c.date_end_utc < @now"
	queryOpts := &azcosmos.QueryOptions{
		QueryParameters: []azcosmos.QueryParameter{
			{Name: "@season", Value: season},
			{Name: "@now", Value: now.Format(time.RFC3339)},
		},
	}

	pager := c.sessions.NewQueryItemsPager(query, pk, queryOpts)
	out := make(map[int]struct{})

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("cosmos: query completed race session keys: %w", err)
		}
		for _, item := range resp.Items {
			var row struct {
				SessionKey int `json:"session_key"`
			}
			if err := json.Unmarshal(item, &row); err != nil {
				return nil, fmt.Errorf("cosmos: unmarshal completed race session key: %w", err)
			}
			out[row.SessionKey] = struct{}{}
		}
	}
	return out, nil
}

// GetCompletedRaceSessions returns full session documents for race and sprint
// sessions whose date_end_utc is before the given time. Used by the standings
// progression to build descriptive round labels from session metadata.
func (c *Client) GetCompletedRaceSessions(ctx context.Context, season int, now time.Time) ([]storage.Session, error) {
	pk := azcosmos.NewPartitionKeyNumber(float64(season))
	query := "SELECT * FROM c WHERE c.season = @season AND c.type = 'session' AND c.session_type IN ('race', 'sprint') AND c.date_end_utc < @now"
	queryOpts := &azcosmos.QueryOptions{
		QueryParameters: []azcosmos.QueryParameter{
			{Name: "@season", Value: season},
			{Name: "@now", Value: now.Format(time.RFC3339)},
		},
	}

	pager := c.sessions.NewQueryItemsPager(query, pk, queryOpts)
	var sessions []storage.Session

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("cosmos: query completed race sessions: %w", err)
		}
		for _, item := range resp.Items {
			var s storage.Session
			if err := json.Unmarshal(item, &s); err != nil {
				return nil, fmt.Errorf("cosmos: unmarshal completed race session: %w", err)
			}
			sessions = append(sessions, s)
		}
	}
	return sessions, nil
}

// GetFinalizedSessions returns all session documents for the season where
// finalized=true. Used by the backfill CLI.
func (c *Client) GetFinalizedSessions(ctx context.Context, season int) ([]storage.Session, error) {
	pk := azcosmos.NewPartitionKeyNumber(float64(season))
	query := "SELECT * FROM c WHERE c.season = @season AND c.type = 'session' AND c.finalized = true"
	queryOpts := &azcosmos.QueryOptions{
		QueryParameters: []azcosmos.QueryParameter{
			{Name: "@season", Value: season},
		},
	}

	pager := c.sessions.NewQueryItemsPager(query, pk, queryOpts)
	var sessions []storage.Session

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("cosmos: query finalized sessions: %w", err)
		}
		for _, item := range resp.Items {
			var s storage.Session
			if err := json.Unmarshal(item, &s); err != nil {
				return nil, fmt.Errorf("cosmos: unmarshal session: %w", err)
			}
			sessions = append(sessions, s)
		}
	}
	return sessions, nil
}

// DeleteSession removes a session document by its ID.
func (c *Client) DeleteSession(ctx context.Context, season int, id string) error {
	pk := azcosmos.NewPartitionKeyNumber(float64(season))
	_, err := c.sessions.DeleteItem(ctx, pk, id, nil)
	if err != nil {
		return fmt.Errorf("cosmos: delete session %s: %w", id, err)
	}
	return nil
}

// DeleteSessionResultsBySessionType removes all session_result documents for
// a given season, round, and session_type. Used during stale-session cleanup
// to remove orphaned results when a session no longer exists upstream.
func (c *Client) DeleteSessionResultsBySessionType(ctx context.Context, season, round int, sessionType string) error {
	pk := azcosmos.NewPartitionKeyNumber(float64(season))
	query := "SELECT c.id FROM c WHERE c.season = @season AND c.round = @round AND c.session_type = @sessionType AND c.type = 'session_result'"
	queryOpts := &azcosmos.QueryOptions{
		QueryParameters: []azcosmos.QueryParameter{
			{Name: "@season", Value: season},
			{Name: "@round", Value: round},
			{Name: "@sessionType", Value: sessionType},
		},
	}

	pager := c.sessions.NewQueryItemsPager(query, pk, queryOpts)
	var ids []string
	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("cosmos: query session results for delete: %w", err)
		}
		for _, item := range resp.Items {
			var row struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(item, &row); err != nil {
				return fmt.Errorf("cosmos: unmarshal result id: %w", err)
			}
			ids = append(ids, row.ID)
		}
	}

	for _, id := range ids {
		if _, err := c.sessions.DeleteItem(ctx, pk, id, nil); err != nil {
			return fmt.Errorf("cosmos: delete session result %s: %w", id, err)
		}
	}
	return nil
}
