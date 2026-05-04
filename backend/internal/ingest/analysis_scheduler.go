package ingest

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// analysisDelay is how long after a session's DateEndUTC we wait before
// attempting to fetch analysis data. OpenF1 typically publishes data
// 60-90 minutes after session end.
const analysisDelay = 90 * time.Minute

// analysisRetryInterval is how long to wait between retry attempts when
// a fetch fails (data may not be available yet).
const analysisRetryInterval = 30 * time.Minute

// analysisMaxRetries is the maximum number of retry attempts per session
// before giving up until the next poll cycle.
const analysisMaxRetries = 4

// analysisPollInterval is how often the scheduler checks for sessions
// needing analysis data.
const analysisPollInterval = 15 * time.Minute

// AnalysisScheduler watches for completed Race/Sprint sessions and
// automatically fetches analysis data from OpenF1 once it becomes
// available (~90 minutes post-session).
type AnalysisScheduler struct {
	sessionRepo  storage.SessionRepository
	analysisRepo storage.AnalysisRepository
	client       *http.Client
	logger       *slog.Logger

	// retries tracks per-session retry counts (keyed by "season:round:type").
	retries map[string]int
}

// NewAnalysisScheduler creates a new scheduler.
func NewAnalysisScheduler(
	sessionRepo storage.SessionRepository,
	analysisRepo storage.AnalysisRepository,
	logger *slog.Logger,
) *AnalysisScheduler {
	return &AnalysisScheduler{
		sessionRepo:  sessionRepo,
		analysisRepo: analysisRepo,
		client:       &http.Client{Timeout: 30 * time.Second},
		logger:       logger,
		retries:      make(map[string]int),
	}
}

// Start begins the scheduler loop. It blocks until the context is cancelled.
func (s *AnalysisScheduler) Start(ctx context.Context, season int) {
	s.logger.Info("analysis scheduler starting", "season", season, "poll_interval", analysisPollInterval)

	// Run immediately on startup to catch any missed sessions.
	s.poll(ctx, season)

	ticker := time.NewTicker(analysisPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("analysis scheduler stopped")
			return
		case <-ticker.C:
			s.poll(ctx, season)
		}
	}
}

func (s *AnalysisScheduler) poll(ctx context.Context, season int) {
	sessions, err := s.sessionRepo.GetFinalizedSessions(ctx, season)
	if err != nil {
		s.logger.Error("analysis scheduler: fetch finalized sessions failed", "error", err)
		return
	}

	now := time.Now().UTC()

	for _, sess := range sessions {
		// Only Race and Sprint sessions have meaningful analysis data.
		if sess.SessionType != "race" && sess.SessionType != "sprint" {
			continue
		}

		// Wait for the analysis delay after session end.
		readyAt := sess.DateEndUTC.Add(analysisDelay)
		if now.Before(readyAt) {
			continue
		}

		retryKey := retryKeyFor(sess)

		// Skip if max retries exhausted (will reset on next server restart).
		if s.retries[retryKey] >= analysisMaxRetries {
			continue
		}

		// Check if we already have analysis data (idempotent).
		hasData, err := s.analysisRepo.HasAnalysisData(ctx, sess.Season, sess.Round, sess.SessionType)
		if err != nil {
			s.logger.Warn("analysis scheduler: HasAnalysisData check failed",
				"round", sess.Round, "session_type", sess.SessionType, "error", err)
			continue
		}
		if hasData {
			continue
		}

		// Check retry timing: if we've already retried, enforce the retry interval.
		if s.retries[retryKey] > 0 {
			lastAttemptAt := readyAt.Add(time.Duration(s.retries[retryKey]) * analysisRetryInterval)
			if now.Before(lastAttemptAt) {
				continue
			}
		}

		s.logger.Info("analysis scheduler: fetching analysis data",
			"round", sess.Round,
			"session_type", sess.SessionType,
			"session_key", sess.SessionKey,
			"attempt", s.retries[retryKey]+1,
		)

		if err := s.fetchAndStore(ctx, sess); err != nil {
			s.retries[retryKey]++
			s.logger.Warn("analysis scheduler: fetch failed, will retry",
				"round", sess.Round,
				"session_type", sess.SessionType,
				"attempt", s.retries[retryKey],
				"max_retries", analysisMaxRetries,
				"error", err,
			)
		} else {
			s.logger.Info("analysis scheduler: analysis data stored successfully",
				"round", sess.Round,
				"session_type", sess.SessionType,
			)
			// Clear retry counter on success.
			delete(s.retries, retryKey)
		}
	}
}

func (s *AnalysisScheduler) fetchAndStore(ctx context.Context, sess storage.Session) error {
	// Build driver info map from session results.
	results, err := s.sessionRepo.GetSessionResultsByRound(ctx, sess.Season, sess.Round)
	if err != nil {
		return err
	}

	drivers := make(map[int]DriverInfo)
	for _, r := range results {
		if r.SessionType == sess.SessionType {
			drivers[r.DriverNumber] = DriverInfo{
				DriverNumber:  r.DriverNumber,
				DriverName:    r.DriverName,
				DriverAcronym: r.DriverAcronym,
				TeamName:      r.TeamName,
			}
		}
	}

	if len(drivers) == 0 {
		return fmt.Errorf("no driver data available for round %d %s", sess.Round, sess.SessionType)
	}

	fetchResult, err := FetchAllAnalysisData(ctx, s.client, sess.SessionKey, drivers, s.logger)
	if err != nil {
		return err
	}

	// Persist each data type.
	if err := s.analysisRepo.UpsertSessionPositions(ctx, ToStoragePositions(sess.Season, sess.Round, sess.SessionType, fetchResult.Positions)); err != nil {
		return fmt.Errorf("upsert positions: %w", err)
	}
	if len(fetchResult.Intervals) > 0 {
		if err := s.analysisRepo.UpsertSessionIntervals(ctx, ToStorageIntervals(sess.Season, sess.Round, sess.SessionType, fetchResult.Intervals)); err != nil {
			s.logger.Warn("analysis scheduler: upsert intervals failed", "error", err)
		}
	}
	if len(fetchResult.Stints) > 0 {
		if err := s.analysisRepo.UpsertSessionStints(ctx, ToStorageStints(sess.Season, sess.Round, sess.SessionType, fetchResult.Stints)); err != nil {
			s.logger.Warn("analysis scheduler: upsert stints failed", "error", err)
		}
	}
	if len(fetchResult.Pits) > 0 {
		if err := s.analysisRepo.UpsertSessionPits(ctx, ToStoragePits(sess.Season, sess.Round, sess.SessionType, fetchResult.Pits)); err != nil {
			s.logger.Warn("analysis scheduler: upsert pits failed", "error", err)
		}
	}
	if len(fetchResult.Overtakes) > 0 {
		if err := s.analysisRepo.UpsertSessionOvertakes(ctx, ToStorageOvertakes(sess.Season, sess.Round, sess.SessionType, fetchResult.Overtakes)); err != nil {
			s.logger.Warn("analysis scheduler: upsert overtakes failed", "error", err)
		}
	}

	return nil
}

func retryKeyFor(sess storage.Session) string {
	return fmt.Sprintf("%d:%d:%s", sess.Season, sess.Round, sess.SessionType)
}
