package cosmos

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// --- AnalysisRepository ---

func (c *Client) UpsertSessionPositions(ctx context.Context, positions []storage.SessionAnalysisPosition) error {
	for _, p := range positions {
		data, err := json.Marshal(p)
		if err != nil {
			return fmt.Errorf("cosmos: marshal analysis position: %w", err)
		}
		pk := azcosmos.NewPartitionKeyNumber(float64(p.Season))
		if _, err = c.sessions.UpsertItem(ctx, pk, data, nil); err != nil {
			return fmt.Errorf("cosmos: upsert analysis position %s: %w", p.ID, err)
		}
	}
	return nil
}

func (c *Client) UpsertSessionIntervals(ctx context.Context, intervals []storage.SessionAnalysisInterval) error {
	for _, iv := range intervals {
		data, err := json.Marshal(iv)
		if err != nil {
			return fmt.Errorf("cosmos: marshal analysis interval: %w", err)
		}
		pk := azcosmos.NewPartitionKeyNumber(float64(iv.Season))
		if _, err = c.sessions.UpsertItem(ctx, pk, data, nil); err != nil {
			return fmt.Errorf("cosmos: upsert analysis interval %s: %w", iv.ID, err)
		}
	}
	return nil
}

func (c *Client) UpsertSessionStints(ctx context.Context, stints []storage.SessionAnalysisStint) error {
	for _, s := range stints {
		data, err := json.Marshal(s)
		if err != nil {
			return fmt.Errorf("cosmos: marshal analysis stint: %w", err)
		}
		pk := azcosmos.NewPartitionKeyNumber(float64(s.Season))
		if _, err = c.sessions.UpsertItem(ctx, pk, data, nil); err != nil {
			return fmt.Errorf("cosmos: upsert analysis stint %s: %w", s.ID, err)
		}
	}
	return nil
}

func (c *Client) UpsertSessionPits(ctx context.Context, pits []storage.SessionAnalysisPit) error {
	for _, p := range pits {
		data, err := json.Marshal(p)
		if err != nil {
			return fmt.Errorf("cosmos: marshal analysis pit: %w", err)
		}
		pk := azcosmos.NewPartitionKeyNumber(float64(p.Season))
		if _, err = c.sessions.UpsertItem(ctx, pk, data, nil); err != nil {
			return fmt.Errorf("cosmos: upsert analysis pit %s: %w", p.ID, err)
		}
	}
	return nil
}

func (c *Client) UpsertSessionOvertakes(ctx context.Context, overtakes []storage.SessionAnalysisOvertake) error {
	for _, o := range overtakes {
		data, err := json.Marshal(o)
		if err != nil {
			return fmt.Errorf("cosmos: marshal analysis overtake: %w", err)
		}
		pk := azcosmos.NewPartitionKeyNumber(float64(o.Season))
		if _, err = c.sessions.UpsertItem(ctx, pk, data, nil); err != nil {
			return fmt.Errorf("cosmos: upsert analysis overtake %s: %w", o.ID, err)
		}
	}
	return nil
}

func (c *Client) GetSessionAnalysis(ctx context.Context, season, round int, sessionType string) (*storage.SessionAnalysisData, error) {
	pk := azcosmos.NewPartitionKeyNumber(float64(season))
	query := `SELECT * FROM c WHERE c.season = @season AND c.round = @round AND c.session_type = @sessionType AND STARTSWITH(c.type, 'analysis_')`
	queryOpts := &azcosmos.QueryOptions{
		QueryParameters: []azcosmos.QueryParameter{
			{Name: "@season", Value: season},
			{Name: "@round", Value: round},
			{Name: "@sessionType", Value: sessionType},
		},
	}

	pager := c.sessions.NewQueryItemsPager(query, pk, queryOpts)

	result := &storage.SessionAnalysisData{}
	found := false

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("cosmos: query analysis data: %w", err)
		}
		for _, item := range resp.Items {
			found = true
			// Peek at "type" to route deserialization
			var peek struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(item, &peek); err != nil {
				continue
			}
			switch peek.Type {
			case "analysis_position":
				var doc storage.SessionAnalysisPosition
				if err := json.Unmarshal(item, &doc); err == nil {
					result.Positions = append(result.Positions, doc)
				}
			case "analysis_interval":
				var doc storage.SessionAnalysisInterval
				if err := json.Unmarshal(item, &doc); err == nil {
					result.Intervals = append(result.Intervals, doc)
				}
			case "analysis_stint":
				var doc storage.SessionAnalysisStint
				if err := json.Unmarshal(item, &doc); err == nil {
					result.Stints = append(result.Stints, doc)
				}
			case "analysis_pit":
				var doc storage.SessionAnalysisPit
				if err := json.Unmarshal(item, &doc); err == nil {
					result.Pits = append(result.Pits, doc)
				}
			case "analysis_overtake":
				var doc storage.SessionAnalysisOvertake
				if err := json.Unmarshal(item, &doc); err == nil {
					result.Overtakes = append(result.Overtakes, doc)
				}
			}
		}
	}

	if !found {
		return nil, nil
	}
	return result, nil
}

func (c *Client) HasAnalysisData(ctx context.Context, season, round int, sessionType string) (bool, error) {
	pk := azcosmos.NewPartitionKeyNumber(float64(season))
	query := `SELECT VALUE COUNT(1) FROM c WHERE c.season = @season AND c.round = @round AND c.session_type = @sessionType AND c.type = 'analysis_position'`
	queryOpts := &azcosmos.QueryOptions{
		QueryParameters: []azcosmos.QueryParameter{
			{Name: "@season", Value: season},
			{Name: "@round", Value: round},
			{Name: "@sessionType", Value: sessionType},
		},
	}

	pager := c.sessions.NewQueryItemsPager(query, pk, queryOpts)
	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return false, fmt.Errorf("cosmos: count analysis data: %w", err)
		}
		for _, item := range resp.Items {
			var count int
			if err := json.Unmarshal(item, &count); err == nil && count > 0 {
				return true, nil
			}
		}
	}
	return false, nil
}

// DeleteAnalysisData deletes all analysis documents for a given season/round/sessionType.
func (c *Client) DeleteAnalysisData(ctx context.Context, season, round int, sessionType string) (int, error) {
	pk := azcosmos.NewPartitionKeyNumber(float64(season))
	query := `SELECT c.id FROM c WHERE c.season = @season AND c.round = @round AND c.session_type = @sessionType AND STARTSWITH(c.type, 'analysis_')`
	queryOpts := &azcosmos.QueryOptions{
		QueryParameters: []azcosmos.QueryParameter{
			{Name: "@season", Value: season},
			{Name: "@round", Value: round},
			{Name: "@sessionType", Value: sessionType},
		},
	}

	pager := c.sessions.NewQueryItemsPager(query, pk, queryOpts)
	deleted := 0

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return deleted, fmt.Errorf("cosmos: query analysis docs for delete: %w", err)
		}
		for _, item := range resp.Items {
			var doc struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(item, &doc); err != nil {
				continue
			}
			if _, err := c.sessions.DeleteItem(ctx, pk, doc.ID, nil); err != nil {
				return deleted, fmt.Errorf("cosmos: delete analysis doc %s: %w", doc.ID, err)
			}
			deleted++
		}
	}
	return deleted, nil
}
