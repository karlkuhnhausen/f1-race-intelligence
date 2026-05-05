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

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/domain"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/ingest"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/observability"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/standings"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage/cosmos"
)

func main() {
	season := flag.Int("season", 0, "F1 season year to backfill (required)")
	dryRun := flag.Bool("dry-run", false, "Log what would change without writing to Cosmos")
	rateLimitMs := flag.Int("rate-limit-ms", 1000, "Delay between OpenF1 fetches in milliseconds")
	analysisMode := flag.Bool("analysis", false, "Backfill session analysis data (positions, intervals, stints, pits, overtakes) for Race/Sprint sessions")
	analysisOnly := flag.Bool("analysis-only", false, "Run only analysis backfill, skip race control backfill")
	championshipMode := flag.Bool("championship", false, "Backfill championship standings, session results, and starting grids for Race/Sprint sessions")
	stampMeetingKeys := flag.Bool("stamp-meeting-keys", false, "Stamp meeting_key (and session_key for analysis docs) on pre-migration documents")
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

	// --- Stamp meeting keys (Feature 008) ---
	if *stampMeetingKeys {
		stampMeetingKeysBackfill(ctx, client, *season, *dryRun, logger)
	}

	// --- Championship backfill (Feature 007) ---
	if *championshipMode {
		backfillChampionship(ctx, client, sessions, delay, *dryRun, *season, logger)
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
		if err := analysisRepo.UpsertSessionPositions(ctx, ingest.ToStoragePositions(sess.Season, sess.Round, sess.MeetingKey, sess.SessionKey, sess.SessionType, fetchResult.Positions)); err != nil {
			logger.Warn("backfill-analysis: upsert positions failed", "error", err.Error())
			aFailed++
			time.Sleep(delay)
			continue
		}
		if len(fetchResult.Intervals) > 0 {
			if err := analysisRepo.UpsertSessionIntervals(ctx, ingest.ToStorageIntervals(sess.Season, sess.Round, sess.MeetingKey, sess.SessionKey, sess.SessionType, fetchResult.Intervals)); err != nil {
				logger.Warn("backfill-analysis: upsert intervals failed", "error", err.Error())
			}
		}
		if len(fetchResult.Stints) > 0 {
			if err := analysisRepo.UpsertSessionStints(ctx, ingest.ToStorageStints(sess.Season, sess.Round, sess.MeetingKey, sess.SessionKey, sess.SessionType, fetchResult.Stints)); err != nil {
				logger.Warn("backfill-analysis: upsert stints failed", "error", err.Error())
			}
		}
		if len(fetchResult.Pits) > 0 {
			if err := analysisRepo.UpsertSessionPits(ctx, ingest.ToStoragePits(sess.Season, sess.Round, sess.MeetingKey, sess.SessionKey, sess.SessionType, fetchResult.Pits)); err != nil {
				logger.Warn("backfill-analysis: upsert pits failed", "error", err.Error())
			}
		}
		if len(fetchResult.Overtakes) > 0 {
			if err := analysisRepo.UpsertSessionOvertakes(ctx, ingest.ToStorageOvertakes(sess.Season, sess.Round, sess.MeetingKey, sess.SessionKey, sess.SessionType, fetchResult.Overtakes)); err != nil {
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
	MeetingKey  int
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

		if err := analysisRepo.UpsertSessionPositions(ctx, ingest.ToStoragePositions(sess.Season, sess.Round, sess.MeetingKey, sess.SessionKey, sess.SessionType, fetchResult.Positions)); err != nil {
			logger.Warn("backfill-analysis-explicit: upsert positions failed", "error", err.Error())
			aFailed++
			time.Sleep(delay)
			continue
		}
		if len(fetchResult.Intervals) > 0 {
			if err := analysisRepo.UpsertSessionIntervals(ctx, ingest.ToStorageIntervals(sess.Season, sess.Round, sess.MeetingKey, sess.SessionKey, sess.SessionType, fetchResult.Intervals)); err != nil {
				logger.Warn("backfill-analysis-explicit: upsert intervals failed", "error", err.Error())
			}
		}
		if len(fetchResult.Stints) > 0 {
			if err := analysisRepo.UpsertSessionStints(ctx, ingest.ToStorageStints(sess.Season, sess.Round, sess.MeetingKey, sess.SessionKey, sess.SessionType, fetchResult.Stints)); err != nil {
				logger.Warn("backfill-analysis-explicit: upsert stints failed", "error", err.Error())
			}
		}
		if len(fetchResult.Pits) > 0 {
			if err := analysisRepo.UpsertSessionPits(ctx, ingest.ToStoragePits(sess.Season, sess.Round, sess.MeetingKey, sess.SessionKey, sess.SessionType, fetchResult.Pits)); err != nil {
				logger.Warn("backfill-analysis-explicit: upsert pits failed", "error", err.Error())
			}
		}
		if len(fetchResult.Overtakes) > 0 {
			if err := analysisRepo.UpsertSessionOvertakes(ctx, ingest.ToStorageOvertakes(sess.Season, sess.Round, sess.MeetingKey, sess.SessionKey, sess.SessionType, fetchResult.Overtakes)); err != nil {
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

func backfillChampionship(
	ctx context.Context,
	client *cosmos.Client,
	sessions []storage.Session,
	delay time.Duration,
	dryRun bool,
	season int,
	logger *slog.Logger,
) {
	// Filter to Race and Sprint sessions only
	var champSessions []storage.Session
	for _, sess := range sessions {
		if sess.SessionType == "race" || sess.SessionType == "sprint" {
			champSessions = append(champSessions, sess)
		}
	}

	logger.Info("backfill-championship: starting",
		"season", season,
		"total_race_sprint_sessions", len(champSessions),
		"dry_run", dryRun,
	)

	ingester := standings.NewChampionshipIngester(client, logger)
	updated, failed := 0, 0

	for _, sess := range champSessions {
		logger.Info("backfill-championship: processing",
			"session_key", sess.SessionKey,
			"meeting_key", sess.MeetingKey,
			"round", sess.Round,
			"session_type", sess.SessionType,
		)

		if dryRun {
			logger.Info("backfill-championship: dry-run — would ingest",
				"session_key", sess.SessionKey,
				"outcome", "would-update",
			)
			updated++
			time.Sleep(delay)
			continue
		}

		if err := ingester.IngestSession(ctx, season, sess.SessionKey, sess.MeetingKey, sess.SessionType); err != nil {
			logger.Warn("backfill-championship: ingestion failed",
				"session_key", sess.SessionKey,
				"error", err.Error(),
				"outcome", "failed",
			)
			failed++
		} else {
			updated++
		}
		time.Sleep(delay)
	}

	logger.Info("backfill-championship: complete",
		"season", season,
		"updated", updated,
		"failed", failed,
		"dry_run", dryRun,
	)
}

// stampMeetingKeysBackfill populates meeting_key (and session_key for analysis
// docs) on pre-migration documents that have meeting_key=0.
func stampMeetingKeysBackfill(ctx context.Context, client *cosmos.Client, season int, dryRun bool, logger *slog.Logger) {
	// Step 1: Build MeetingIndex from calendar data.
	meetings, err := client.GetMeetingsBySeason(ctx, season)
	if err != nil {
		log.Fatalf("stamp-meeting-keys: fetch meetings: %v", err)
	}
	indexInputs := make([]domain.MeetingForIndex, 0, len(meetings))
	for _, m := range meetings {
		indexInputs = append(indexInputs, domain.MeetingForIndex{
			MeetingKey:       m.MeetingKey,
			RaceName:         m.RaceName,
			StartDatetimeUTC: m.StartDatetimeUTC,
			IsCancelled:      m.IsCancelled,
		})
	}
	meetingIdx := domain.BuildMeetingIndex(indexInputs)

	// Also build round→meeting_key lookup from actual RaceMeeting docs
	// (which have assigned round numbers).
	roundToMeetingKey := make(map[int]int, len(meetings))
	for _, m := range meetings {
		if m.MeetingKey > 0 {
			roundToMeetingKey[m.Round] = m.MeetingKey
		}
	}

	logger.Info("stamp-meeting-keys: starting",
		"season", season,
		"total_meetings", len(meetings),
		"index_rounds", meetingIdx.TotalRounds(),
		"dry_run", dryRun,
	)

	// Step 2: Stamp sessions.
	allSessions, err := client.GetFinalizedSessions(ctx, season)
	if err != nil {
		log.Fatalf("stamp-meeting-keys: fetch sessions: %v", err)
	}
	// Also get non-finalized sessions (upcoming/in-progress).
	allSessionsByRound := make(map[int][]storage.Session)
	for i := 1; i <= meetingIdx.TotalRounds(); i++ {
		roundSessions, qErr := client.GetSessionsByRound(ctx, season, i)
		if qErr != nil {
			logger.Warn("stamp-meeting-keys: query sessions by round failed", "round", i, "error", qErr)
			continue
		}
		allSessionsByRound[i] = roundSessions
		for _, s := range roundSessions {
			// Avoid duplicates with finalized list.
			found := false
			for _, fs := range allSessions {
				if fs.ID == s.ID {
					found = true
					break
				}
			}
			if !found {
				allSessions = append(allSessions, s)
			}
		}
	}

	sessStamped, sessSkipped, sessFailed := 0, 0, 0
	for _, sess := range allSessions {
		if sess.MeetingKey > 0 {
			sessSkipped++
			continue
		}
		mk, ok := roundToMeetingKey[sess.Round]
		if !ok {
			mk = meetingIdx.MeetingKeyForRound(sess.Round)
		}
		if mk == 0 {
			logger.Warn("stamp-meeting-keys: cannot resolve meeting_key for session",
				"session_id", sess.ID, "round", sess.Round)
			sessFailed++
			continue
		}
		sess.MeetingKey = mk
		if dryRun {
			logger.Info("stamp-meeting-keys: dry-run — would stamp session",
				"session_id", sess.ID, "round", sess.Round, "meeting_key", mk)
			sessStamped++
		} else {
			if err := client.UpsertSession(ctx, sess); err != nil {
				logger.Warn("stamp-meeting-keys: session upsert failed",
					"session_id", sess.ID, "error", err)
				sessFailed++
			} else {
				sessStamped++
			}
		}
	}
	logger.Info("stamp-meeting-keys: sessions complete",
		"stamped", sessStamped, "skipped", sessSkipped, "failed", sessFailed)

	// Step 3: Stamp session results.
	allResults, err := client.GetSessionResultsBySeason(ctx, season)
	if err != nil {
		log.Fatalf("stamp-meeting-keys: fetch results: %v", err)
	}
	resStamped, resSkipped, resFailed := 0, 0, 0
	for _, res := range allResults {
		if res.MeetingKey > 0 {
			resSkipped++
			continue
		}
		mk, ok := roundToMeetingKey[res.Round]
		if !ok {
			mk = meetingIdx.MeetingKeyForRound(res.Round)
		}
		if mk == 0 {
			logger.Warn("stamp-meeting-keys: cannot resolve meeting_key for result",
				"result_id", res.ID, "round", res.Round)
			resFailed++
			continue
		}
		res.MeetingKey = mk
		if dryRun {
			logger.Info("stamp-meeting-keys: dry-run — would stamp result",
				"result_id", res.ID, "round", res.Round, "meeting_key", mk)
			resStamped++
		} else {
			if err := client.UpsertSessionResult(ctx, res); err != nil {
				logger.Warn("stamp-meeting-keys: result upsert failed",
					"result_id", res.ID, "error", err)
				resFailed++
			} else {
				resStamped++
			}
		}
	}
	logger.Info("stamp-meeting-keys: results complete",
		"stamped", resStamped, "skipped", resSkipped, "failed", resFailed)

	// Step 4: Stamp analysis documents.
	// Build a session lookup for resolving session_key: season+round+type → session_key.
	sessionKeyLookup := make(map[string]int) // "round:type" → session_key
	for _, sess := range allSessions {
		key := fmt.Sprintf("%d:%s", sess.Round, sess.SessionType)
		if sess.SessionKey > 0 {
			sessionKeyLookup[key] = sess.SessionKey
		}
	}

	// Query all analysis docs for the season by querying each round+type combo.
	analysisStamped, analysisSkipped, analysisFailed := 0, 0, 0
	sessionTypes := []string{"race", "sprint"}

	for round := 1; round <= meetingIdx.TotalRounds(); round++ {
		mk, ok := roundToMeetingKey[round]
		if !ok {
			mk = meetingIdx.MeetingKeyForRound(round)
		}
		if mk == 0 {
			continue
		}

		for _, st := range sessionTypes {
			data, qErr := client.GetSessionAnalysis(ctx, season, round, st)
			if qErr != nil {
				logger.Warn("stamp-meeting-keys: query analysis failed",
					"round", round, "type", st, "error", qErr)
				analysisFailed++
				continue
			}
			if data == nil {
				continue
			}

			sk := sessionKeyLookup[fmt.Sprintf("%d:%s", round, st)]

			// Stamp positions.
			needsStamp := false
			for i := range data.Positions {
				if data.Positions[i].MeetingKey == 0 {
					needsStamp = true
					data.Positions[i].MeetingKey = mk
					data.Positions[i].SessionKey = sk
				}
			}
			if needsStamp {
				if dryRun {
					logger.Info("stamp-meeting-keys: dry-run — would stamp positions",
						"round", round, "type", st, "meeting_key", mk, "session_key", sk,
						"count", len(data.Positions))
				} else {
					if err := client.UpsertSessionPositions(ctx, data.Positions); err != nil {
						logger.Warn("stamp-meeting-keys: positions upsert failed",
							"round", round, "type", st, "error", err)
						analysisFailed++
					}
				}
				analysisStamped += len(data.Positions)
			} else {
				analysisSkipped += len(data.Positions)
			}

			// Stamp intervals.
			needsStamp = false
			for i := range data.Intervals {
				if data.Intervals[i].MeetingKey == 0 {
					needsStamp = true
					data.Intervals[i].MeetingKey = mk
					data.Intervals[i].SessionKey = sk
				}
			}
			if needsStamp {
				if dryRun {
					logger.Info("stamp-meeting-keys: dry-run — would stamp intervals",
						"round", round, "type", st, "meeting_key", mk, "session_key", sk,
						"count", len(data.Intervals))
				} else {
					if err := client.UpsertSessionIntervals(ctx, data.Intervals); err != nil {
						logger.Warn("stamp-meeting-keys: intervals upsert failed",
							"round", round, "type", st, "error", err)
						analysisFailed++
					}
				}
				analysisStamped += len(data.Intervals)
			} else {
				analysisSkipped += len(data.Intervals)
			}

			// Stamp stints.
			needsStamp = false
			for i := range data.Stints {
				if data.Stints[i].MeetingKey == 0 {
					needsStamp = true
					data.Stints[i].MeetingKey = mk
					data.Stints[i].SessionKey = sk
				}
			}
			if needsStamp {
				if dryRun {
					logger.Info("stamp-meeting-keys: dry-run — would stamp stints",
						"round", round, "type", st, "meeting_key", mk, "session_key", sk,
						"count", len(data.Stints))
				} else {
					if err := client.UpsertSessionStints(ctx, data.Stints); err != nil {
						logger.Warn("stamp-meeting-keys: stints upsert failed",
							"round", round, "type", st, "error", err)
						analysisFailed++
					}
				}
				analysisStamped += len(data.Stints)
			} else {
				analysisSkipped += len(data.Stints)
			}

			// Stamp pits.
			needsStamp = false
			for i := range data.Pits {
				if data.Pits[i].MeetingKey == 0 {
					needsStamp = true
					data.Pits[i].MeetingKey = mk
					data.Pits[i].SessionKey = sk
				}
			}
			if needsStamp {
				if dryRun {
					logger.Info("stamp-meeting-keys: dry-run — would stamp pits",
						"round", round, "type", st, "meeting_key", mk, "session_key", sk,
						"count", len(data.Pits))
				} else {
					if err := client.UpsertSessionPits(ctx, data.Pits); err != nil {
						logger.Warn("stamp-meeting-keys: pits upsert failed",
							"round", round, "type", st, "error", err)
						analysisFailed++
					}
				}
				analysisStamped += len(data.Pits)
			} else {
				analysisSkipped += len(data.Pits)
			}

			// Stamp overtakes.
			needsStamp = false
			for i := range data.Overtakes {
				if data.Overtakes[i].MeetingKey == 0 {
					needsStamp = true
					data.Overtakes[i].MeetingKey = mk
					data.Overtakes[i].SessionKey = sk
				}
			}
			if needsStamp {
				if dryRun {
					logger.Info("stamp-meeting-keys: dry-run — would stamp overtakes",
						"round", round, "type", st, "meeting_key", mk, "session_key", sk,
						"count", len(data.Overtakes))
				} else {
					if err := client.UpsertSessionOvertakes(ctx, data.Overtakes); err != nil {
						logger.Warn("stamp-meeting-keys: overtakes upsert failed",
							"round", round, "type", st, "error", err)
						analysisFailed++
					}
				}
				analysisStamped += len(data.Overtakes)
			} else {
				analysisSkipped += len(data.Overtakes)
			}
		}
	}

	logger.Info("stamp-meeting-keys: analysis complete",
		"stamped", analysisStamped, "skipped", analysisSkipped, "failed", analysisFailed)

	logger.Info("stamp-meeting-keys: all done",
		"season", season,
		"sessions_stamped", sessStamped,
		"results_stamped", resStamped,
		"analysis_stamped", analysisStamped,
		"dry_run", dryRun,
	)
}
