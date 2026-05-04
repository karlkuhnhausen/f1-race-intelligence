# Research: Standings Overhaul

## Decision 1: OpenF1 Championship Endpoint Availability and Behavior

- **Decision**: Use OpenF1 beta endpoints `/v1/championship_drivers` and `/v1/championship_teams` as the primary data source, replacing the fictional Hyprace integration.
- **Rationale**: Both endpoints return real data for 2025 and 2026 sessions. They provide `points_start`, `points_current`, `position_start`, `position_current` per driver/team per race session — exactly what's needed for progression charts and current standings.
- **Alternatives considered**:
  - Compute standings from `/v1/session_result` position data using the F1 points system: rejected because it duplicates complex logic (sprint points, fastest lap bonus, half points for shortened races) and OpenF1 already computes this.
  - Use Ergast/Jolpica API: rejected because it adds a new external dependency outside the constitutional scope (OpenF1-only).
- **Research findings**:
  - Confirmed working for 2025 sessions (e.g., session_key=9839) and 2026 sessions (e.g., session_key=11234, 11275).
  - Championship data is available for both Race and Sprint sessions.
  - For the first race of a season, `position_start` and `points_start` may be `null` (no prior standing). Backend must handle null → 0 coercion.
  - Rate limit: ~1 req/s applies as with all OpenF1 endpoints.
  - Beta stability risk: endpoints could change. Mitigation is graceful error handling and cached data fallback.

## Decision 2: Teams Championship Endpoint Schema Quirks

- **Decision**: Handle `null` team names in `/v1/championship_teams` responses by cross-referencing with `/v1/drivers` data for team identity resolution.
- **Rationale**: The 2026 Australia race (session_key=11234) showed `team_name: null` for positions 1 and 2. This appears to be a data completeness issue in early 2026 data. Later sessions (e.g., 11275 Miami Sprint) have complete team names.
- **Alternatives considered**:
  - Skip rows with null team_name: rejected because these are real championship entries (positions 1 & 2 had points).
  - Fail the entire ingestion: rejected for poor UX.
- **Resolution**: When `team_name` is null in the championship response, resolve it by looking up the drivers at that meeting (from `/v1/drivers?meeting_key=X`) and matching by points totals. Store resolved team name. Log a warning for transparency.

## Decision 3: Starting Grid / Pole Position Data Source

- **Decision**: Use `/v1/starting_grid?meeting_key={meeting_key}&position=1` to determine pole position per race, not by qualifying session_key.
- **Rationale**: The `starting_grid` endpoint is keyed by the qualifying `session_key`, but querying by `meeting_key` returns the grid for that meeting regardless of session. This is simpler than finding the qualifying session_key for each race.
- **Research findings**:
  - `starting_grid?session_key={race_session_key}` returns 404 (Not Found) — it only works with qualifying session keys.
  - `starting_grid?meeting_key={meeting_key}` works and returns the grid sorted by position.
  - Confirmed working for 2025 (meeting_key=1276) and 2026 (meeting_key=1279).
  - Returns: `position`, `driver_number`, `lap_duration`, `meeting_key`, `session_key` (of the qualifying session).
- **Important**: For sprint weekends, the starting grid is for the main race only. Sprint grids are not relevant for the "poles" stat (pole position refers to main race grid P1).

## Decision 4: Session Result Data for Stats Derivation

- **Decision**: Use `/v1/session_result?session_key={key}` to derive wins, podiums, and DNFs per driver per race.
- **Rationale**: This endpoint provides `position`, `driver_number`, `dnf`, `dns`, `dsq`, `points`, `number_of_laps` per race result. All required stats can be computed from this data.
- **Research findings**:
  - Confirmed working for 2026: `session_result?session_key=11234` returns all drivers with position, dnf status, points.
  - Sprint session results also available (session_key=11275, Miami Sprint).
  - `dnf=true` clearly indicates a driver did not finish — use this for DNF count.
  - Sprint wins/podiums should count toward stats (they are separate competitive sessions).
- **Stats derivation logic**:
  - Wins: `position == 1` in session_result for Race and Sprint sessions.
  - Podiums: `position <= 3` in session_result for Race and Sprint sessions.
  - DNFs: `dnf == true` in session_result for Race and Sprint sessions.
  - Poles: `position == 1` in starting_grid for the meeting (main race only).

## Decision 5: Data Ingestion Trigger and Polling Strategy

- **Decision**: Extend the existing session poller's post-session finalization hook to trigger championship data ingestion after Race and Sprint sessions are finalized (2-hour buffer).
- **Rationale**: The session poller already detects when sessions end and waits 2 hours for official results. Championship data should be fetched at the same time since it's available from OpenF1 only after results are official.
- **Alternatives considered**:
  - Separate 5-minute polling loop (like old Hyprace): rejected because championship data only changes after races, not continuously. Polling every 5 minutes is wasteful.
  - Manual trigger only: rejected because it defeats the purpose of automation.
- **Implementation**:
  - When the session poller finalizes a Race or Sprint session, additionally call the championship ingestion module.
  - Fetch `/v1/championship_drivers?session_key={key}` and `/v1/championship_teams?session_key={key}`.
  - Fetch `/v1/session_result?session_key={key}` (may already be fetched by Feature 005/006 ingestion — reuse if present).
  - Fetch `/v1/starting_grid?meeting_key={meeting_key}` (for Race sessions only, to get pole position).

## Decision 6: Cosmos DB Storage Schema for Championship Snapshots

- **Decision**: Store championship snapshots as individual documents per driver/team per session in the existing `standings` container, partition key = `season`. Store session results and starting grids in the existing `sessions` container alongside other session data.
- **Rationale**: Reusing the existing `standings` container avoids creating new Cosmos containers and stays aligned with the current architecture. The partition key (`season`) supports all query patterns (get all snapshots for a season, get latest snapshot).
- **Alternatives considered**:
  - New `championship` container: rejected because it adds infrastructure complexity and the data volume is small (~20 drivers × ~24 races × 4 years = ~2000 documents).
  - Single aggregated document per season: rejected because it makes per-race progression queries harder and prevents incremental updates.
- **Document IDs**:
  - Driver championship: `{season}-champ-driver-{session_key}-{driver_number}` (e.g., `2026-champ-driver-11234-63`)
  - Team championship: `{season}-champ-team-{session_key}-{team_name_slug}` (e.g., `2026-champ-team-11234-mclaren`)
  - The old `{season}-driver-{position}` format (Hyprace) will be deprecated/deleted.

## Decision 7: Historical Season Backfill Strategy

- **Decision**: Extend the existing backfill CLI with a `--championship` flag that fetches championship data for all completed race sessions in a given season. On-demand backend fetch for historical years not yet cached.
- **Rationale**: 2023–2025 data needs to be backfilled once. The existing backfill CLI pattern (Feature 006) already handles rate-limited sequential fetching.
- **Implementation**:
  - CLI: `go run cmd/backfill/main.go --season=2025 --championship`
  - Backend fallback: When a year is requested that has no cached data, trigger a synchronous backfill from OpenF1 (with caching) and return results. Cap this at a reasonable timeout (30s). If it times out, return empty with a message.
- **Rate limiting**: 500ms between requests during backfill. For a full season (~24 races × 4 endpoints = ~96 requests), this takes ~48 seconds.

## Decision 8: Recharts Usage for Progression and Comparison Charts

- **Decision**: Reuse `recharts` (already in `package.json` as `^3.8.1`, justified for Feature 006) for progression line charts and head-to-head overlay charts.
- **Rationale**: No new dependency needed. Recharts `<LineChart>`, `<Line>`, `<Tooltip>`, `<Legend>` components are well-suited for multi-series time-sequence data.
- **Alternatives considered**:
  - Chart.js / react-chartjs-2: rejected because adding a second charting library contradicts dependency discipline.
  - D3 direct: rejected as over-engineering for standard line charts.
- **Chart architecture**:
  - Progression: `<LineChart>` with one `<Line>` per driver/team, `stroke` set to team color, `dataKey` = points.
  - Comparison overlay: Same `<LineChart>` with exactly 2 `<Line>` components, highlighted with thicker strokes.
  - Responsive: Use `<ResponsiveContainer>` wrapper for mobile support.

## Decision 9: Hyprace Removal Scope

- **Decision**: Delete all Hyprace code and references. No migration path — clean removal.
- **Rationale**: Hyprace never returned real data. The standings container in Cosmos DB has zero Hyprace-sourced documents in production. There is nothing to migrate.
- **Removal checklist**:
  - `backend/internal/standings/hyprace_client.go` — DELETE entire file
  - `backend/cmd/api/main.go` — remove `HypraceClient` instantiation and goroutine launch
  - `backend/internal/storage/cosmos/client.go` — remove `source = 'hyprace'` filter from queries
  - Spec/plan references in `specs/002-*` — leave as historical record (do not modify old specs)
  - Any egress firewall rules for `api.hyprace.com` in Bicep/Helm — remove if present
  - Key Vault secrets for Hyprace — remove references if present in Bicep/Helm values

## Open Questions (all resolved)

None. All technical decisions resolved through API exploration and architecture analysis.
