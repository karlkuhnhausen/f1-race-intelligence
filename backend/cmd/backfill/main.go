// Command backfill populates RaceControlSummary for pre-existing finalized sessions
// that are missing it. It is a one-shot CLI run after Feature 005 deployment.
// With --analysis, it also backfills session analysis data (positions, intervals,
// stints, pits, overtakes) for Race and Sprint sessions (Feature 006).
//
// Usage:
//
//	COSMOS_ACCOUNT_ENDPOINT=https://... backfill --season=2026 [--dry-run] [--rate-limit-ms=1000] [--analysis]
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/ingest"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/observability"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage/cosmos"
)

func main() {
	season := flag.Int("season", 0, "F1 season year to backfill (required)")
	dryRun := flag.Bool("dry-run", false, "Log what would change without writing to Cosmos")
	rateLimitMs := flag.Int("rate-limit-ms", 1000, "Delay between OpenF1 fetches in milliseconds")
	analysisMode := flag.Bool("analysis", false, "Backfill session analysis data (positions, intervals, stints, pits, overtakes) for Race/Sprint sessions")
	analysisOnly := flag.Bool("analysis-only", false, "Run only analysis backfill, skip race control backfill")
	sessionsFlag := flag.String("sessions", "", "Explicit session mappings: round:session_key:type,... (e.g. 1:11234:race,2:11245:race)")
	deleteAnalysis := flag.String("delete-analysis", "", "Delete analysis data for round:type pairs (e.g. 4:race,5:race)")
	flag.Parse()

	if *season == 0 {
		log.Fatal("backfill: --season is required")
	}

	logger := observability.NewLogger(slog.LevelInfo)

	cosmosEndpoint := os.Getenv("COSMOS_ACCOUNT_ENDPOINT")
	if cosmosEndpoint == "" {
		log.Fatal("backfill: COSMOS_ACCOUNT_ENDPOINT env var is required")
	}

	client, err := cosmos.NewClient(cosmosEndpoint)
	if err != nil {
		log.Fatalf("backfill: cosmos client: %v", err)
	}

	ctx := context.Background()

	sessions, err := client.GetFinalizedSessions(ctx, *season)
	if err != nil {
		log.Fatalf("backfill: fetch finalized sessions: %v", err)
	}

	logger.Info("backfill: starting",
		"season", *season,
		"total_sessions", len(sessions),
		"dry_run", *dryRun,
		"rate_limit_ms", *rateLimitMs,
	)

	httpClient := &http.Client{Timeout: 30 * time.Second}
	delay := time.Duration(*rateLimitMs) * time.Millisecond

	if !*analysisOnly {
		updated, skipped, failed := 0, 0, 0

		for _, sess := range sessions {
			if sess.RaceControlSummary != nil {
				logger.Info("backfill: skipped — already has race control data",
					"session_id", sess.ID,
					"session_key", sess.SessionKey,
					"outcome", "skipped",
				)
				skipped++
				continue
			}

			logger.Info("backfill: fetching race control",
				"session_id", sess.ID,
				"session_key", sess.SessionKey,
			)

			msgs, fetchErr := ingest.FetchRaceControlMsgs(ctx, httpClient, sess.SessionKey)
			if fetchErr != nil {
				logger.Warn("backfill: fetch failed — skipping session",
					"session_id", sess.ID,
					"session_key", sess.SessionKey,
					"error", fetchErr.Error(),
					"outcome", "failed",
				)
				failed++
				time.Sleep(delay)
				continue
			}

			summary := ingest.SummarizeRaceControl(msgs)
			sess.RaceControlSummary = &summary

			if *dryRun {
				logger.Info("backfill: dry-run — would update",
					"session_id", sess.ID,
					"session_key", sess.SessionKey,
					"red_flags", summary.RedFlagCount,
					"safety_cars", summary.SafetyCarCount,
					"vscs", summary.VSCCount,
					"outcome", "would-update",
				)
			} else {
				if upsertErr := client.UpsertSession(ctx, sess); upsertErr != nil {
					logger.Warn("backfill: upsert failed — skipping session",
						"session_id", sess.ID,
						"session_key", sess.SessionKey,
						"error", upsertErr.Error(),
						"outcome", "failed",
					)
					failed++
					time.Sleep(delay)
					continue
				}
				logger.Info("backfill: updated",
					"session_id", sess.ID,
					"session_key", sess.SessionKey,
					"red_flags", summary.RedFlagCount,
					"safety_cars", summary.SafetyCarCount,
					"vscs", summary.VSCCount,
					"outcome", "updated",
				)
			}
			updated++
			time.Sleep(delay)
		}

		logger.Info("backfill: complete",
			"season", *season,
			"updated", updated,
			"skipped", skipped,
			"failed", failed,
			"dry_run", *dryRun,
		)
	}

	// --- Delete analysis data ---
	if *deleteAnalysis != "" {
		for _, pair := range strings.Split(*deleteAnalysis, ",") {
			parts := strings.Split(strings.TrimSpace(pair), ":")
			if len(parts) != 2 {
				log.Fatalf("backfill: invalid --delete-analysis entry %q (expected round:type)", pair)
			}
			round, err := strconv.Atoi(parts[0])
			if err != nil {
				log.Fatalf("backfill: invalid round in delete-analysis %q: %v", pair, err)
			}
			sessionType := parts[1]
			deleted, err := client.DeleteAnalysisData(ctx, *season, round, sessionType)
			if err != nil {
				logger.Warn("backfill: delete-analysis failed", "round", round, "type", sessionType, "error", err.Error())
			} else {
				logger.Info("backfill: deleted analysis data", "round", round, "type", sessionType, "deleted_docs", deleted)
			}
		}
	}

	// --- Analysis backfill (Feature 006) ---
	if *analysisMode || *analysisOnly || *sessionsFlag != "" {
		if *sessionsFlag != "" {
			// Explicit session mappings: round:session_key:type,...
			explicitSessions := parseSessionsFlag(*sessionsFlag, *season)
			backfillAnalysisExplicit(ctx, client, httpClient, delay, *dryRun, explicitSessions, logger)
		} else {
			backfillAnalysis(ctx, client, client, sessions, httpClient, delay, *dryRun, logger)
		}
	}
}

func backfillAnalysis(
	ctx context.Context,
	sessionRepo *cosmos.Client,
	analysisRepo *cosmos.Client,
	sessions []storage.Session,
	httpClient *http.Client,
	delay time.Duration,
	dryRun bool,
	logger *slog.Logger,
) {
	// Filter to Race and Sprint sessions only
	var analysisSessions []storage.Session
	for _, sess := range sessions {
		st := sess.SessionType
		if st == "race" || st == "sprint" {
			analysisSessions = append(analysisSessions, sess)
		}
	}

	logger.Info("backfill-analysis: starting",
		"total_race_sprint_sessions", len(analysisSessions),
		"dry_run", dryRun,
	)

	aUpdated, aSkipped, aFailed := 0, 0, 0

	for _, sess := range analysisSessions {
		// Check if analysis data already exists (idempotent)
		hasData, err := analysisRepo.HasAnalysisData(ctx, sess.Season, sess.Round, sess.SessionType)
		if err != nil {
			logger.Warn("backfill-analysis: HasAnalysisData check failed",
				"session_key", sess.SessionKey,
				"error", err.Error(),
			)
			aFailed++
			time.Sleep(delay)
			continue
		}
		if hasData {
			logger.Info("backfill-analysis: skipped — already has analysis data",
				"session_key", sess.SessionKey,
				"round", sess.Round,
				"session_type", sess.SessionType,
				"outcome", "skipped",
			)
			aSkipped++
			continue
		}

		// Build driver info map from existing session results
		results, err := sessionRepo.GetSessionResultsByRound(ctx, sess.Season, sess.Round)
		if err != nil {
			logger.Warn("backfill-analysis: failed to get session results for drivers",
				"session_key", sess.SessionKey,
				"error", err.Error(),
			)
			aFailed++
			time.Sleep(delay)
			continue
		}

		drivers := make(map[int]ingest.DriverInfo)
		for _, r := range results {
			if r.SessionType == sess.SessionType {
				drivers[r.DriverNumber] = ingest.DriverInfo{
					DriverNumber:  r.DriverNumber,
					DriverName:    r.DriverName,
					DriverAcronym: r.DriverAcronym,
					TeamName:      r.TeamName,
				}
			}
		}

		if len(drivers) == 0 {
			logger.Warn("backfill-analysis: no driver data available, skipping",
				"session_key", sess.SessionKey,
				"outcome", "failed",
			)
			aFailed++
			time.Sleep(delay)
			continue
		}

		logger.Info("backfill-analysis: fetching analysis data",
			"session_key", sess.SessionKey,
			"round", sess.Round,
			"session_type", sess.SessionType,
			"drivers", len(drivers),
		)

		if dryRun {
			logger.Info("backfill-analysis: dry-run — would fetch and store",
				"session_key", sess.SessionKey,
				"outcome", "would-update",
			)
			aUpdated++
			time.Sleep(delay)
			continue
		}

		fetchResult, err := ingest.FetchAllAnalysisData(ctx, httpClient, sess.SessionKey, drivers, logger)
		if err != nil {
			logger.Warn("backfill-analysis: fetch failed",
				"session_key", sess.SessionKey,
				"error", err.Error(),
				"outcome", "failed",
			)
			aFailed++
			time.Sleep(delay)
			continue
		}

		// Persist each data type
		if err := analysisRepo.UpsertSessionPositions(ctx, ingest.ToStoragePositions(sess.Season, sess.Round, sess.SessionType, fetchResult.Positions)); err != nil {
			logger.Warn("backfill-analysis: upsert positions failed", "error", err.Error())
			aFailed++
			time.Sleep(delay)
			continue
		}
		if len(fetchResult.Intervals) > 0 {
			if err := analysisRepo.UpsertSessionIntervals(ctx, ingest.ToStorageIntervals(sess.Season, sess.Round, sess.SessionType, fetchResult.Intervals)); err != nil {
				logger.Warn("backfill-analysis: upsert intervals failed", "error", err.Error())
			}
		}
		if len(fetchResult.Stints) > 0 {
			if err := analysisRepo.UpsertSessionStints(ctx, ingest.ToStorageStints(sess.Season, sess.Round, sess.SessionType, fetchResult.Stints)); err != nil {
				logger.Warn("backfill-analysis: upsert stints failed", "error", err.Error())
			}
		}
		if len(fetchResult.Pits) > 0 {
			if err := analysisRepo.UpsertSessionPits(ctx, ingest.ToStoragePits(sess.Season, sess.Round, sess.SessionType, fetchResult.Pits)); err != nil {
				logger.Warn("backfill-analysis: upsert pits failed", "error", err.Error())
			}
		}
		if len(fetchResult.Overtakes) > 0 {
			if err := analysisRepo.UpsertSessionOvertakes(ctx, ingest.ToStorageOvertakes(sess.Season, sess.Round, sess.SessionType, fetchResult.Overtakes)); err != nil {
				logger.Warn("backfill-analysis: upsert overtakes failed", "error", err.Error())
			}
		}

		logger.Info("backfill-analysis: updated",
			"session_key", sess.SessionKey,
			"round", sess.Round,
			"session_type", sess.SessionType,
			"positions", len(fetchResult.Positions),
			"intervals", len(fetchResult.Intervals),
			"stints", len(fetchResult.Stints),
			"pits", len(fetchResult.Pits),
			"overtakes", len(fetchResult.Overtakes),
			"outcome", "updated",
		)
		aUpdated++
		time.Sleep(delay)
	}

	logger.Info("backfill-analysis: complete",
		"updated", aUpdated,
		"skipped", aSkipped,
		"failed", aFailed,
	)
}

type explicitSession struct {
	Round       int
	SessionKey  int
	SessionType string
	Season      int
}

func parseSessionsFlag(flag string, season int) []explicitSession {
	var result []explicitSession
	for _, part := range strings.Split(flag, ",") {
		parts := strings.Split(strings.TrimSpace(part), ":")
		if len(parts) != 3 {
			log.Fatalf("backfill: invalid --sessions entry %q (expected round:session_key:type)", part)
		}
		round, err := strconv.Atoi(parts[0])
		if err != nil {
			log.Fatalf("backfill: invalid round in %q: %v", part, err)
		}
		key, err := strconv.Atoi(parts[1])
		if err != nil {
			log.Fatalf("backfill: invalid session_key in %q: %v", part, err)
		}
		result = append(result, explicitSession{
			Round:       round,
			SessionKey:  key,
			SessionType: parts[2],
			Season:      season,
		})
	}
	return result
}

func backfillAnalysisExplicit(
	ctx context.Context,
	analysisRepo *cosmos.Client,
	httpClient *http.Client,
	delay time.Duration,
	dryRun bool,
	sessions []explicitSession,
	logger *slog.Logger,
) {
	logger.Info("backfill-analysis-explicit: starting",
		"sessions", len(sessions),
		"dry_run", dryRun,
	)

	aUpdated, aSkipped, aFailed := 0, 0, 0

	for _, sess := range sessions {
		hasData, err := analysisRepo.HasAnalysisData(ctx, sess.Season, sess.Round, sess.SessionType)
		if err != nil {
			logger.Warn("backfill-analysis-explicit: HasAnalysisData check failed",
				"round", sess.Round,
				"session_key", sess.SessionKey,
				"error", err.Error(),
			)
			aFailed++
			time.Sleep(delay)
			continue
		}
		if hasData {
			logger.Info("backfill-analysis-explicit: skipped — already has data",
				"round", sess.Round,
				"session_key", sess.SessionKey,
				"outcome", "skipped",
			)
			aSkipped++
			continue
		}

		// Build driver map from session results in Cosmos
		results, err := analysisRepo.GetSessionResultsByRound(ctx, sess.Season, sess.Round)
		if err != nil {
			logger.Warn("backfill-analysis-explicit: failed to get session results",
				"round", sess.Round,
				"error", err.Error(),
			)
			aFailed++
			time.Sleep(delay)
			continue
		}

		drivers := make(map[int]ingest.DriverInfo)
		for _, r := range results {
			if r.SessionType == sess.SessionType {
				drivers[r.DriverNumber] = ingest.DriverInfo{
					DriverNumber:  r.DriverNumber,
					DriverName:    r.DriverName,
					DriverAcronym: r.DriverAcronym,
					TeamName:      r.TeamName,
				}
			}
		}

		if len(drivers) == 0 {
			logger.Warn("backfill-analysis-explicit: no drivers found, skipping",
				"round", sess.Round,
				"session_key", sess.SessionKey,
			)
			aFailed++
			time.Sleep(delay)
			continue
		}

		logger.Info("backfill-analysis-explicit: fetching",
			"round", sess.Round,
			"session_key", sess.SessionKey,
			"session_type", sess.SessionType,
			"drivers", len(drivers),
		)

		if dryRun {
			logger.Info("backfill-analysis-explicit: dry-run — would fetch",
				"round", sess.Round,
				"outcome", "would-update",
			)
			aUpdated++
			time.Sleep(delay)
			continue
		}

		fetchResult, err := ingest.FetchAllAnalysisData(ctx, httpClient, sess.SessionKey, drivers, logger)
		if err != nil {
			logger.Warn("backfill-analysis-explicit: fetch failed",
				"round", sess.Round,
				"session_key", sess.SessionKey,
				"error", err.Error(),
			)
			aFailed++
			time.Sleep(delay)
			continue
		}

		if err := analysisRepo.UpsertSessionPositions(ctx, ingest.ToStoragePositions(sess.Season, sess.Round, sess.SessionType, fetchResult.Positions)); err != nil {
			logger.Warn("backfill-analysis-explicit: upsert positions failed", "error", err.Error())
			aFailed++
			time.Sleep(delay)
			continue
		}
		if len(fetchResult.Intervals) > 0 {
			if err := analysisRepo.UpsertSessionIntervals(ctx, ingest.ToStorageIntervals(sess.Season, sess.Round, sess.SessionType, fetchResult.Intervals)); err != nil {
				logger.Warn("backfill-analysis-explicit: upsert intervals failed", "error", err.Error())
			}
		}
		if len(fetchResult.Stints) > 0 {
			if err := analysisRepo.UpsertSessionStints(ctx, ingest.ToStorageStints(sess.Season, sess.Round, sess.SessionType, fetchResult.Stints)); err != nil {
				logger.Warn("backfill-analysis-explicit: upsert stints failed", "error", err.Error())
			}
		}
		if len(fetchResult.Pits) > 0 {
			if err := analysisRepo.UpsertSessionPits(ctx, ingest.ToStoragePits(sess.Season, sess.Round, sess.SessionType, fetchResult.Pits)); err != nil {
				logger.Warn("backfill-analysis-explicit: upsert pits failed", "error", err.Error())
			}
		}
		if len(fetchResult.Overtakes) > 0 {
			if err := analysisRepo.UpsertSessionOvertakes(ctx, ingest.ToStorageOvertakes(sess.Season, sess.Round, sess.SessionType, fetchResult.Overtakes)); err != nil {
				logger.Warn("backfill-analysis-explicit: upsert overtakes failed", "error", err.Error())
			}
		}

		logger.Info("backfill-analysis-explicit: updated",
			"round", sess.Round,
			"session_key", sess.SessionKey,
			"positions", len(fetchResult.Positions),
			"intervals", len(fetchResult.Intervals),
			"stints", len(fetchResult.Stints),
			"pits", len(fetchResult.Pits),
			"overtakes", len(fetchResult.Overtakes),
			"outcome", "updated",
		)
		aUpdated++
		time.Sleep(delay)
	}

	logger.Info("backfill-analysis-explicit: complete",
		"updated", aUpdated,
		"skipped", aSkipped,
		"failed", aFailed,
	)

	// Clean up incorrectly-tagged analysis data if any
	if aUpdated > 0 {
		fmt.Println("Backfill explicit complete.")
	}
}
