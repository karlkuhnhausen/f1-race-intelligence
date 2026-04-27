# Day 10: The Rate Limit Cascade — Three PRs to Make a Race Page Render

*Posted April 26, 2026 · Karl Kuhnhausen*

---

The plan today was small: ship Feature 003 Phases 5–7 (qualifying and practice tables) and call it a feature. The work had been waiting in a stash for two days while [Day 8](day-8-ops-security-lockdown.md) and [Day 9](day-9-cosmos-private-endpoint.md) consumed the weekend on security. Just merge the branch, watch the deploy, screenshot it for the blog.

I shipped three PRs instead. Each one fixed a bug exposed by the previous one. By the end of it the site was actually working — but the road there was a tour through every layer of the stack: data shape, transform logic, and an emergent rate-limit cascade that was silently dropping 95% of the season's data.

This is the story of how "the feature is done" turned into "the feature is done now, for real, I think."

---

## PR #14: Ship the components

The first PR was the easy one. Phases 5–7 of Feature 003:

- A `QualifyingResults` component with Q1/Q2/Q3 columns
- A `PracticeResults` component with best-lap and gap-to-fastest
- Polish: shared formatters, em-dash placeholders for missing values, a session-type tab strip on the round detail page

Tests went green: 38 frontend, 22 backend. CI passed. PR merged. I clicked into Round 3 on the live site to admire my work.

Every cell was empty.

Driver names, team names, positions — all populated. But every column that came from the new code was an em-dash. Time/Gap: `—`. Q1/Q2/Q3: `—`. Best Lap: `—`. The components rendered fine. The data wasn't there.

---

## PR #15: The endpoint I was hitting wasn't the right endpoint

I dropped into the backend pod logs. No errors. The session poller ran every 5 minutes and reported "session poll complete" cheerfully. The Cosmos `session_results` documents existed. But when I queried one directly, fields like `points`, `race_time`, `gap_to_leader`, and `q1_time` were just... missing.

It took me longer than I want to admit to figure out why. The OpenF1 API has two endpoints that look like they do the same thing:

- `/v1/position` — real-time position data, one row per driver per timestamp during a session. Useful for live tracking.
- `/v1/session_result` — the *final classification*, one row per driver per session, with points, gaps, lap counts, and per-segment qualifying times.

My poller had been calling `/v1/position`. It was happily ingesting the *last position update* for each driver, which gave it a position number and a driver number — but none of the rich result fields existed in that response shape. Those fields only exist in `/v1/session_result`.

The fix was structural:

1. Replace the `/v1/position` call with `/v1/session_result`.
2. Rewrite the transform logic to handle the polymorphic shape: for races and practice, `duration` and `gap_to_leader` are scalars; for qualifying, they're three-element arrays (`[Q1, Q2, Q3]`). I modeled them as `json.RawMessage` and dispatched on session type.
3. Derive finishing status (`DNF` > `DNS` > `DSQ` > `Finished`) from the boolean flags OpenF1 returns.
4. For races, also fetch `/v1/laps` and compute the fastest-lap holder ourselves — OpenF1 doesn't flag it.

PR #15 went out, CI passed, deploy ran, pods rolled. I refreshed the page expecting victory.

Round 3 P1 had data. Round 3 P2 had data.

Round 3 P3, qualifying, race — empty. Round 4, Round 5 — empty. Last-updated timestamp was advancing, but the data wasn't.

---

## PR #16: The cascade

Back to the logs. This time I scrolled past "session poll complete" and looked at what happened *during* the cycle:

```
ERROR session results failed session_key=11383 error="...unexpected status 429"
ERROR session results failed session_key=11384 error="...unexpected status 429"
ERROR session results failed session_key=11388 error="...unexpected status 429"
ERROR session results failed session_key=11392 error="...unexpected status 404"
ERROR session results failed session_key=11399 error="...unexpected status 404"
ERROR session results failed session_key=11428 error="...unexpected status 429"
```

Hundreds of them. The poller was iterating *all 126 sessions of the 2026 season* every 5 minutes — including:

- ~95 future sessions (Miami, Imola, Monaco, ... all the way to Abu Dhabi). OpenF1 returns 404 for these because they haven't happened yet.
- ~10 cancelled sessions from cancelled meetings (the Bahrain round-6 GP and Saudi round-7 GP got cancelled this season). OpenF1 returns 404 for those too — permanently.
- ~21 sessions that *did* have data.

The 500ms inter-session delay I'd added wasn't enough. After about 10–20 requests OpenF1 starts returning 429 (rate limit), and once the rate limit kicks in it cascades — the 429s keep coming as the loop continues to hammer the endpoint. The first few sessions in the iteration got through cleanly. Round 3 P1 and P2 happened to be the first non-test sessions hit, which is why they had data and nothing else did.

I'd built a tight loop that hit the API ~120 times every 5 minutes when ~21 of those calls were the only useful ones.

PR #16 fixed it with three layered changes:

### 1. Pre-filter the loop

Before any HTTP call, skip sessions that obviously don't need fetching:

```go
if raw.IsCancelled {
    skippedCancelled++
    continue
}
if dateEnd, err := time.Parse(time.RFC3339, raw.DateEnd); err == nil && dateEnd.After(now) {
    skippedFuture++
    continue
}
```

The OpenF1 `/v1/sessions` response includes an `is_cancelled` flag — I just hadn't been parsing it. And future sessions are trivially detectable from `date_end`. These two checks cut the per-cycle session count from 126 to ~21.

### 2. Treat 404 as benign

When OpenF1 *does* return 404 — for a session that exists in the schedule but has no published results yet — log it at Debug, not ERROR. Counted as `skipped_no_results`. The 404 cascade was a real signal earlier (sessions that should have had data didn't), but now that the future and cancelled cases are filtered out, a 404 means "OpenF1 hasn't published this yet, try again next cycle."

```go
if errors.Is(err, errNoResultsYet) {
    skippedNoResults++
    p.logger.Debug("session results not yet published", "session_key", raw.SessionKey)
    continue
}
```

### 3. Lock finalized sessions

Once a session has ended *and* the results have been successfully fetched *and* it's been ≥2 hours since the session ended (the FIA can issue penalties up to ~90 min post-session), mark it `finalized=true` in Cosmos with a `schema_version`. On every subsequent poll cycle, those sessions are skipped before any HTTP call:

```go
if cachedVer, isFinalized := finalizedKeys[raw.SessionKey]; isFinalized && cachedVer >= SessionSchemaVersion {
    skipped++
    continue
}
```

Bumping `SessionSchemaVersion` invalidates every cached document and forces a re-fetch — the backfill mechanism for whenever I add new fields to the result shape. Setting `INGEST_FORCE_REFRESH_SESSIONS=1` does the same per-deploy without a code change.

After PR #16 deployed, the first poll cycle reported:

```json
{"msg":"session poll complete","sessions":126,"processed":14,"skipped_finalized":7,"skipped_cancelled":10,"skipped_future":95,"skipped_no_results":0}
```

14 sessions actually fetched, 95 skipped before they ever became HTTP requests, 10 cancelled, 7 already locked. No 429s. No 404 noise.

The next cycle, five minutes later:

```json
{"msg":"session poll complete","sessions":126,"processed":0,"skipped_finalized":21,"skipped_cancelled":10,"skipped_future":95}
```

**Zero OpenF1 calls.** Twenty-one finalized sessions sitting peacefully in Cosmos, getting served to the UI from cache. That's the steady state until a new race weekend kicks off.

---

## What I Took From This

**1. The shape of the API matters more than the URL of the API.** I spent a non-trivial amount of time staring at JSON responses thinking "where's the points field?" before I realized I was looking at a fundamentally different resource. Two endpoints can return data that *looks similar enough* to fool you, especially when the failure mode is "fields are silently `null`" instead of "wrong endpoint." Curl your upstream API and verify the response shape *before* writing the transform.

**2. Rate-limit cascades hide in success metrics.** The poller logged "session poll complete" every cycle. The metric that mattered was *how many sessions actually succeeded inside that cycle*. A poll that processes 0 useful sessions because everything 429'd is technically "complete." I added explicit `processed`/`skipped_*` counters to the completion log so future me can see the cascade pattern at a glance.

**3. "Iterate everything every cycle" is a bug that scales with the season.** This worked fine in week 1 when there were 6 pre-season test sessions and 0 future sessions. It worked OK in week 2 with ~12 sessions. By Round 3 it was hitting OpenF1 hard enough to trip the rate limit. The complexity I needed wasn't fancier rate limiting — it was *not making the calls at all.* Cancelled and future sessions don't ever need to be polled. Finalized sessions don't either, except once.

**4. Schema-versioned caching is cheap insurance.** It took maybe 30 minutes to thread `schema_version` through the data model and cache check. The payoff is enormous: any future change to the result shape becomes a one-line constant bump that automatically backfills the entire season on the next poll. Without it I'd be scripting one-off Cosmos rewrites every time I add a field.

**5. A 2-hour finalization buffer is the right balance.** Too short and we lock in unofficial results before stewards finish post-race investigations. Too long and we keep hammering the API for nothing. Two hours covers virtually every real-world FIA penalty window. (Abu Dhabi 2021 would have needed longer, but that's a once-a-decade exception, not a default.)

**6. Three small PRs beat one heroic PR.** Each fix made the next bug visible. If I'd tried to "do it all at once" I'd never have spotted the rate-limit cascade — the empty-fields bug was hiding it. Merge the obvious fix, deploy, observe what's still wrong, repeat.

---

## What's Live Now

- **Feature 003 (Race Session Results)**: complete. Race, qualifying, and practice tables all render with correct data. 21 sessions finalized in cache, 95 future sessions waiting, 10 cancelled sessions skipped forever.
- **Steady-state OpenF1 traffic**: 0 requests per cycle for finalized weekends. Roughly one cycle of fetches per new session that ends.
- **Test counts**: 38 frontend tests, 22 backend tests (contract + integration + unit). All green.

Next up: probably starting Feature 004, or possibly a long-overdue look at observability — Azure Monitor dashboards exist in the repo but nobody has loved them yet.

For now, the rate-limit cascade is dead. The race page renders. I'm calling it a day.

---

**Live:** http://f1raceintel.westus3.cloudapp.azure.com/
**Round 3 (Australian GP):** http://f1raceintel.westus3.cloudapp.azure.com/rounds/3?year=2026
**API:** http://f1raceintel.westus3.cloudapp.azure.com/api/v1/rounds/3?year=2026

[← Day 9: The Struggle Bus to a Private Cosmos DB](day-9-cosmos-private-endpoint.md)
