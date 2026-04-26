# Day 7: Race Results, Rate Limits, and the Branch You Forgot You Were On

*Posted April 20, 2026 · Karl Kuhnhausen*

---

Here's the thing about AI-assisted co-generation: the code writes itself, but the git branches don't manage themselves.

This session delivered Feature 003 Phase 4 — a dedicated `RaceResults` component with DNF/DNS/DSQ separation, fastest lap badges, and proper race time formatting. It also diagnosed and fixed an OpenF1 rate limiting problem that had been silently corrupting driver data since the first deploy. But the more interesting story is what happened between the two: I accidentally created an entire new feature specification on the wrong branch, got confused about which tasks were executing where, and had to untangle the mess.

---

## The Branch Problem

Here's how it happened. Feature 003 (Race Session Results) was partially complete — Phases 1–3 deployed, Phases 4–7 remaining. I got excited about what the dashboard would look like with proper branding and asked Claude to spec out Feature 004: Design System and Brand Identity. Claude ran `speckit specify`, then `speckit plan`, then `speckit tasks` — generating a full 37-task implementation plan.

All of that landed on a new `004-design-system-brand` branch. Fine so far.

The problem came when I went back to continue Feature 003. The agent had created the 004 branch from master after the Phase 1–3 merge, so Feature 003's Phase 4 work needed its own branch. Claude created `003-race-results-phase4` from master and implemented the `RaceResults` component there. That merged cleanly.

But then I looked at the branch list:

```
  003-race-session-results          ← original 003 branch (Phases 1-3, stale)
  003-race-results-phase4           ← Phase 4 work (merged to master)
  003-race-session-results-phase5-7 ← Phases 5-7 (current)
  004-design-system-brand           ← Feature 004 spec/plan/tasks
  master                            ← deployed
```

Five branches for two features. The 004 branch had spec artifacts *and* a `vite-env.d.ts` fix that was made while debugging a TypeScript error — a fix that should have been on master but was committed to the 004 branch because that's where we happened to be at the time. The 003 original branch was stale. The Phase 4 branch was merged and should be deleted.

When you're pair-programming with an AI agent, the agent doesn't have a mental model of your branch strategy. It does what you ask. If you say "create a new feature spec," it runs Spec Kit commands on whatever branch is checked out. If you say "now go back and finish Feature 003," it creates a new branch and starts working. The branches multiply because the agent is responsive, not strategic.

**The lesson:** Before telling the agent to start a new feature, finish or checkpoint the current one. Switch to master. Make the branch cut intentional. The agent will faithfully execute `speckit specify` on whatever branch it's sitting on — and if that's the wrong branch, you'll spend time later untangling commits that don't belong together.

---

## Phase 4: The RaceResults Component

With the branch situation sorted, Phase 4 itself was straightforward. Three tasks: build the component, write tests, integrate it into the round detail page.

### Classified vs. Non-Classified

The key design decision was splitting race results into two sections. Formula 1 distinguishes between classified finishers (drivers who completed enough laps to be officially classified) and non-classified entries (DNF, DNS, DSQ, Retired). The component filters on a status set:

```typescript
const NON_CLASSIFIED_STATUSES = new Set(['DNF', 'DNS', 'DSQ', 'Retired', 'Disqualified']);
```

Classified finishers render in the main table body with position, time gap, and points. Non-classified entries appear below a "Not Classified" divider row showing their finishing status instead of a time gap. This matches the presentation format used in official FIA race classifications.

### Race Time Formatting

The race winner shows an absolute time (`1:32:45.678`), while every other driver shows a gap to the leader (`+5.432s`). The `formatRaceTime` helper handles the conversion from seconds to `h:mm:ss.sss`:

```typescript
function formatRaceTime(seconds: number | undefined): string {
  if (seconds == null) return '—';
  const hours = Math.floor(seconds / 3600);
  const mins = Math.floor((seconds % 3600) / 60);
  const secs = (seconds % 60).toFixed(3);
  if (hours > 0) return `${hours}:${String(mins).padStart(2, '0')}:${secs.padStart(6, '0')}`;
  if (mins > 0) return `${mins}:${secs.padStart(6, '0')}`;
  return secs;
}
```

### Fastest Lap

The driver who set the fastest lap gets a `⏱` badge next to their name and a highlighted row via the `fastest-lap` CSS class. This data comes from the OpenF1 API and is already part of the session result model — the component just needs to conditionally render it.

### Integration

`RoundDetailPage.tsx` gained a routing decision: if the session type is `race` or `sprint`, render `RaceResults`. Otherwise, use the generic `SessionResultsTable`. A `Set` of race types keeps the check clean:

```typescript
const RACE_TYPES = new Set(['race', 'sprint']);
```

Eight tests cover the component: classified finishers rendering, race time vs. gap display, fastest lap badge and row highlighting, DNF/DNS/DSQ separation under the "Not Classified" divider, laps completed, table headers, and empty results.

All 29 frontend tests pass. Phase 4 merged to master as commit `8085ffd`.

---

## The Ghost Data Bug

After deploy, I opened the dashboard expecting to see driver names and teams in the race results. Every cell was blank.

The results table rendered perfectly — correct positions, correct structure — but `driver_name`, `driver_acronym`, and `team_name` were all empty strings. Every session, every round, every driver.

The first instinct was to blame the new `RaceResults` component. It was the last thing that changed. But a quick `curl` to the API confirmed the data was empty at the source:

```json
{
  "position": 1,
  "driver_number": 63,
  "driver_name": "",
  "driver_acronym": "",
  "team_name": "",
  "finishing_status": "Finished"
}
```

The frontend was rendering exactly what the API returned. The problem was upstream.

---

## The 429 Wall

Backend logs told the story. The session poller iterates every session in the 2026 season — over 40 sessions across all practice, qualifying, and race events — and for each one, hits two OpenF1 endpoints: `/v1/positions` and `/v1/drivers`. That's 80+ HTTP requests in a tight loop with no delay.

OpenF1 rate-limits. The first few sessions ingested successfully. Then:

```
HTTP 429 Too Many Requests
HTTP 429 Too Many Requests
HTTP 429 Too Many Requests
```

The poller logged warnings but didn't stop. It continued upserting session results with empty driver data — overwriting any previously good records with blank fields. Every 5-minute poll cycle made it worse.

A direct curl to OpenF1 confirmed the data exists:

```bash
curl -s "https://api.openf1.org/v1/drivers?session_key=11465" | jq '.[0]'
```
```json
{
  "full_name": "George RUSSELL",
  "name_acronym": "RUS",
  "team_name": "Mercedes"
}
```

The data was there. We were just asking for it too fast.

### The Fix

Three changes to `session_poller.go`:

**1. 500ms delay between sessions.** After processing each session, the poller waits half a second before starting the next. Across 40+ sessions, this turns a 2-second burst into a 20-second crawl — well within rate limits.

```go
if i > 0 {
    select {
    case <-ctx.Done():
        return
    case <-time.After(500 * time.Millisecond):
    }
}
```

**2. 300ms delay before driver fetch.** Within each session, the positions fetch and driver fetch are now separated by 300ms.

**3. Skip upsert when driver data is missing.** If the driver fetch fails or returns empty data, the result is *not* written to Cosmos. This prevents overwriting good records from a previous successful poll.

```go
if driver == nil || driver.FullName == "" {
    p.logger.Debug("skipping result upsert: no driver data",
        "driver_number", pos.DriverNumber,
        "session_key", raw.SessionKey)
    continue
}
```

Both `select` blocks respect context cancellation — if the poller is shutting down, it doesn't block on the sleep. The delays are intentionally simple. A proper exponential backoff with retry would be more robust, but 500ms between requests is well under OpenF1's observed rate limit, and the poller runs every 5 minutes anyway. If a poll cycle partially fails, the next one picks up what was missed.

---

## The AKS Deployment Hiccup

Between Phase 4 and the rate-limiting fix, the CI/CD pipeline failed with:

```
Error: INSTALLATION FAILED: failed to create resource:
Internal error occurred: failed calling webhook "validate.nginx.ingress.kubernetes.io"
```

The AKS cluster auto-stops at midnight Pacific and auto-starts at 8 AM. The deploy ran during the startup window. The nginx ingress admission webhook wasn't ready yet — the ingress controller pod was still initializing while Helm tried to validate the ingress resource.

The fix was trivial: wait for the cluster to fully start, push an empty commit to re-trigger the pipeline. Not elegant, but effective. The real fix would be retry logic in the CI/CD pipeline or a readiness check before the Helm upgrade step.

---

## Lessons for AI Co-Generation

This session surfaced a pattern that I suspect will become common as more people build with AI agents:

**The agent is stateless about your workflow.** It doesn't remember your branching strategy, doesn't warn you that you're about to commit Feature 004 spec artifacts to a branch that should only have Feature 003 work, and doesn't suggest switching branches before starting a new feature. It executes the task you give it in the context it has.

**The human's job shifts from writing code to managing state.** When the agent writes 100% of the code, the human's role becomes: Which branch? Which feature? In what order? Is this the right context for this task? These are project management decisions, not programming decisions.

**Rate limits are invisible until they're not.** The session poller worked perfectly in testing with a handful of sessions. It only failed at scale — 40+ sessions, 80+ API calls, one angry rate limiter. The agent wrote the poller correctly; the missing piece was operational knowledge about OpenF1's rate limiting behavior that only surfaced in production.

---

## What's Next

Phase 4 is deployed. The rate-limiting fix is deployed. Driver data should repopulate on the next poller cycle.

Phases 5–7 remain:

- **Phase 5 (US2)**: Qualifying results with Q1/Q2/Q3 segment times
- **Phase 6 (US3)**: Practice session results with best lap times and gaps
- **Phase 7**: Polish — unit tests for transform logic, integration tests, network boundary enforcement

And Feature 004 — the Design System and Brand Identity — has 37 tasks waiting on its branch. The spec, plan, and task list are generated. The implementation hasn't started. One feature at a time.

**Frontend:** http://f1.20.171.233.61.nip.io/
**Round Detail Example:** http://f1.20.171.233.61.nip.io/rounds/3?year=2026
**API:** http://api-f1.20.171.233.61.nip.io/api/v1/rounds/3?year=2026
**Source:** https://github.com/karlkuhnhausen/f1-race-intelligence

---

*Previous: [Day 6: Clicking Into the Race — Session Results & Round Detail](day-6-race-session-results.md)* | *Next: [Day 8: The Security Alert I Got at 5 AM — And What I Did About It](day-8-ops-security-lockdown.md)*
