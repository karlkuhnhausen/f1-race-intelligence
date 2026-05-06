# Day 25: The Sprint Sessions That Showed Nothing — A Three-Bug Cascade

*Posted May 5, 2026 · Karl Kuhnhausen*

---

Miami was a sprint weekend. Five sessions: Practice 1, Sprint Qualifying, Sprint, Qualifying, Race. When you clicked into the Miami round detail page, you saw three sessions — Practice 1, Qualifying, and Race — each with 22 results. Sprint Qualifying and Sprint were invisible. Then they appeared with zero results. Then the results finally populated, but only after discovering a poller bug that had been silently wasting every OpenF1 API call for weeks.

Three PRs. Three distinct root causes. One frustrating afternoon of "we are thrashing back and forth."

## Bug 1: The Missing Sprint Sessions (PR #77)

The round detail page showed only three of Miami's five sessions. The sprint sessions existed in Cosmos but weren't appearing in the API response.

The root cause was session deduplication logic in the rounds service. When the Feature 008 stable-identity migration ran, it created new session documents with correct `session_type` values ("sprint", "sprint_qualifying") alongside old documents that had stale types. The dedup logic kept one session per `session_type` — but the old documents and new documents had different IDs, so both survived in Cosmos. When the API loaded sessions for a round, the dedup saw the old "race"-typed document and the new "sprint"-typed document as different types, kept both, and the frontend rendered whichever came first.

Except the old documents came first. And they had `session_type: "race"` — so the dedup kept the old race-typed document and discarded the new sprint-typed one.

**Fix:** Bump `SessionSchemaVersion` to 7, forcing the poller to re-fetch all sessions. The new documents overwrote the old ones with correct session types. The dedup in the rounds service was updated to keep the *newest* session per type (by `DateStartUTC`), and sessions were sorted chronologically for display.

## Bug 2: The Results That Went to the Wrong Session (PR #78)

After PR #77 deployed, Sprint Qualifying and Sprint appeared in the round detail page — but with zero results each. Meanwhile, the Race session had duplicate results (its own 22 plus another 22 from somewhere). The total result count was right; the distribution was wrong.

The root cause: result documents in Cosmos were grouped by `session_type` to match them to sessions. Old sprint result documents had `session_type: "race"` (inherited from pre-008 ingestion when sprints were mistyped). When the Sprint session (correctly typed `"sprint"`) looked up `resultsByType["sprint"]`, it found nothing. The Race session's lookup for `resultsByType["race"]` found its own 22 results *plus* the 22 mis-typed sprint results.

**Fix:** Group results by `session_key` instead of `session_type`. Each session has a globally unique `session_key` (assigned by OpenF1), and every result document stores the `session_key` it belongs to. This eliminated cross-contamination regardless of what `session_type` string was stored in stale documents.

```go
// Before: results grouped by session_type (fragile)
resultsByType := make(map[string][]storage.SessionResult)
for _, r := range results {
    resultsByType[r.SessionType] = append(resultsByType[r.SessionType], r)
}

// After: results grouped by session_key (unique per session)
resultsByKey := make(map[int][]storage.SessionResult)
for _, r := range results {
    resultsByKey[r.SessionKey] = append(resultsByKey[r.SessionKey], r)
}
```

## Bug 3: The Poller That Kept Undoing Its Own Work (PR #79)

After PR #78 deployed, the sprint sessions appeared with the correct structure — but still had zero results. The poller logs told the story:

```json
{"level":"ERROR","msg":"session results failed","session_key":11271,"error":"...unexpected status 429"}
{"level":"ERROR","msg":"session results failed","session_key":11275,"error":"...unexpected status 429"}
```

Every poll cycle, the poller re-processed sessions that should have been finalized and skipped. It finalized Round 1's sessions, then Round 2's, then hit OpenF1's rate limit before reaching Miami (Round 4). On the next cycle — same thing. Round 1 finalized again. Round 2 finalized again. Rate limited again. Miami never got a turn.

The root cause was a subtle ordering bug in the poll loop:

```go
// 1. Upsert session metadata (creates doc with Finalized=false)
sess := TransformSession(raw, season, round)
p.repo.UpsertSession(ctx, sess)  // ← Clobbers finalized flag!

// 2. Check if already finalized (too late — we just set it to false)
if cachedVer, isFinalized := finalizedKeys[raw.SessionKey]; isFinalized {
    skipped++
    continue
}
```

`TransformSession()` always creates a session document with `Finalized: false`. The metadata upsert ran *before* the finalized-skip check. So every previously-finalized session had its `Finalized` flag clobbered back to `false` in Cosmos on every single poll cycle. The `finalizedKeys` map was loaded at the top of the cycle from Cosmos — but by the time the skip check ran, the upsert had already overwritten the document.

The poller was doing the digital equivalent of checking whether a door is locked, then unlocking it, then checking again and finding it unlocked.

**Fix:** Move the finalized-skip check *before* the metadata upsert:

```go
// 1. Check if already finalized (skip entirely — no writes, no API calls)
if cachedVer, isFinalized := finalizedKeys[raw.SessionKey]; isFinalized {
    skipped++
    continue
}

// 2. Only then upsert metadata for sessions that need processing
sess := TransformSession(raw, season, round)
p.repo.UpsertSession(ctx, sess)
```

This matched the original design intent documented in the comment above `finalizedKeys`: "Sessions whose cached schema_version matches the current code version are skipped entirely — no metadata upsert, no results/drivers/laps fetch."

## The Rate-Limit Cascade

The finalized-clobber bug was particularly destructive because AKS runs two backend replicas. Both pods have independent pollers. Both were re-processing every session every cycle. With 36 completed sessions, each pod made ~108 OpenF1 API calls per cycle (session_result + drivers + laps per session) instead of the expected ~0 for already-finalized sessions. Two pods × 108 calls × every 5 minutes = sustained 429s from OpenF1.

After deploying the fix and temporarily scaling to 1 replica to let the backlog drain, the poller worked through all sessions in about 15 minutes. Miami's sprint sessions finally got their results:

![Miami round detail showing all five sessions with results](images/day-25-miami-full.png)

All five sessions populated:
- Practice 1: 22 results
- Sprint Qualifying: 21 results
- Sprint: 22 results
- Qualifying: 22 results
- Race: 22 results

(Sprint Qualifying has 21 instead of 22 — one driver likely had no data published by OpenF1 for that session.)

## The Sprint Results

![Miami Sprint session results](images/day-25-miami-sprint.png)

Compare with a standard (non-sprint) weekend like Suzuka, which has three sessions — Practice 1, Qualifying, Race — instead of five:

![Japan round detail — standard weekend](images/day-25-japan-full.png)

## What Each PR Fixed

| PR | Problem | Root Cause | Fix |
|----|---------|------------|-----|
| [#77](https://github.com/karlkuhnhausen/f1-race-intelligence/pull/77) | Sprint sessions invisible | Stale documents with wrong `session_type` survived dedup | Schema v7 bump + dedup keeps newest per type |
| [#78](https://github.com/karlkuhnhausen/f1-race-intelligence/pull/78) | Sprint sessions show 0 results | Results grouped by `session_type`, stale results had wrong type | Group by `session_key` instead |
| [#79](https://github.com/karlkuhnhausen/f1-race-intelligence/pull/79) | Poller never reaches Miami | Metadata upsert clobbers `Finalized` flag before skip check | Move skip check before upsert |

## Lesson

When a migration introduces new document versions alongside old ones, every query that touches those documents must tolerate both versions coexisting. Dedup by type, group by type, filter by type — all of these broke because the "type" field in old documents was wrong. Using a truly unique key (`session_key`) instead of a semantic label (`session_type`) made the system resilient to stale data without requiring a backfill.

The poller clobber bug is the more insidious lesson. The code *looked* correct — there was a finalized-skip check right there in the loop. But ordering matters. An upsert-then-check is not the same as check-then-upsert, and in a system where the upsert resets the very flag you're about to check, the difference is between "process 0 sessions per cycle" and "process all 36 sessions per cycle, forever."

---

[← Day 24: "USA Race" Is Not a Circuit — Fixing the Progression Chart Labels](day-24-circuit-name-labels.md) · [Day 26: 657 Unhealthy — Triaging Defender for Cloud on a Side Project →](day-26-defender-recommendations.md)
