// Package cosmos implements Cosmos DB storage using the Azure SDK.
package cosmos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

const (
	databaseName       = "f1raceintelligence"
	meetingsContainer  = "meetings"
	standingsContainer = "standings"
	sessionsContainer  = "sessions"
)

// Client wraps the azcosmos client and provides repository implementations.
type Client struct {
	db        *azcosmos.DatabaseClient
	meetings  *azcosmos.ContainerClient
	standings *azcosmos.ContainerClient
	sessions  *azcosmos.ContainerClient
}

// NewClient creates a Cosmos DB client using DefaultAzureCredential (supports Managed Identity + local dev).
func NewClient(endpoint string) (*Client, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("cosmos: credential: %w", err)
	}

	client, err := azcosmos.NewClient(endpoint, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("cosmos: client: %w", err)
	}

	db, err := client.NewDatabase(databaseName)
	if err != nil {
		return nil, fmt.Errorf("cosmos: database: %w", err)
	}

	meetings, err := db.NewContainer(meetingsContainer)
	if err != nil {
		return nil, fmt.Errorf("cosmos: meetings container: %w", err)
	}

	standings, err := db.NewContainer(standingsContainer)
	if err != nil {
		return nil, fmt.Errorf("cosmos: standings container: %w", err)
	}

	sessions, err := db.NewContainer(sessionsContainer)
	if err != nil {
		return nil, fmt.Errorf("cosmos: sessions container: %w", err)
	}

	return &Client{
		db:        db,
		meetings:  meetings,
		standings: standings,
		sessions:  sessions,
	}, nil
}

// --- CalendarRepository ---

func (c *Client) UpsertMeeting(ctx context.Context, m storage.RaceMeeting) error {
	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("cosmos: marshal meeting: %w", err)
	}

	pk := azcosmos.NewPartitionKeyNumber(float64(m.Season))
	_, err = c.meetings.UpsertItem(ctx, pk, data, nil)
	if err != nil {
		return fmt.Errorf("cosmos: upsert meeting %s: %w", m.ID, err)
	}
	return nil
}

func (c *Client) GetMeetingsBySeason(ctx context.Context, season int) ([]storage.RaceMeeting, error) {
	pk := azcosmos.NewPartitionKeyNumber(float64(season))
	query := "SELECT * FROM c WHERE c.season = @season ORDER BY c.round"
	queryOpts := &azcosmos.QueryOptions{
		QueryParameters: []azcosmos.QueryParameter{
			{Name: "@season", Value: season},
		},
	}

	pager := c.meetings.NewQueryItemsPager(query, pk, queryOpts)
	var meetings []storage.RaceMeeting

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("cosmos: query meetings: %w", err)
		}
		for _, item := range resp.Items {
			var m storage.RaceMeeting
			if err := json.Unmarshal(item, &m); err != nil {
				return nil, fmt.Errorf("cosmos: unmarshal meeting: %w", err)
			}
			meetings = append(meetings, m)
		}
	}
	return meetings, nil
}

func (c *Client) GetMeetingByID(ctx context.Context, season int, id string) (*storage.RaceMeeting, error) {
	pk := azcosmos.NewPartitionKeyNumber(float64(season))
	resp, err := c.meetings.ReadItem(ctx, pk, id, nil)
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("cosmos: read meeting %s: %w", id, err)
	}

	var m storage.RaceMeeting
	if err := json.Unmarshal(resp.Value, &m); err != nil {
		return nil, fmt.Errorf("cosmos: unmarshal meeting: %w", err)
	}
	return &m, nil
}

// --- StandingsRepository ---

func (c *Client) UpsertDriverStandings(ctx context.Context, rows []storage.DriverStandingRow) error {
	for _, row := range rows {
		data, err := json.Marshal(row)
		if err != nil {
			return fmt.Errorf("cosmos: marshal driver standing: %w", err)
		}
		pk := azcosmos.NewPartitionKeyNumber(float64(row.Season))
		if _, err = c.standings.UpsertItem(ctx, pk, data, nil); err != nil {
			return fmt.Errorf("cosmos: upsert driver standing %s: %w", row.ID, err)
		}
	}
	return nil
}

func (c *Client) GetDriverStandings(ctx context.Context, season int) ([]storage.DriverStandingRow, error) {
	pk := azcosmos.NewPartitionKeyNumber(float64(season))
	query := "SELECT * FROM c WHERE c.season = @season AND c.source = 'hyprace' AND IS_DEFINED(c.driver_name) ORDER BY c.position"
	queryOpts := &azcosmos.QueryOptions{
		QueryParameters: []azcosmos.QueryParameter{
			{Name: "@season", Value: season},
		},
	}

	pager := c.standings.NewQueryItemsPager(query, pk, queryOpts)
	var rows []storage.DriverStandingRow

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("cosmos: query driver standings: %w", err)
		}
		for _, item := range resp.Items {
			var row storage.DriverStandingRow
			if err := json.Unmarshal(item, &row); err != nil {
				return nil, fmt.Errorf("cosmos: unmarshal driver standing: %w", err)
			}
			rows = append(rows, row)
		}
	}
	return rows, nil
}

func (c *Client) UpsertConstructorStandings(ctx context.Context, rows []storage.ConstructorStandingRow) error {
	for _, row := range rows {
		data, err := json.Marshal(row)
		if err != nil {
			return fmt.Errorf("cosmos: marshal constructor standing: %w", err)
		}
		pk := azcosmos.NewPartitionKeyNumber(float64(row.Season))
		if _, err = c.standings.UpsertItem(ctx, pk, data, nil); err != nil {
			return fmt.Errorf("cosmos: upsert constructor standing %s: %w", row.ID, err)
		}
	}
	return nil
}

func (c *Client) GetConstructorStandings(ctx context.Context, season int) ([]storage.ConstructorStandingRow, error) {
	pk := azcosmos.NewPartitionKeyNumber(float64(season))
	query := "SELECT * FROM c WHERE c.season = @season AND c.source = 'hyprace' AND IS_DEFINED(c.team_name) AND NOT IS_DEFINED(c.driver_name) ORDER BY c.position"
	queryOpts := &azcosmos.QueryOptions{
		QueryParameters: []azcosmos.QueryParameter{
			{Name: "@season", Value: season},
		},
	}

	pager := c.standings.NewQueryItemsPager(query, pk, queryOpts)
	var rows []storage.ConstructorStandingRow

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("cosmos: query constructor standings: %w", err)
		}
		for _, item := range resp.Items {
			var row storage.ConstructorStandingRow
			if err := json.Unmarshal(item, &row); err != nil {
				return nil, fmt.Errorf("cosmos: unmarshal constructor standing: %w", err)
			}
			rows = append(rows, row)
		}
	}
	return rows, nil
}

// isNotFound checks if an error indicates a 404 status.
func isNotFound(err error) bool {
	var respErr *azcore.ResponseError
	if errors.As(err, &respErr) {
		return respErr.StatusCode == http.StatusNotFound
	}
	return false
}
