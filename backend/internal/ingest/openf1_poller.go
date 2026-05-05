// Package ingest provides scheduled polling of upstream F1 data sources.
package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

const (
	// DefaultPollInterval is the default polling cadence per research decision 1.
	DefaultPollInterval = 5 * time.Minute

	// openF1BaseURL is the public OpenF1 API base.
	openF1BaseURL = "https://api.openf1.org/v1"
)

// openF1Meeting is the raw upstream meeting shape from OpenF1.
type openF1Meeting struct {
	MeetingKey  int    `json:"meeting_key"`
	MeetingName string `json:"meeting_name"`
	CircuitKey  int    `json:"circuit_key"`
	CircuitName string `json:"circuit_short_name"`
	CountryName string `json:"country_name"`
	DateStart   string `json:"date_start"`
	DateEnd     string `json:"date_end"`
	IsCancelled bool   `json:"is_cancelled"`
	Year        int    `json:"year"`
}

// OpenF1Poller polls the OpenF1 meetings endpoint on a fixed cadence
// and upserts results into the CalendarRepository.
type OpenF1Poller struct {
	repo     storage.CalendarRepository
	client   *http.Client
	interval time.Duration
	logger   *slog.Logger

	mu       sync.Mutex
	lastPoll time.Time
	lastErr  error
}

// NewOpenF1Poller creates a new poller with the given repository and options.
func NewOpenF1Poller(repo storage.CalendarRepository, logger *slog.Logger) *OpenF1Poller {
	return &OpenF1Poller{
		repo:     repo,
		client:   &http.Client{Timeout: 30 * time.Second},
		interval: DefaultPollInterval,
		logger:   logger,
	}
}

// Start begins the polling loop. It blocks until the context is cancelled.
func (p *OpenF1Poller) Start(ctx context.Context, season int) {
	p.logger.Info("openf1 poller starting", "season", season, "interval", p.interval)

	// Run immediately on start, then on ticker.
	p.poll(ctx, season)

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("openf1 poller stopped")
			return
		case <-ticker.C:
			p.poll(ctx, season)
		}
	}
}

// poll fetches meetings from OpenF1 and upserts them.
func (p *OpenF1Poller) poll(ctx context.Context, season int) {
	p.logger.Debug("openf1 poll starting", "season", season)

	meetings, err := p.fetchMeetings(ctx, season)
	if err != nil {
		p.mu.Lock()
		p.lastErr = err
		p.mu.Unlock()
		p.logger.Error("openf1 poll failed", "error", err)
		return
	}

	now := time.Now().UTC()
	for _, m := range meetings {
		if err := p.repo.UpsertMeeting(ctx, m); err != nil {
			p.logger.Error("openf1 upsert failed", "meeting_id", m.ID, "error", err)
		}
	}

	p.mu.Lock()
	p.lastPoll = now
	p.lastErr = nil
	p.mu.Unlock()

	p.logger.Info("openf1 poll complete", "season", season, "meetings", len(meetings))
}

// fetchMeetings calls the OpenF1 meetings endpoint for a season.
func (p *OpenF1Poller) fetchMeetings(ctx context.Context, season int) ([]storage.RaceMeeting, error) {
	url := fmt.Sprintf("%s/meetings?year=%d", openF1BaseURL, season)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("openf1: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openf1: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openf1: unexpected status %d", resp.StatusCode)
	}

	var raw []openF1Meeting
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("openf1: decode: %w", err)
	}

	// Delegate normalization (pre-season testing filter + sequential round
	// numbering) to NormalizeMeetings so the ingest write path and tests
	// share one implementation. Round 1 is the first non-testing meeting.
	return NormalizeMeetings(raw, season), nil
}

// LastPoll returns the time of the last successful poll and any error from the most recent attempt.
func (p *OpenF1Poller) LastPoll() (time.Time, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.lastPoll, p.lastErr
}
