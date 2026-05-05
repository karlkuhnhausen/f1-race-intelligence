# Day 22: The Standings Overhaul — Real Data, Real Bugs, and a Credit Long Overdue

*Posted May 4, 2026 · Karl Kuhnhausen*

---

Feature 007 shipped today: the Standings Overhaul. If you'd visited the standings page before today, the numbers you were looking at were made up. Fabricated by a fictional integration called "Hyprace" — a placeholder API client that generated synthetic driver and constructor standings with no connection to any real race data. It was scaffolding from the project's founding sprint that outlived its welcome by about five features.

Today it's gone. Replaced by real championship data from [OpenF1](https://openf1.org/).

## What Was Hyprace

`hyprace_client.go` was 205 lines of dead weight. It defined an `openHypraceStandings()` function that fetched from `https://api.hyprace.dev/v1/standings` — a domain that doesn't exist — and transformed the results into driver and constructor rows. The backend polled it on a timer. The CI tests mocked it. The frontend rendered its fabricated numbers with team-color accents.

Every driver on the standings page had data. All of it fictional.

The Hyprace integration was always supposed to be temporary — a placeholder until the real championship endpoints were mature enough to use. OpenF1's `/v1/championship_drivers` and `/v1/championship_teams` endpoints launched as beta endpoints earlier this year, and with four rounds of 2026 data already in the bag, they're reliable enough to depend on.

The deletion was one line in the task list: `T001 Delete backend/internal/standings/hyprace_client.go entirely`. In practice it cascaded through 39 files and 4,008 insertions before it was done.

## What Replaced It

The new `ChampionshipIngester` is the heart of the feature. After a Race or Sprint session is finalized (2 hours post-session end), the session poller's existing finalization hook now calls `ChampionshipIngester.IngestSession(ctx, season, sessionKey, meetingKey)`. That triggers four sequential OpenF1 fetches:

1. `/v1/championship_drivers?session_key={key}` — championship standings snapshot after this session
2. `/v1/championship_teams?session_key={key}` — constructor standings snapshot after this session
3. `/v1/session_result?session_key={key}` — each driver's finishing position, DNF flag, and points for this specific race
4. `/v1/starting_grid?meeting_key={key}` — qualifying grid positions (for pole attribution)

Each of these gets cached in Cosmos DB under the `standings` container. The `championship_driver` and `championship_team` documents are keyed by `{season}-champ-driver-{session_key}-{driver_number}`, so each post-race snapshot is its own document — preserving the full championship history rather than overwriting a single current-standings document.

The standings API service then reads the **latest snapshot per driver** (the one with the highest `session_key`) for the current standings table, and reads **all snapshots in chronological order** for the progression charts.

A backfill CLI extension (`--championship` flag on `cmd/backfill/main.go`) populates historical sessions from 2023–2026 at 500ms between requests to respect OpenF1's rate limit.

## The New UI

The standings page gained five new components in this feature:

**`ProgressionChart`** — A recharts `LineChart` showing cumulative points per round for every competitor. One line per driver or constructor, colored by team. Hover any point to see the round name and exact points total. The toggle between table view and chart view lives in a tab bar at the top of the standings section.

**`YearPicker`** — A dropdown that lets you select any season from 2023 to the current year. Changing the year re-fetches all standings data and updates both the table and chart views simultaneously. On first load for a historical season that has no cached data, the service triggers a synchronous ingestion from OpenF1 before responding.

**`ComparisonPanel`** — Select any two drivers (or constructors) by clicking their rows in the table. A panel slides in below the table showing side-by-side stats with delta badges: "+15 pts", "+2 wins", "-1 DNF". A two-line progression overlay makes the gap visual across the full season. Deselect either competitor to dismiss the panel. The comparison clears automatically when you switch seasons if either competitor doesn't exist in the new season's data.

**`ConstructorBreakdown`** — Click any constructor row to expand it inline and see each driver's individual contribution: points, wins, podiums, and a percentage-of-team-total bar. Points sum to the team total.

**`StandingsTable` expansion** — The existing standings table now renders wins, podiums, DNFs, and poles as additional columns. Zero values display as `0` rather than blank — important because a driver with zero wins is a different state than a driver with missing data.

## What's Not Working Yet

Shipping a feature with a known bug is never the goal, but honesty matters more than a perfect narrative.

The wins, podiums, DNFs, and poles columns in the standings table are not calculating cumulatively for the 2026 season. They show correct values for individual sessions but don't sum correctly across all rounds.

Here's what the code is supposed to do: `StatsAggregator.GetDriverStats()` fetches all `championship_result` documents for the season from Cosmos and iterates over them, incrementing wins, podiums, and DNF counters for each entry. The logic is right — if 20 `championship_result` documents exist for a driver across 4 race sessions, the aggregator should sum all 20. The query `SELECT * FROM c WHERE c.season = @season AND c.type = 'championship_result'` returns everything for the partition in a single pass.

The problem is almost certainly in the data, not the code. The `IngestSession` hook fires post-finalization, so the only sessions that have had championship data ingested via the live poller are the ones that completed *after* Feature 007 was deployed today. The backfill CLI was added and tested, but hasn't been run against the 2026 season from inside the AKS cluster yet (Cosmos DB's private endpoint blocks the backfill from running anywhere outside the cluster). Until that backfill runs, most drivers have partial data — maybe one or two sessions worth of `championship_result` documents rather than the full four rounds.

The fix is operational, not code: run `go run cmd/backfill/main.go --season=2026 --championship` from inside the cluster. That will populate all four rounds, and the stats columns will reflect the correct cumulative totals.

That said, this *is* a gap in the feature — the on-demand backfill path (the service triggering synchronous ingestion for seasons with no cached data, implemented in T041) handles the case of zero data, not partial data. A partial-data state shows wrong numbers instead of a loading state, which is worse than showing nothing. That's a bug to fix in a follow-up.

## A Credit Long Overdue — OpenF1

Every piece of race data on this site comes from [OpenF1](https://openf1.org/). The lap times, the tire stints, the pit stop durations, the session results, the championship standings, the intervals — all of it flows through `api.openf1.org/v1`. The project has been live for weeks, and the site has no acknowledgment of that anywhere.

That needs to change. OpenF1 is a free, open API operated by volunteers. There's no API key required, no subscription fee, no rate-limit enforcement at the server side (though we rate-limit ourselves out of respect). The team behind it is scraping and normalizing F1 telemetry data in real time so that projects like this one can exist. Using their data without attribution is the kind of thing that discourages people from running open infrastructure in the first place.

The plan is to add a visible, permanent attribution to the application footer: something like "Race data provided by [OpenF1](https://openf1.org/) — free, open F1 data." It should link out, it should be visible on every page, and it shouldn't be buried in a `README.md` that most users will never read.

## The Follow-Up Commits

Three small fixes shipped after the main Feature 007 PR:

**`fix(backend): bump Dockerfile to golang:1.26`** — The `golangci-lint` typecheck step was failing in CI because the `go.mod` required Go 1.25 features but the Docker build image was still on `golang:1.24`. Bumped to `golang:1.26` to match the toolchain version and clear the lint errors.

**`fix(calendar): show podium during active race weekend`** — The calendar service was returning `null` for the podium finisher data during the period when a race weekend was actively ongoing (post-race but before the 2-hour finalization window closed). Fixed by reading the most recent available session result for the meeting and returning partial podium data when available, rather than waiting for full finalization.

**`fix(ingest): bump SessionSchemaVersion to 4`** — The session poller uses a schema version to detect documents that need re-processing. Adding the championship ingestion hook in Phase 2 introduced a new data contract for session documents, but the version number wasn't bumped. Without the bump, the poller was skipping already-finalized sessions that now needed to run through the championship ingestion path. Version 4 forces those documents to be reprocessed on the next poll cycle.

## What's Next

Two things:

1. Run the 2026 championship backfill from inside the cluster and verify the stats columns are correct.
2. Add the OpenF1 attribution to the application footer.

The stats bug and the missing attribution aren't separate cleanup items — they're part of what makes the feature actually complete. Feature 007 is shipped, but it's not finished.

---

[← Day 21: Analysis UX Polish — Making Four Charts Tell a Clearer Story](day-21-analysis-ux-polish.md) · [Day 23: The Round Numbers Were Lying — A Cancelled-Race Desync →](day-23-cancelled-round-desync.md)
