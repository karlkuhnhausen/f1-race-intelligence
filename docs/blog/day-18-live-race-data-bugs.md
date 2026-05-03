# Day 18: The Race Weekend That Showed Nothing

*Posted May 2, 2026 · Karl Kuhnhausen*

---

Miami is live. Practice 1 happened yesterday. Sprint Qualifying happened. The Sprint ran this morning. Qualifying just finished. Tomorrow's the race.

And the dashboard was showing "completed" badges on every session — with zero results. The race, which hasn't happened yet, was marked completed. No data for any session. The whole round detail page was a grid of green "completed" chips and empty tables.

Time to dig.

---

## Bug 1: Invisible Sessions

The session poller has a gate called `isFutureSession`. It's supposed to prevent us from hitting the OpenF1 `/session_result` endpoint for sessions that haven't happened yet (those return 404 and waste our rate-limit budget).

The problem: the gate was too aggressive. It skipped the **entire session** — metadata *and* results. Sessions whose `date_end` was in the future never got written to Cosmos DB at all.

So the Race (tomorrow), and for a brief window the Qualifying session (today), simply didn't exist in our cache. The frontend couldn't show what wasn't there.

The fix:

```go
// Before: future sessions were completely invisible
if isFutureSession(raw, now, p.logger) {
    skippedFuture++
    continue  // skips EVERYTHING including metadata upsert
}

// After: always write metadata, only skip results fetch
sess := TransformSession(raw, season, round)
p.repo.UpsertSession(ctx, sess)

if isFutureSession(raw, now, p.logger) {
    skippedFuture++
    continue  // only skips the results/drivers/laps fetch
}
```

Now upcoming sessions appear in the round detail with status "upcoming" or "in_progress" derived at read time. The frontend can show them with countdown timers. They just won't have results until the session ends and the next poll cycle picks them up.

---

## Bug 2: Finalized With No Results

This one was sneakier. Some sessions that *had* ended — FP1, Sprint Qualifying — were showing as completed but with empty result tables. The data exists in OpenF1 (I verified: 22 results each). So why wasn't it in our cache?

The session poller fetches results in three stages:
1. `/session_result` — positions and times
2. `/drivers` — names, acronyms, teams
3. `/laps` — for fastest-lap derivation (race/sprint only)

There's a safety guard: if driver enrichment fails (rate-limit, timeout), we skip individual result upserts to avoid overwriting good data with empty `driver_name` fields. Sensible.

But the guard was too broad. When `/drivers` returned an error, `driverMap` was empty, so *every* result hit the skip condition. And `fetchAndUpsertResults` returned `nil` — success. The poll loop then checked "has this session ended long enough ago?" Yes. "Mark it finalized." Done.

The session was now **permanently locked** with zero results. Future polls would skip it via the finalized cache, and the data would never be retried.

```go
// The fix: detect when ALL results were skipped due to missing drivers
if len(rawResults) > 0 && len(driverMap) == 0 {
    return errNoDriverData  // prevents finalization, retries next cycle
}
```

I bumped `SessionSchemaVersion` to 3 so that sessions incorrectly finalized under version 2 would be treated as stale and re-fetched.

---

## Bug 3: Ghost Sessions From a Past Life

After fixing bugs 1 and 2, I noticed Miami was showing Practice 2 and Practice 3. But Miami is a sprint weekend — there is no FP2 or FP3. The schedule is FP1 → Sprint Qualifying → Sprint → Qualifying → Race.

These weren't coming from OpenF1 (confirmed: only 5 sessions for meeting 1284). They were **orphaned Cosmos DB documents** from schema version 1, when a round-numbering bug had assigned a different meeting's sessions to round 6. When the numbering was corrected in v2, the correct Miami sessions got written with new IDs — but the old `2026-06-fp2` and `2026-06-fp3` documents were never deleted.

The system only upserts. It never reconciles.

The fix: a new reconciliation step that runs at the end of each poll cycle:

```go
func (p *SessionPoller) reconcileStaleSessions(ctx, upstreamSessions, meetingRounds, season) {
    // Build set of valid (round, session_type) from upstream
    // For each round, query cached sessions
    // Delete any that don't match upstream
}
```

Added `DeleteSession` and `DeleteSessionResultsBySessionType` to the repository interface. On next deployment, the orphaned documents get cleaned up automatically.

---

## The Interconnection

These three bugs compounded each other to create the worst possible UX:
- Bug 1 hid the upcoming Race session → dashboard looked "done"
- Bug 2 locked completed sessions with empty results → nothing to show
- Bug 3 showed phantom FP2/FP3 → confusing sprint-weekend layout

Each bug individually would have been noticeable but not catastrophic. Together, they made the live race weekend page completely useless at the one moment you'd actually want to check it.

---

## What I Shipped

- **PR #43**: Fix session metadata visibility + prevent finalization without results + schema v3 bump
- **PR #44**: Stale session reconciliation with delete methods

Both merged. The cluster will pick up the changes on next deploy. By tomorrow's race, the page should show:
- FP1 results (completed, 22 drivers)
- Sprint Qualifying results (completed, 21 drivers)
- Sprint results (completed, 22 drivers)
- Qualifying results (completed, 22 drivers)
- Race (upcoming, countdown to 8 PM UTC Sunday)

---

## Lesson

Cache-only architectures need reconciliation, not just upsert. If you only ever write forward, you accumulate ghosts from every schema migration and round-renumbering event. And finalization gates need to verify they're locking in *good* data — not empty data that happened to pass through without errors.

---

[← Day 17: A Real CVE Lands on a Real Cluster](day-17-cve-aks-5753-patching.md) · [Day 19: The Session Recap Strip →](day-19-session-recap-strip.md)
