package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/domain"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// errNoResultsYet is returned by fetchSessionResults when OpenF1 responds 404,
// which means the session has no published results yet (cancelled, not yet
// run, or test session). It is treated as a benign skip rather than an error.
var errNoResultsYet = errors.New("openf1: no results yet (404)")

// errNoDriverData is returned by fetchAndUpsertResults when driver enrichment
// fails and all results are skipped to avoid overwriting good data with empty
// fields. The session should NOT be finalized when this occurs so it retries
// on the next poll cycle.
var errNoDriverData = errors.New("openf1: driver data unavailable, results skipped")

// SessionSchemaVersion is the current layout version of cached storage.Session
// documents. Bump this whenever new fields are added to the session or
// session_result documents that should trigger a re-fetch from OpenF1.
// Cached documents whose schema_version is below this constant are treated
// as not-finalized and re-polled.
//
// Version history:
//
//	1 — initial schema.
//	2 — round numbers recalculated after pre-season testing exclusion.
//	    All previously-finalized sessions must re-fetch to fix round mapping.
//	3 — fix: sessions finalized without results due to driver-fetch failure
//	    are re-polled to acquire results.
const SessionSchemaVersion = 3

// finalizationBuffer is how long after a session's DateEndUTC we wait
// before marking it finalized in the cache. This gives OpenF1 time to
// publish official final results (penalties, classification changes, etc.).
const finalizationBuffer = 2 * time.Hour

// ChampionshipHook is called after a Race or Sprint session is finalized
// to ingest championship data from OpenF1.
type ChampionshipHook interface {
	IngestSession(ctx context.Context, season, sessionKey, meetingKey int) error
}

// SessionPoller polls the OpenF1 sessions, session_result, drivers, and laps
// endpoints, transforms the data, and upserts session metadata + per-driver
// results into the SessionRepository.
type SessionPoller struct {
	repo             storage.SessionRepository
	client           *http.Client
	interval         time.Duration
	logger           *slog.Logger
	forceRefresh     bool
	championshipHook ChampionshipHook

	mu       sync.Mutex
	lastPoll time.Time
	lastErr  error
}

// NewSessionPoller creates a new session poller.
//
// If the INGEST_FORCE_REFRESH_SESSIONS env var is set to a truthy value
// ("1", "true"), the finalized-session skip optimization is bypassed and
// every poll re-fetches all sessions. Use this to backfill new fields
// after a SessionSchemaVersion bump or to recover from corrupted cache
// state.
func NewSessionPoller(repo storage.SessionRepository, logger *slog.Logger) *SessionPoller {
	return &SessionPoller{
		repo:         repo,
		client:       &http.Client{Timeout: 30 * time.Second},
		interval:     DefaultPollInterval,
		logger:       logger,
		forceRefresh: isTruthy(os.Getenv("INGEST_FORCE_REFRESH_SESSIONS")),
	}
}

// SetChampionshipHook registers a hook that is called after Race or Sprint
// sessions are finalized, to ingest championship standings data.
func (p *SessionPoller) SetChampionshipHook(hook ChampionshipHook) {
	p.championshipHook = hook
}

func isTruthy(v string) bool {
	switch v {
	case "1", "true", "TRUE", "yes", "YES":
		return true
	}
	return false
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

	// Pre-load the set of sessions already finalized in cache (best-effort).
	// Sessions whose cached schema_version matches the current code version
	// are skipped entirely — no metadata upsert, no results/drivers/laps fetch.
	finalizedKeys := map[int]int{}
	if !p.forceRefresh {
		fk, ferr := p.repo.GetFinalizedSessionKeys(ctx, season)
		if ferr != nil {
			p.logger.Warn("session poll: load finalized keys failed; proceeding without skip", "error", ferr)
		} else {
			finalizedKeys = fk
		}
	}

	meetingRounds := p.buildMeetingRoundMap(sessions)

	skipped := 0
	skippedCancelled := 0
	skippedFuture := 0
	skippedNoResults := 0
	processed := 0
	now := time.Now().UTC()
	for _, raw := range sessions {
		round, ok := meetingRounds[raw.MeetingKey]
		if !ok {
			continue
		}

		// Skip cancelled sessions — OpenF1 won't have results for them and
		// they would otherwise generate a 404 on every poll cycle.
		if raw.IsCancelled {
			skippedCancelled++
			continue
		}

		future := isFutureSession(raw, now, p.logger)

		// Always upsert session metadata so that upcoming/in-progress
		// sessions appear in the round detail UI. Only the results fetch
		// is gated on the session having ended.
		sess := TransformSession(raw, season, round)
		if err := p.repo.UpsertSession(ctx, sess); err != nil {
			p.logger.Error("session upsert failed", "session_id", sess.ID, "error", err)
			continue
		}

		// Skip results fetch for sessions that haven't ended yet — no
		// results to fetch and hitting /session_result for future sessions
		// just earns 404s and rate-limit pressure.
		if future {
			skippedFuture++
			continue
		}

		// Skip sessions already fully cached at the current schema version.
		if cachedVer, isFinalized := finalizedKeys[raw.SessionKey]; isFinalized && cachedVer >= SessionSchemaVersion {
			skipped++
			continue
		}

		// Rate-limit: pause between sessions we actually fetch to avoid
		// OpenF1 429s. Skipped sessions cost no API calls.
		if processed > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(500 * time.Millisecond):
			}
		}
		processed++

		sessionType := domain.MapOpenF1SessionType(raw.SessionName)
		fastestLapTimeSecs, err := p.fetchAndUpsertResults(ctx, raw, sessionType, season, round)
		if err != nil {
			if errors.Is(err, errNoResultsYet) {
				// Benign: session exists in /sessions but has no published
				// /session_result yet. Will retry on the next poll cycle.
				skippedNoResults++
				p.logger.Debug("session results not yet published", "session_key", raw.SessionKey)
				continue
			}
			if errors.Is(err, errNoDriverData) {
				// Driver enrichment unavailable — results were skipped to
				// avoid overwriting good data. Do NOT finalize; retry next cycle.
				p.logger.Warn("session results skipped: driver data unavailable", "session_key", raw.SessionKey)
				continue
			}
			p.logger.Error("session results failed", "session_key", raw.SessionKey, "error", err)
			continue
		}

		// If the session ended long enough ago for OpenF1 data to settle,
		// mark it finalized so future polls skip it.
		if !sess.DateEndUTC.IsZero() && time.Since(sess.DateEndUTC) >= finalizationBuffer {
			now := time.Now().UTC()
			sess.Finalized = true
			sess.FinalizedAtUTC = &now

			// Fetch race control data for this session (500ms delay to respect rate limit).
			select {
			case <-ctx.Done():
				return
			case <-time.After(500 * time.Millisecond):
			}
			rcMsgs, rcErr := FetchRaceControlMsgs(ctx, p.client, raw.SessionKey)
			if rcErr != nil {
				p.logger.Warn("race control fetch failed during finalization; recap events unavailable",
					"session_id", sess.ID, "error", rcErr)
			} else {
				summary := SummarizeRaceControl(rcMsgs)
				sess.RaceControlSummary = &summary
			}

			// Store fastest lap time derived from the laps already fetched above.
			if fastestLapTimeSecs != nil {
				sess.FastestLapTimeSeconds = fastestLapTimeSecs
			}

			if err := p.repo.UpsertSession(ctx, sess); err != nil {
				p.logger.Error("session finalize upsert failed", "session_id", sess.ID, "error", err)
			} else {
				p.logger.Info("session finalized", "session_id", sess.ID, "session_key", raw.SessionKey)

				// Trigger championship ingestion for Race and Sprint sessions.
				if p.championshipHook != nil && (sessionType == domain.SessionRace || sessionType == domain.SessionSprint) {
					select {
					case <-ctx.Done():
						return
					case <-time.After(500 * time.Millisecond):
					}
					if err := p.championshipHook.IngestSession(ctx, season, raw.SessionKey, raw.MeetingKey); err != nil {
						p.logger.Error("championship ingestion failed", "session_key", raw.SessionKey, "error", err)
					}
				}
			}
		}
	}

	// Reconcile: remove stale session documents that no longer exist
	// upstream. This handles cases where round numbering changed (e.g.,
	// pre-season testing exclusion) and orphaned sessions remain in cache.
	p.reconcileStaleSessions(ctx, sessions, meetingRounds, season)

	p.mu.Lock()
	p.lastPoll = time.Now().UTC()
	p.lastErr = nil
	p.mu.Unlock()

	p.logger.Info("session poll complete",
		"season", season,
		"sessions", len(sessions),
		"processed", processed,
		"skipped_finalized", skipped,
		"skipped_cancelled", skippedCancelled,
		"skipped_future", skippedFuture,
		"skipped_no_results", skippedNoResults,
		"force_refresh", p.forceRefresh,
	)
}

func (p *SessionPoller) buildMeetingRoundMap(sessions []openF1Session) map[int]int {
	type meetingInfo struct {
		key       int
		dateStart string
	}
	// Collect candidate meeting keys (excluding any whose sessions look like
	// pre-season testing — e.g., session_name "Day 1/2/3"). Round 1 must be
	// the first real race, not a testing meeting.
	testingMeetings := map[int]bool{}
	for _, s := range sessions {
		if isTestingSessionName(s.SessionName) {
			testingMeetings[s.MeetingKey] = true
		}
	}

	seen := map[int]bool{}
	var meetings []meetingInfo
	for _, s := range sessions {
		if testingMeetings[s.MeetingKey] {
			continue
		}
		if !seen[s.MeetingKey] {
			seen[s.MeetingKey] = true
			meetings = append(meetings, meetingInfo{key: s.MeetingKey, dateStart: s.DateStart})
		}
	}

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

// reconcileStaleSessions compares cached sessions per round against what
// OpenF1 currently reports and deletes any cached sessions (and their results)
// that no longer exist upstream. This handles orphaned documents left behind
// by round-renumbering or format changes (e.g., sprint weekends).
func (p *SessionPoller) reconcileStaleSessions(ctx context.Context, upstreamSessions []openF1Session, meetingRounds map[int]int, season int) {
	// Build set of valid (round, session_type) pairs from upstream.
	type roundSession struct {
		round       int
		sessionType string
	}
	validSessions := make(map[roundSession]bool)
	for _, raw := range upstreamSessions {
		round, ok := meetingRounds[raw.MeetingKey]
		if !ok {
			continue
		}
		st := string(domain.MapOpenF1SessionType(raw.SessionName))
		validSessions[roundSession{round: round, sessionType: st}] = true
	}

	// Check each round that has cached sessions.
	roundsToCheck := make(map[int]bool)
	for _, round := range meetingRounds {
		roundsToCheck[round] = true
	}

	for round := range roundsToCheck {
		cached, err := p.repo.GetSessionsByRound(ctx, season, round)
		if err != nil {
			p.logger.Error("reconcile: failed to get cached sessions", "round", round, "error", err)
			continue
		}

		for _, sess := range cached {
			key := roundSession{round: round, sessionType: sess.SessionType}
			if !validSessions[key] {
				// This session no longer exists upstream — delete it and its results.
				p.logger.Info("reconcile: removing stale session",
					"session_id", sess.ID,
					"round", round,
					"session_type", sess.SessionType,
				)
				if err := p.repo.DeleteSession(ctx, season, sess.ID); err != nil {
					p.logger.Error("reconcile: delete session failed", "session_id", sess.ID, "error", err)
					continue
				}
				if err := p.repo.DeleteSessionResultsBySessionType(ctx, season, round, sess.SessionType); err != nil {
					p.logger.Error("reconcile: delete session results failed", "round", round, "session_type", sess.SessionType, "error", err)
				}
			}
		}
	}
}

func (p *SessionPoller) fetchAndUpsertResults(ctx context.Context, raw openF1Session, sessionType domain.SessionType, season, round int) (*float64, error) {
	rawResults, err := p.fetchSessionResults(ctx, raw.SessionKey)
	if err != nil {
		return nil, fmt.Errorf("fetch session_result: %w", err)
	}

	// Rate-limit before driver fetch
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(300 * time.Millisecond):
	}

	drivers, err := p.fetchDrivers(ctx, raw.SessionKey)
	if err != nil {
		p.logger.Warn("session drivers fetch failed, continuing without enrichment", "error", err)
		drivers = nil
	}

	driverMap := make(map[int]*openF1Driver, len(drivers))
	for i := range drivers {
		driverMap[drivers[i].DriverNumber] = &drivers[i]
	}

	// For race/sprint, fetch laps and derive the fastest-lap holder and fastest lap time.
	fastestLapDriver := 0
	hasFastest := false
	var fastestLapTimeSecs *float64
	if domain.IsRaceType(sessionType) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
		laps, err := p.fetchLaps(ctx, raw.SessionKey)
		if err != nil {
			p.logger.Warn("session laps fetch failed, fastest lap will be unset", "error", err)
		} else {
			fastestLapDriver, hasFastest = DeriveFastestLap(laps)
			// Derive fastest lap time: minimum non-nil LapDuration across all laps.
			for _, l := range laps {
				if l.LapDuration != nil && (fastestLapTimeSecs == nil || *l.LapDuration < *fastestLapTimeSecs) {
					val := *l.LapDuration
					fastestLapTimeSecs = &val
				}
			}
		}
	}

	for _, rr := range rawResults {
		driver := driverMap[rr.DriverNumber]
		result := TransformSessionResult(rr, driver, sessionType, season, round)

		if hasFastest && rr.DriverNumber == fastestLapDriver {
			t := true
			result.FastestLap = &t
		}

		// Skip upsert if driver data is missing — don't overwrite good data with empty fields.
		if driver == nil || driver.FullName == "" {
			p.logger.Debug("skipping result upsert: no driver data", "driver_number", rr.DriverNumber, "session_key", raw.SessionKey)
			continue
		}

		if err := p.repo.UpsertSessionResult(ctx, result); err != nil {
			p.logger.Error("session result upsert failed", "result_id", result.ID, "error", err)
		}
	}

	// If we had results from OpenF1 but skipped ALL of them due to missing
	// driver data, return an error so the session is not finalized. It will
	// be retried on the next poll cycle when drivers may be available.
	if len(rawResults) > 0 && len(driverMap) == 0 {
		return nil, errNoDriverData
	}

	return fastestLapTimeSecs, nil
}

func (p *SessionPoller) fetchSessions(ctx context.Context, season int) ([]openF1Session, error) {
	url := fmt.Sprintf("%s/sessions?year=%d", openF1BaseURL, season)
	var raw []openF1Session
	if err := p.fetchJSON(ctx, url, &raw); err != nil {
		return nil, fmt.Errorf("openf1 sessions: %w", err)
	}
	return raw, nil
}

func (p *SessionPoller) fetchSessionResults(ctx context.Context, sessionKey int) ([]openF1SessionResult, error) {
	url := fmt.Sprintf("%s/session_result?session_key=%d", openF1BaseURL, sessionKey)
	var raw []openF1SessionResult
	if err := p.fetchJSON(ctx, url, &raw); err != nil {
		// 404 means OpenF1 has no results published for this session yet.
		// Surface a sentinel so the caller can treat it as benign.
		if errors.Is(err, errHTTPNotFound) {
			return nil, errNoResultsYet
		}
		return nil, fmt.Errorf("openf1 session_result: %w", err)
	}
	return raw, nil
}

func (p *SessionPoller) fetchDrivers(ctx context.Context, sessionKey int) ([]openF1Driver, error) {
	url := fmt.Sprintf("%s/drivers?session_key=%d", openF1BaseURL, sessionKey)
	var raw []openF1Driver
	if err := p.fetchJSON(ctx, url, &raw); err != nil {
		return nil, fmt.Errorf("openf1 drivers: %w", err)
	}
	return raw, nil
}

func (p *SessionPoller) fetchLaps(ctx context.Context, sessionKey int) ([]openF1Lap, error) {
	url := fmt.Sprintf("%s/laps?session_key=%d", openF1BaseURL, sessionKey)
	var raw []openF1Lap
	if err := p.fetchJSON(ctx, url, &raw); err != nil {
		return nil, fmt.Errorf("openf1 laps: %w", err)
	}
	return raw, nil
}

// errHTTPNotFound is returned by fetchJSON when the upstream responds 404,
// allowing callers to distinguish "resource not found yet" from other errors.
var errHTTPNotFound = errors.New("upstream 404")

func (p *SessionPoller) fetchJSON(ctx context.Context, url string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return errHTTPNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	return nil
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

// isTestingSessionName reports whether a session_name from OpenF1 looks like
// a pre-season testing session (e.g., "Day 1", "Day 2", "Day 3"). Real race
// weekends use session names like "Practice 1", "Qualifying", "Sprint",
// "Race". Used to exclude testing meetings from the round numbering so
// Round 1 is the first race of the season.
func isTestingSessionName(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	return strings.HasPrefix(n, "day 1") ||
		strings.HasPrefix(n, "day 2") ||
		strings.HasPrefix(n, "day 3") ||
		strings.HasPrefix(n, "day-1") ||
		strings.HasPrefix(n, "day-2") ||
		strings.HasPrefix(n, "day-3")
}

// isFutureSession reports whether a raw OpenF1 session is scheduled in the
// future relative to now. It prefers date_end (session ended) but falls back
// to date_start when date_end is missing or unparseable.
//
// Returning true skips the session in the poll loop. When neither date is
// usable we conservatively treat the session as future to avoid writing
// hardcoded statuses for sessions that may not have happened yet.
func isFutureSession(raw openF1Session, now time.Time, logger *slog.Logger) bool {
	if raw.DateEnd != "" {
		if dateEnd, err := time.Parse(time.RFC3339, raw.DateEnd); err == nil {
			return dateEnd.After(now)
		}
		logger.Warn("session date_end unparseable; falling back to date_start",
			"session_key", raw.SessionKey,
			"date_end_raw", raw.DateEnd,
		)
	}

	if raw.DateStart != "" {
		if dateStart, err := time.Parse(time.RFC3339, raw.DateStart); err == nil {
			return !dateStart.Before(now)
		}
		logger.Warn("session date_start unparseable; treating as future",
			"session_key", raw.SessionKey,
			"date_start_raw", raw.DateStart,
		)
	}

	// Neither date usable — skip defensively.
	return true
}
