// Command backfill populates RaceControlSummary for pre-existing finalized sessions
// that are missing it. It is a one-shot CLI run after Feature 005 deployment.
//
// Usage:
//
//	COSMOS_ACCOUNT_ENDPOINT=https://... backfill --season=2026 [--dry-run] [--rate-limit-ms=1000]
package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/ingest"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/observability"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage/cosmos"
)

func main() {
	season := flag.Int("season", 0, "F1 season year to backfill (required)")
	dryRun := flag.Bool("dry-run", false, "Log what would change without writing to Cosmos")
	rateLimitMs := flag.Int("rate-limit-ms", 1000, "Delay between OpenF1 fetches in milliseconds")
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
