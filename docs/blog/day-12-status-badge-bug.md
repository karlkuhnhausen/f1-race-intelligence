# Day 12: The Bug That Said "Completed" When the Race Hadn't Started

*Posted April 27, 2026 · Karl Kuhnhausen*

---

The dashboard looked great after [Day 11](day-11-design-system.md). Real F1 colors, real F1 typography, team accents on every row. Then I clicked into a round that hadn't happened yet — Round 8, weeks away — and every session card had a green **COMPLETED** badge sitting under it. Practice 1: Completed. Qualifying: Completed. Race: Completed. With no results below them.

The dashboard was telling me a future race was already over.

This post is about what that bug actually was, why it had been hiding behind the design refactor for two days, and how the fix is a small but instructive case study in *where* a status field belongs in a system.

---

## Two bugs in a trench coat

When I dug in, the issue wasn't one bug. It was two, stacked on top of each other.

**Bug #1 — the transform hardcoded the wrong status.** In `backend/internal/ingest/session_transform.go`, the function that converts an OpenF1 session into our Cosmos document looked like this:

```go
return storage.Session{
    SessionType:  string(sessionType),
    Status:       "completed",   // ← this line
    DateStartUTC: dateStart,
    DateEndUTC:   dateEnd,
    ...
}
```

That `"completed"` was a literal string. Every session, regardless of its date, got written to Cosmos with status `completed`. The original author (me, three days ago) presumably reasoned that the poller only fetched *past* sessions, so by the time a session reached `TransformSession`, it was definitionally complete. That reasoning is half right — and the other half is bug #2.

**Bug #2 — the future-skip guard had a silent failure mode.** The poller is supposed to skip sessions that haven't ended yet:

```go
if dateEnd, err := time.Parse(time.RFC3339, raw.DateEnd); err == nil && dateEnd.After(now) {
    skippedFuture++
    continue
}
```

Read that carefully. The condition is `err == nil AND dateEnd.After(now)`. If `time.Parse` fails — say, because OpenF1 returned `null` for `date_end` on a session that hasn't been scheduled yet — then `err != nil`, the whole condition is false, and the session **falls through to be written**. With status `"completed"`. With no results, because `/v1/session_result` returns 404 for sessions that haven't run.

The two bugs needed each other. Bug #1 is harmless if the poller never lets a future session reach the transform. Bug #2 is harmless if the transform writes the correct status. Together: every future session in Cosmos is labeled `completed` with no data.

Why didn't I catch this earlier? Because for the first four rounds of the season, every weekend's `date_end` was well-formed. The bug only manifested for late-season sessions where OpenF1 hadn't published the precise schedule yet. The dashboard quietly began misrepresenting the future as I scrolled forward in the calendar. The design refactor in Day 11 made it visible because now the badge was a vivid green chip instead of grey text — impossible to miss.

---

## Where does status belong?

Once I had the diagnosis, the interesting question was *where* to fix it. There were three plausible options:

1. **Fix it at write time** — make the transform compute status from dates, and patch the poller's parse-failure fall-through.
2. **Fix it at read time** — ignore whatever's in Cosmos, derive status in the API service from `now` vs the session's start/end times.
3. **Both.**

I ended up with option 3, but the reasoning is what matters.

If you only fix at write time, every existing row in Cosmos still has the wrong status. You either need a backfill (re-run the poller with `INGEST_FORCE_REFRESH_SESSIONS=1`) or you accept that the bug persists for already-cached sessions until they're re-fetched. Cosmos doesn't get re-fetched on a steady cadence — Day 10's whole point was to drive OpenF1 traffic toward zero by finalizing sessions. Stale `completed` rows would sit there indefinitely.

If you only fix at read time, the ingest layer is still writing wrong data. New rows enter the database with the same broken value. The bug is masked but not gone, and any future consumer that reads from Cosmos directly — a backfill script, an analytics export, a different microservice — gets the wrong answer.

**Status is a derived value.** It's a function of two inputs: the current time, and the session's scheduled times. Storing it in Cosmos is a denormalization — a cache, basically — and like all caches it needs an authoritative source to fall back on. The authoritative source here is the dates and the clock, not the cached field.

So the right architecture is: **the API service derives status at read time from dates** (this is the source of truth), **and** the ingest layer writes the correct value too (this keeps the cached form honest for any future direct readers). The read-time computation makes the fix immediate — users see the corrected badge as soon as the new code deploys, with no Cosmos backfill required.

The shape of `deriveSessionStatus` is satisfyingly small:

```go
func deriveSessionStatus(now, dateStart, dateEnd time.Time) string {
    if dateStart.IsZero() {
        return statusUpcoming
    }
    if dateStart.After(now) {
        return statusUpcoming
    }
    if dateEnd.IsZero() || dateEnd.After(now) {
        return statusInProgress
    }
    return statusCompleted
}
```

Three rules, four branches, twelve lines. Pure function. Trivially testable. The same logic lives in the transform with a different name (`deriveStoredStatus`) so the cached field eventually catches up to truth on the next ingest cycle.

---

## The other small thing: nomenclature drift

Feature 003's `data-model.md` defined the status enum as `upcoming | in_progress | completed | not_available`. Somewhere along the way the frontend started using `live` for the in-progress state. Probably because "Live" reads better in a UI badge than "In Progress." The frontend was right about the user-facing label and wrong about the internal name.

The fix: keep the spec's name (`in_progress`) on the wire, do a one-line label map at the render layer:

```typescript
const statusLabel =
  session.status === 'completed'   ? 'Completed' :
  session.status === 'in_progress' ? 'Live'      :
  session.status === 'upcoming'    ? 'Upcoming'  :
  session.status;
```

This is the right division. The contract uses the spec's enum. The UI uses whatever phrase is clearest to a human. Translating between them is a presentation concern, not a domain concern.

---

## The unrelated thing the merge revealed

After the PR merged and CI ran the deploy job, I went to run `kubectl` and got a connection refused. Ran the IP-sync skill script:

```
Detected current public IP: <REDACTED>/32

Fetching current AKS authorized IP ranges...
Current ranges: (none — API server is open to all)
```

That's a security incident. The AKS API server had no IP allowlist. Anyone on the internet could attempt to reach it.

The cause was inside the deploy workflow. Look at the cleanup step that runs at the end of every deploy job:

```yaml
- name: Remove runner IP from AKS API server allowlist
  if: always()
  run: |
    az aks update \
      --api-server-authorized-ip-ranges "${{ secrets.ADMIN_IP_RANGES }}"
```

The intent is to remove the GitHub Actions runner's ephemeral IP from the allowlist after each deploy, leaving only the persistent admin IPs. But if the `ADMIN_IP_RANGES` secret is empty or missing, that command resolves to `--api-server-authorized-ip-ranges ""`. Azure interprets the empty string as "remove all restrictions."

It's a classic shell-substitution trap dressed up in YAML. The fix at the script level is to update the secret (which the IP-sync script does automatically when it adds your new IP). The structural fix is harder: the cleanup step should validate that `ADMIN_IP_RANGES` is non-empty before running, and bail out if it isn't, rather than silently opening the cluster to the world.

That hardening is going on the next sprint's list. For now, the IP is locked back down to my home address and the secret is repopulated.

A second small bit of session-startup hygiene came out of this: I added a VS Code task with `runOn: "folderOpen"` so `sync-ip.sh` runs automatically every time I open the workspace. Two things now happen the moment I start work — the AKS allowlist gets verified, and the GitHub secret gets refreshed. I stop being the weakest link in the loop.

---

## Two small bugs, three small lessons

1. **A status field is a cache, not a source of truth.** If the inputs that determine status are already in your row (start time, end time, current time), then storing the status field is a denormalization. It can drift. Treat it like any other cache: derive on read, refresh on write, and don't trust either form unless you've checked the inputs.
2. **`err != nil` should never be a fall-through.** When parsing dates from a third-party API that occasionally returns nulls, "the parse failed" is not a synonym for "the date is in the past." It is a synonym for "I don't know." The correct response is to fall back to a related signal (here: `date_start`), or to skip defensively, not to let control flow continue as if the check passed.
3. **An empty string is a wildcard.** Bash, YAML, Azure CLI — they all treat `""` as a value, and many tools treat that value as "match everything." Any cleanup step that resets a security-critical configuration from a secret needs to validate the secret is populated before running. `if: always()` plus a missing secret is how you accidentally publish your API server.

---

## What's live now

- **PR #20** ([fix(rounds): show correct session status for future rounds](https://github.com/karlkuhnhausen/f1-race-intelligence/pull/20)) — merged, deployed.
- **PR #21** ([chore: auto-approve git checkout and git pull in chat terminal](https://github.com/karlkuhnhausen/f1-race-intelligence/pull/21)) — small DX win merged the same hour.
- **AKS allowlist** — restored to my home IP, secret repopulated.
- **VS Code folder-open task** — `sync-ip.sh` now runs automatically when I open this workspace.
- **Tests**: 22 backend + 48 frontend, all green. New unit tests for `deriveSessionStatus`, `deriveStoredStatus`, `isFutureSession`, and the read-time override.

The dashboard now correctly reports that a race that hasn't happened hasn't happened. Sometimes a feature is just convincing the system to admit what it already knows.

---

[← Day 11: From Generic Data Table to Race Car](day-11-design-system.md) · [Day 13: An Empty String Is a Wildcard →](day-13-allowlist-guard.md)
