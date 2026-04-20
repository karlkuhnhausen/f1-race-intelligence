package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/domain"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// SessionPoller polls the OpenF1 sessions and positions endpoints and upserts
// session metadata and results into the SessionRepository.
type SessionPoller struct {
	repo     storage.SessionRepository
	client   *http.Client
	interval time.Duration
	logger   *slog.Logger

	mu       sync.Mutex
	lastPoll time.Time
	lastErr  error
}

// NewSessionPoller creates a new session poller.
func NewSessionPoller(repo storage.SessionRepository, logger *slog.Logger) *SessionPoller {
	return &SessionPoller{
		repo:     repo,
		client:   &http.Client{Timeout: 30 * time.Second},
		interval: DefaultPollInterval,
		logger:   logger,
	}
}

// Start begins the polling loop. It blocks until the context is cancelled.
func (p *SessionPoller) Start(ctx context.Context, season int) {
	p.logger.Info("session poller starting", "season", season, "interval", p.interval)

	p.poll(ctx, season)

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("session poller stopped")
			return
		case <-ticker.C:
			p.poll(ctx, season)
		}
	}
}

func (p *SessionPoller) poll(ctx context.Context, season int) {
	p.logger.Debug("session poll starting", "season", season)

	sessions, err := p.fetchSessions(ctx, season)
	if err != nil {
		p.setErr(err)
		p.logger.Error("session poll: fetch sessions failed", "error", err)
		return
	}

	// Build meeting_key -> round mapping from sessions
	meetingRounds := p.buildMeetingRoundMap(sessions)

	for i, raw := range sessions {
		round, ok := meetingRounds[raw.MeetingKey]
		if !ok {
			continue
		}

		// Rate-limit: pause between sessions to avoid OpenF1 429s
		if i > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(500 * time.Millisecond):
			}
		}

		sess := TransformSession(raw, season, round)
		if err := p.repo.UpsertSession(ctx, sess); err != nil {
			p.logger.Error("session upsert failed", "session_id", sess.ID, "error", err)
			continue
		}

		// Fetch positions for this session
		sessionType := domain.MapOpenF1SessionType(raw.SessionName)
		if err := p.fetchAndUpsertResults(ctx, raw, sessionType, season, round); err != nil {
			p.logger.Error("session results failed", "session_key", raw.SessionKey, "error", err)
		}
	}

	p.mu.Lock()
	p.lastPoll = time.Now().UTC()
	p.lastErr = nil
	p.mu.Unlock()

	p.logger.Info("session poll complete", "season", season, "sessions", len(sessions))
}

func (p *SessionPoller) buildMeetingRoundMap(sessions []openF1Session) map[int]int {
	// Group sessions by meeting_key, find the earliest meeting keys and assign round numbers
	type meetingInfo struct {
		key       int
		dateStart string
	}
	seen := map[int]bool{}
	var meetings []meetingInfo
	for _, s := range sessions {
		if !seen[s.MeetingKey] {
			seen[s.MeetingKey] = true
			meetings = append(meetings, meetingInfo{key: s.MeetingKey, dateStart: s.DateStart})
		}
	}

	// Sort by date_start to assign round numbers
	for i := 0; i < len(meetings); i++ {
		for j := i + 1; j < len(meetings); j++ {
			if meetings[j].dateStart < meetings[i].dateStart {
				meetings[i], meetings[j] = meetings[j], meetings[i]
			}
		}
	}

	result := make(map[int]int, len(meetings))
	for i, m := range meetings {
		result[m.key] = i + 1
	}
	return result
}

func (p *SessionPoller) fetchAndUpsertResults(ctx context.Context, raw openF1Session, sessionType domain.SessionType, season, round int) error {
	// Fetch positions
	positions, err := p.fetchPositions(ctx, raw.SessionKey)
	if err != nil {
		return fmt.Errorf("fetch positions: %w", err)
	}

	// Rate-limit before driver fetch
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(300 * time.Millisecond):
	}

	// Fetch drivers for enrichment
	drivers, err := p.fetchDrivers(ctx, raw.SessionKey)
	if err != nil {
		p.logger.Warn("session drivers fetch failed, continuing without enrichment", "error", err)
		drivers = nil
	}

	driverMap := make(map[int]*openF1Driver, len(drivers))
	for i := range drivers {
		driverMap[drivers[i].DriverNumber] = &drivers[i]
	}

	// Get final positions only (the last position entry per driver)
	finalPositions := p.getFinalPositions(positions)

	for _, pos := range finalPositions {
		driver := driverMap[pos.DriverNumber]
		result := TransformSessionResult(pos, driver, sessionType, season, round, 0)

		// Skip upsert if driver data is missing — don't overwrite good data with empty fields
		if driver == nil || driver.FullName == "" {
			p.logger.Debug("skipping result upsert: no driver data", "driver_number", pos.DriverNumber, "session_key", raw.SessionKey)
			continue
		}

		if err := p.repo.UpsertSessionResult(ctx, result); err != nil {
			p.logger.Error("session result upsert failed", "result_id", result.ID, "error", err)
		}
	}

	return nil
}

// getFinalPositions returns the last position entry for each driver (the final classification).
func (p *SessionPoller) getFinalPositions(positions []openF1Position) []openF1Position {
	latest := make(map[int]openF1Position)
	for _, pos := range positions {
		if existing, ok := latest[pos.DriverNumber]; !ok || pos.Date > existing.Date {
			latest[pos.DriverNumber] = pos
		}
	}

	result := make([]openF1Position, 0, len(latest))
	for _, pos := range latest {
		result = append(result, pos)
	}
	return result
}

func (p *SessionPoller) fetchSessions(ctx context.Context, season int) ([]openF1Session, error) {
	url := fmt.Sprintf("%s/sessions?year=%d", openF1BaseURL, season)

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
		return nil, fmt.Errorf("openf1: sessions unexpected status %d", resp.StatusCode)
	}

	var raw []openF1Session
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("openf1: decode sessions: %w", err)
	}
	return raw, nil
}

func (p *SessionPoller) fetchPositions(ctx context.Context, sessionKey int) ([]openF1Position, error) {
	url := fmt.Sprintf("%s/position?session_key=%d", openF1BaseURL, sessionKey)

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
		return nil, fmt.Errorf("openf1: positions unexpected status %d", resp.StatusCode)
	}

	var raw []openF1Position
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("openf1: decode positions: %w", err)
	}
	return raw, nil
}

func (p *SessionPoller) fetchDrivers(ctx context.Context, sessionKey int) ([]openF1Driver, error) {
	url := fmt.Sprintf("%s/drivers?session_key=%d", openF1BaseURL, sessionKey)

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
		return nil, fmt.Errorf("openf1: drivers unexpected status %d", resp.StatusCode)
	}

	var raw []openF1Driver
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("openf1: decode drivers: %w", err)
	}
	return raw, nil
}

func (p *SessionPoller) setErr(err error) {
	p.mu.Lock()
	p.lastErr = err
	p.mu.Unlock()
}

// LastPoll returns the time of the last successful poll and any error.
func (p *SessionPoller) LastPoll() (time.Time, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.lastPoll, p.lastErr
}
