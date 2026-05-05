# Day 23: The Round Numbers Were Lying — A Cancelled-Race Desync

*Posted May 5, 2026 · Karl Kuhnhausen*

---

The standings page was showing race winners for rounds that haven't happened yet. Canada (Round 5) and Monaco (Round 6) had podium data — races still weeks away. Miami's sprint session was invisible. The data felt wrong because it *was* wrong, and the root cause traced back to a two-line gap between two pollers that should have agreed on what "Round 1" means.

## The Symptom

After adding Bahrain and Saudi Arabia to the cancellation overrides (both races were removed from the 2026 calendar), the cached data went haywire:

- Podium winners appearing for future rounds
- Miami's sprint race invisible in the round detail view
- Standings showing championship data attributed to wrong races

The site looked like it was hallucinating.

## The Root Cause

Two pollers assign round numbers independently:

1. **Meeting poller** (`NormalizeMeetings`) — fetches `/meetings`, filters out cancelled races, assigns sequential rounds. Australia = Round 1. ✓
2. **Session poller** (`buildMeetingRoundMap`) — fetches `/sessions`, groups by meeting key, assigns sequential rounds... but **only** excluded pre-season testing. It never checked the cancellation overrides. Bahrain = Round 1. ✗

So the meeting poller said Australia is Round 1. The session poller said Australia is Round 3 (because Bahrain and Saudi consumed slots 1 and 2). Every session document in Cosmos had a round number that was +2 off from what the meetings container reported.

The championship ingester — which runs after race/sprint finalization — stored standings snapshots keyed by these wrong round numbers. When the calendar API assembled podium data by joining meetings with session results on round number, it pulled results from sessions that belonged to future rounds.

The sprint session for Miami *existed* in Cosmos, but under the wrong round number, so the round detail page couldn't find it.

## The Fix

PR [#67](https://github.com/karlkuhnhausen/f1-race-intelligence/pull/67) — a targeted fix in three parts:

**1. Align session poller with meeting poller filtering**

Added `buildCancelledMeetingKeys()` to the session poller. Before computing round numbers, it now fetches `/meetings?year={season}` and builds a set of meeting keys to exclude — checking both the OpenF1 `is_cancelled` field and our domain `IsCancelled()` overrides. The existing `buildMeetingRoundMap()` function received a `cancelledKeys` parameter and skips those meetings in the same loop where it already skips testing meetings.

One additional API call per poll cycle. Acceptable cost for correctness.

**2. Bump `SessionSchemaVersion` to 5**

The session poller skips documents whose cached schema version matches the current code version. Bumping from 4 to 5 forces a full re-poll — every session re-fetched with correct round numbers. The session reconciliation logic (already in place from Feature 003) then auto-deletes orphaned session documents whose round numbers no longer match.

**3. Meeting reconciliation**

The meeting poller had no cleanup logic. If Bahrain was previously stored as `2026-01`, adding it to the cancellation list would leave a ghost document in Cosmos forever. Added `reconcileStaleMeetings()` — after upserting the normalized meeting set, it queries existing meetings for the season and deletes any whose ID isn't in the new set.

This required adding `DeleteMeeting` to the `CalendarRepository` interface and implementing it in the Cosmos client.

## The Data Refresh

After deploying, the schema version bump triggered automatic re-polling. The session poller processed all 20 completed sessions with correct round numbers. The reconciliation removed 16 stale session documents:

```
reconcile: removing stale session  session_id=2026-18-sprint-qualifying
reconcile: removing stale session  session_id=2026-18-sprint
reconcile: removing stale session  session_id=2026-11-sprint-qualifying
reconcile: removing stale session  session_id=2026-11-sprint
reconcile: removing stale session  session_id=2026-06-sprint-qualifying
reconcile: removing stale session  session_id=2026-06-sprint
...
```

Then a `--championship` backfill re-ingested standings for all three completed race/sprint sessions (Australia race, Japan race, Miami sprint) with correct round numbers. The calendar API immediately returned correct data:

```
Round 1: Australian Grand Prix — Winner: George RUSSELL
Round 2: Chinese Grand Prix — Winner: Kimi ANTONELLI
Round 3: Japanese Grand Prix — Winner: Kimi ANTONELLI
Round 4: Miami Grand Prix — Winner: Kimi ANTONELLI
```

Miami sprint is now visible in the round detail view.

## The Bug That Remains

Rounds 5 and 6 (Canada, Monaco) are still showing podium data despite being future races. This is a different issue — the championship backfill stored standings snapshots that reference future rounds, and the calendar's podium enrichment query isn't filtering by round status. The data says "here's what the latest championship snapshot shows" but the UI is presenting it as "here's who won this round." Separate fix needed.

## Lesson

When two independent processes assign the same semantic value (round numbers), they must use identical filtering logic. A change to one without updating the other creates a silent data integrity failure that manifests as confusing user-facing symptoms far from the root cause.

The fix was 236 lines of code across 11 files, a new unit test, and one additional API call per poll cycle. The bug existed because a cancellation override was added to the domain layer without auditing all consumers of round-number assignment. The meeting poller happened to use the domain function. The session poller happened not to. Both seemed to work correctly in isolation.

---

[← Day 22: The Standings Overhaul — Real Data, Real Bugs, and a Credit Long Overdue](day-22-standings-overhaul.md)
