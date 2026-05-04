# Feature Specification: Standings Overhaul

**Feature Branch**: `007-standings-overhaul`  
**Created**: 2026-05-03  
**Status**: Draft  
**Input**: User description: "Replace the fictional Hyprace standings integration with real OpenF1 championship data and enhance the standings page with rich analytics. (1) Rip out all Hyprace dead code — the poller client, its references in config, egress rules, Key Vault secret references, and any Hyprace-specific types — replacing it with ingestion from the OpenF1 beta endpoints /v1/championship_drivers and /v1/championship_teams, which provide per-race points_start, points_current, position_start, and position_current keyed by session_key. Join with /v1/drivers for driver names, team names, and team colors. Backfill all completed 2026 race sessions. (2) Standings progression charts — interactive line charts showing each driver's and constructor's cumulative points after every race in the season, derived directly from the per-race points_current values stored from each session. (3) Expanded statistics — add wins (position=1 from /v1/session_result), podiums (position<=3), DNFs (dnf=true), and poles (position=1 from /v1/starting_grid) columns to both standings tables, computed from session result data already ingested or newly fetched. (4) Historical season selector — a year picker letting users browse standings for past seasons (2023 onward, matching OpenF1 data availability). (5) Head-to-head comparisons — select two drivers or two constructors for a side-by-side panel with stats deltas and a progression overlay chart. (6) Constructor driver breakdown — expanding a constructor row shows each driver's individual point contributions, wins, and podiums toward the constructor total."

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Real Championship Standings from OpenF1 (Priority: P1)

As an F1 fan visiting the standings page, I can see accurate, up-to-date driver and constructor championship standings sourced from real OpenF1 data — so I no longer see empty tables and can trust the numbers match the official championship.

**Why this priority**: The current standings page shows zero rows because it polls a fictional API. Without real data, every other enhancement (charts, comparisons, breakdowns) is useless. This is the foundation.

**Independent Test**: Navigate to the standings page for the 2026 season. Both the Drivers and Constructors tabs display rows with position, driver/team name, team color, and points. Data matches what OpenF1 reports for the most recent completed race session.

**Acceptance Scenarios**:

1. **Given** the backend has ingested championship data from OpenF1 for at least one 2026 race session, **When** a user opens the Drivers standings tab, **Then** the table shows one row per driver with position, full name, team name, team color accent, and current points — sorted by position ascending.
2. **Given** the backend has ingested championship data from OpenF1, **When** a user opens the Constructors standings tab, **Then** the table shows one row per team with position, team name, and current points — sorted by position ascending.
3. **Given** a new race session completes and its championship data becomes available on OpenF1, **When** the backend's next poll cycle runs, **Then** the new standings snapshot is ingested and the frontend reflects updated positions and points within the configured poll interval.
4. **Given** the OpenF1 championship endpoints are temporarily unavailable, **When** a user opens the standings page, **Then** the last successfully cached standings are shown with a `data_as_of_utc` timestamp indicating data freshness.
5. **Given** all Hyprace code has been removed, **When** the backend starts, **Then** no HTTP requests are made to any `hyprace.com` domain and no Hyprace-related configuration, types, or imports exist in the codebase.

---

### User Story 2 — Expanded Statistics Columns (Priority: P1)

As an F1 fan studying the championship, I can see wins, podiums, DNFs, and pole positions alongside points in the standings tables — so I can understand each competitor's season story beyond just the points total.

**Why this priority**: Stats like wins, podiums, and DNFs are the most requested data points fans look for alongside points. They add significant value to the existing table layout with minimal UI complexity.

**Independent Test**: Navigate to the Drivers standings tab. Each row shows columns for wins, podiums, DNFs, and poles in addition to position, name, team, and points. Values are consistent with the number of race sessions completed.

**Acceptance Scenarios**:

1. **Given** session result data has been ingested for completed 2026 race sessions, **When** a user views the Drivers standings, **Then** each driver row includes: wins count (number of races finished P1), podiums count (finished P1–P3), DNFs count (races where the driver did not finish), and poles count (number of times on pole position from the starting grid).
2. **Given** session result data has been ingested, **When** a user views the Constructors standings, **Then** each team row includes: combined wins, combined podiums, and combined DNFs across both of its drivers.
3. **Given** a driver has zero wins, podiums, DNFs, or poles, **When** their row is displayed, **Then** the value shows as "0" (not blank or a dash).

---

### User Story 3 — Standings Progression Charts (Priority: P2)

As an F1 fan, I can view interactive line charts showing cumulative championship points after each race for every driver and constructor — so I can see momentum shifts, crossover moments, and the championship narrative over the season.

**Why this priority**: Progression charts are the signature visual for understanding a championship fight. They require per-race data which Story 1 already provides, making this a natural second step.

**Independent Test**: Navigate to the standings page and switch to the progression chart view. A line chart renders with one line per driver (or constructor), the x-axis labeled with race names, and the y-axis showing cumulative points. Hovering over a data point shows driver name, race name, and points.

**Acceptance Scenarios**:

1. **Given** championship data has been ingested for multiple race sessions in a season, **When** a user selects the Drivers progression chart view, **Then** an interactive line chart renders with one line per driver, each line colored by team color, the x-axis showing race round names in chronological order, and the y-axis showing cumulative points.
2. **Given** the chart is rendered, **When** a user hovers over a data point, **Then** a tooltip displays the driver name, race name, and exact points total at that round.
3. **Given** the chart is rendered, **When** a user views the Constructors progression chart, **Then** a similar chart renders with one line per constructor team.
4. **Given** a season has only one completed race, **When** the progression chart is shown, **Then** a single data point per competitor is displayed (the chart is still useful, not hidden or errored).

---

### User Story 4 — Historical Season Selector (Priority: P2)

As an F1 fan, I can select a past season (2023, 2024, 2025) from a year picker to view historical driver and constructor standings — so I can explore championship results from prior years.

**Why this priority**: Historical data is available from OpenF1 starting in 2023. A season selector unlocks this data with relatively simple backend changes (fetching by different session keys) and is broadly useful.

**Independent Test**: Navigate to the standings page. A year picker control is visible, defaulting to the current season. Select 2024. The standings tables and any visible progression charts update to show 2024 championship data.

**Acceptance Scenarios**:

1. **Given** the standings page is loaded, **When** a user sees the year selector, **Then** it defaults to the current year (2026) and offers selectable options for each year from 2023 to the current year.
2. **Given** a user selects a different year (e.g., 2024), **When** the selection changes, **Then** both the Drivers and Constructors standings tables update to show that season's championship data.
3. **Given** a user selects a historical year, **When** progression charts are available, **Then** the charts also update to show the selected season's race-by-race progression.
4. **Given** a user selects a year for which no data has been backfilled yet, **When** the backend receives the request, **Then** it triggers a fetch from OpenF1 for that season, caches the results in Cosmos DB, and returns them — or returns an empty state with a clear message if OpenF1 has no data for that year.

---

### User Story 5 — Head-to-Head Comparisons (Priority: P3)

As an F1 fan, I can select two drivers (or two constructors) and see a side-by-side comparison panel with stats deltas and an overlaid progression chart — so I can directly compare championship rivals.

**Why this priority**: Head-to-head comparison is a power-user feature that builds on top of the data from Stories 1–3. It adds engagement depth but is not essential for the core standings experience.

**Independent Test**: On the standings page, select two drivers from a comparison picker. A comparison panel appears showing both drivers' stats side-by-side with deltas, plus an overlay of their two progression lines on a single chart.

**Acceptance Scenarios**:

1. **Given** a user is on the Drivers standings tab, **When** the user selects two drivers from a comparison control, **Then** a comparison panel appears showing side-by-side: position, points, wins, podiums, DNFs, and poles for each driver, with computed deltas (e.g., "+15 points", "+2 wins").
2. **Given** the comparison panel is shown, **When** both drivers have progression data, **Then** an overlay line chart shows both drivers' cumulative point trajectories on the same axes, each colored by their team color.
3. **Given** a user is on the Constructors standings tab, **When** two constructors are selected, **Then** the comparison panel shows team-level stats (position, points, wins, podiums, DNFs) with deltas and an overlay progression chart.
4. **Given** a user has a comparison open, **When** they change the season via the year picker, **Then** the comparison updates to reflect the selected season's data for those same competitors (or clears if competitors don't exist in that season).

---

### User Story 6 — Constructor Driver Breakdown (Priority: P3)

As an F1 fan, I can expand a constructor's row in the standings table to see each of its drivers' individual point contributions, wins, and podiums — so I can understand how each driver contributes to the team's championship total.

**Why this priority**: This is a detail-drill feature that enriches the constructors view. It depends on per-driver data already available from Stories 1–2 and adds depth without requiring new data sources.

**Independent Test**: On the Constructors standings tab, click or expand a team row. A sub-table or inline detail shows each driver on that team with their individual points, wins, and podiums. The individual driver points sum to the team's total.

**Acceptance Scenarios**:

1. **Given** a user is viewing the Constructors standings, **When** they click or expand a constructor row, **Then** an inline detail shows each driver on that team with their individual position in the drivers standings, points, wins, and podiums.
2. **Given** the constructor breakdown is expanded, **When** the user inspects the numbers, **Then** the sum of the individual drivers' points equals the constructor's total points shown in the parent row.
3. **Given** a constructor has only one driver who participated in races (e.g., a reserve driver substituted), **When** the breakdown is shown, **Then** all drivers who scored at least one point or started at least one race are included.

---

### Edge Cases

- What happens when OpenF1 championship beta endpoints are deprecated or their schema changes? The backend must handle unexpected response shapes gracefully, log warnings, and continue serving last cached data.
- How does the system handle a race session that is red-flagged and declared with partial points (e.g., half points for shortened races)? The system trusts OpenF1's reported `points_current` values and does not compute points independently.
- What happens when a driver transfers teams mid-season? The driver's row shows their current team (from the most recent `/v1/drivers` data), and progression chart points are attributed to the driver regardless of team changes.
- What happens when a race result is modified post-session (e.g., disqualification)? The next poll cycle picks up the corrected championship data from OpenF1 and overwrites the stale snapshot.
- What happens when a sprint race awards points? Sprint session championship updates from OpenF1 are ingested the same way as main race sessions — the `session_key` determines the snapshot.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Backend MUST remove all Hyprace-related code: the polling client, Hyprace-specific domain types, configuration references, egress firewall rules, and Key Vault secret references. No code or config referencing Hyprace may remain after this feature is complete.
- **FR-002**: Backend MUST ingest driver championship standings from OpenF1 `GET /v1/championship_drivers?session_key={key}`, storing `driver_number`, `meeting_key`, `session_key`, `position_start`, `position_current`, `points_start`, and `points_current` per driver per race session.
- **FR-003**: Backend MUST ingest constructor championship standings from OpenF1 `GET /v1/championship_teams?session_key={key}`, storing `team_name`, `meeting_key`, `session_key`, `position_start`, `position_current`, `points_start`, and `points_current` per team per race session.
- **FR-004**: Backend MUST join championship data with driver identity data from OpenF1 `/v1/drivers` to resolve `driver_number` into full name, team name, and team color for API responses.
- **FR-005**: Backend MUST provide a drivers standings endpoint returning rows with `position`, `driver_name`, `team_name`, `team_color`, `points`, `wins`, `podiums`, `dnfs`, and `poles` for a given season.
- **FR-006**: Backend MUST provide a constructors standings endpoint returning rows with `position`, `team_name`, `team_color`, `points`, `wins`, `podiums`, and `dnfs` for a given season.
- **FR-007**: Backend MUST provide a standings progression endpoint returning per-race cumulative points for all drivers (or constructors) in a given season, suitable for rendering line charts.
- **FR-008**: Backend MUST derive `wins` (final position = 1), `podiums` (final position ≤ 3), `DNFs` (`dnf = true`), and `poles` (starting grid position = 1) from OpenF1 `/v1/session_result` and `/v1/starting_grid` data for the corresponding race sessions.
- **FR-009**: Backend MUST support a `year` parameter on all standings endpoints, allowing queries for seasons from 2023 to the current year.
- **FR-010**: Backend MUST provide a head-to-head comparison endpoint accepting two driver numbers (or two team names) and a season, returning side-by-side stats and per-race progression data.
- **FR-011**: Backend MUST provide a constructor breakdown endpoint returning each driver's individual points, wins, and podiums for a given team in a given season.
- **FR-012**: Backend MUST backfill championship, session result, and starting grid data for all completed 2026 race sessions upon deployment of this feature.
- **FR-013**: Backend MUST poll OpenF1 championship endpoints after each race session ends (using the session poller's existing post-session detection) and cache results in Cosmos DB before serving clients.
- **FR-014**: Frontend MUST consume only backend standings endpoints and MUST NOT call OpenF1 directly.
- **FR-015**: Frontend MUST render standings tables with all columns specified in FR-005 and FR-006.
- **FR-016**: Frontend MUST render interactive progression line charts with one line per competitor, colored by team color, with tooltip on hover showing competitor name, race name, and points.
- **FR-017**: Frontend MUST provide a year picker defaulting to the current season, allowing selection of any year from 2023 to the current year.
- **FR-018**: Frontend MUST render a head-to-head comparison panel when two competitors are selected, showing stats deltas and an overlay progression chart.
- **FR-019**: Frontend MUST render an expandable constructor row showing individual driver contributions when a constructor row is clicked or expanded.
- **FR-020**: All standings responses MUST include a `data_as_of_utc` timestamp for freshness transparency.

### Constitution Alignment *(mandatory)*

- **CA-001**: Feature runs on Go backend + React frontend + Cosmos DB on AKS. No new platform components introduced.
- **CA-002**: UI calls backend standings API endpoints only. No browser-side calls to OpenF1.
- **CA-003**: All OpenF1 championship, session result, starting grid, and driver identity data is cached in Cosmos DB before serving clients. Freshness defined by post-race poll cycle. No pass-through.
- **CA-004**: No new secrets required. OpenF1 is free-tier with no API key. Hyprace Key Vault references are removed (net reduction in secret surface). Managed Identity continues for Cosmos DB access.
- **CA-005**: HTTPS enforced at NGINX ingress (existing). Egress firewall already allows `api.openf1.org`. Hyprace egress rule removed (net reduction).
- **CA-006**: No new Kubernetes resources required. Existing backend and frontend Helm charts serve this feature. Helm values may add configuration for championship poll intervals.
- **CA-007**: CI/CD pipeline order unchanged: lint → test → build → push → deploy. Backfill runs as a post-deploy manual step using existing CLI pattern.
- **CA-008**: Championship ingestion emits structured JSON logs: session key, data type, row count, duration, errors.
- **CA-009**: No new backend dependencies. Frontend uses `recharts` (already justified and in use from Feature 006). No additional packages required.
- **CA-010**: All work traces to FR-001 through FR-020 and user stories 1–6 in this spec.

### Key Entities

- **DriverChampionshipSnapshot**: A point-in-time record of a driver's championship standing after a specific race session. Key attributes: driver number, session key, meeting key, points before race, points after race, position before race, position after race.
- **TeamChampionshipSnapshot**: A point-in-time record of a constructor team's championship standing after a specific race session. Key attributes: team name, session key, meeting key, points before race, points after race, position before race, position after race.
- **DriverSeasonStats**: Aggregated season statistics for a driver: total wins, podiums, DNFs, and poles — derived from session results and starting grids across all races in a season.
- **TeamSeasonStats**: Aggregated season statistics for a constructor team: combined wins, podiums, and DNFs across both drivers — derived from session results across all races in a season.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Standings tables display complete, accurate data for all drivers and constructors within one poll cycle after a race session ends.
- **SC-002**: Users can view standings for any season from 2023 to the current year with data appearing within 5 seconds of selecting a year.
- **SC-003**: Progression charts render all data points (up to ~20 drivers × ~24 races = ~480 points per season) without visible lag or jank on desktop and mobile browsers.
- **SC-004**: Head-to-head comparison panel shows stat deltas and overlay chart within 3 seconds of selecting two competitors.
- **SC-005**: Constructor driver breakdown displays individual contributions that sum exactly to the team's total points.
- **SC-006**: Zero references to Hyprace remain in the codebase — code, configuration, documentation, and infrastructure artifacts.
- **SC-007**: Frontend network inspection shows zero direct browser requests to OpenF1 in production mode.
- **SC-008**: All expanded stats columns (wins, podiums, DNFs, poles) are accurate against official F1 results for all backfilled 2026 sessions.

## Assumptions

- OpenF1 beta championship endpoints (`/v1/championship_drivers`, `/v1/championship_teams`) remain available and stable through the implementation period. If they are deprecated, a fallback plan computing standings from `/v1/session_result` position data and the F1 points system will be needed (out of scope for this feature unless the endpoints go down during development).
- OpenF1 data availability starts from the 2023 season. Years before 2023 are not selectable in the year picker.
- The OpenF1 rate limit (~1 request/second) applies. Backfill and polling must respect this with appropriate delays between requests.
- The `recharts` library (already in use for Feature 006 session analysis charts) is reused for progression and comparison charts. No new charting dependency is needed.
- Sprint sessions that award championship points produce championship snapshots on OpenF1 in the same format as main race sessions.
- Driver identity data (name, team, team color) is already ingested by the existing session poller and available in Cosmos DB. This feature joins against that data rather than re-fetching it.
- The existing session poller's post-session detection mechanism (used in Features 005 and 006) is extended to also trigger championship data ingestion after race and sprint sessions end.
- Pole position is determined from the `/v1/starting_grid` endpoint (position = 1 for the race session), not from qualifying results.
- The backfill CLI (existing from Feature 006) is extended with a championship backfill mode for all completed 2026 sessions.
