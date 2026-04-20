package cosmos

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

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
