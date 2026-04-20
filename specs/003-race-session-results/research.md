# Research: Race Results & Session Data

## Decision 1: OpenF1 endpoints for session data ingestion

- **Decision**: Use three OpenF1 endpoints in combination:
  1. `GET /v1/sessions?meeting_key={key}` — lists all sessions for a meeting (FP1, FP2, FP3, Sprint Qualifying, Sprint, Qualifying, Race) with `session_key`, `session_name`, `session_type`, `date_start`, `date_end`
  2. `GET /v1/session_result?session_key={key}` — final standings per session with `position`, `driver_number`, `duration`, `gap_to_leader`, `number_of_laps`, `dnf`, `dns`, `dsq`
  3. `GET /v1/drivers?session_key={key}` — driver metadata (full_name, team_name, name_acronym) to hydrate result rows with human-readable names
- **Rationale**: The `session_result` endpoint provides the definitive post-session results including position, duration (best lap for practice/qualifying, race time for races), gap to leader, and DNF/DNS/DSQ flags. The `drivers` endpoint provides the name and team association needed for display. Together these three endpoints satisfy FR-001 through FR-005 without requiring lap-by-lap or interval data processing.
- **Alternatives considered**:
  - Using `/v1/laps` to reconstruct results: Rejected — more complex, higher request volume, and `session_result` already provides finalized positions.
  - Using `/v1/position` for race results: Rejected — provides in-session position changes but not finalized results with gaps and status flags.
  - Using `/v1/intervals` for gap data: Not needed — `session_result.gap_to_leader` provides the same information in finalized form.

## Decision 2: Qualifying segment times (Q1/Q2/Q3) representation

- **Decision**: The `session_result` endpoint returns `duration` and `gap_to_leader` as arrays of three values `[Q1, Q2, Q3]` for qualifying sessions. Store these as three separate nullable fields (`q1_time`, `q2_time`, `q3_time`) in the Cosmos DB document and API response. Null indicates the driver did not participate in that segment.
- **Rationale**: The OpenF1 API returns qualifying data in this array format. Normalizing to three fields simplifies frontend rendering (display each column independently) and Cosmos DB queries. A driver eliminated in Q1 will have `q2_time: null, q3_time: null`.
- **Alternatives considered**:
  - Storing the raw array: Rejected — less readable in Cosmos DB queries and harder to index.
  - Separate documents per qualifying segment: Rejected — over-normalized for this read pattern.

## Decision 3: Fastest lap attribution for race results

- **Decision**: Derive fastest lap from `session_result` duration data. For the race session, fetch `/v1/laps?session_key={key}` and identify the minimum `lap_duration` across all drivers to flag the fastest lap holder. Alternatively, if the OpenF1 data does not include a direct fastest lap flag, compare lap times during ingestion.
- **Rationale**: The spec (FR-003) requires a fastest lap indicator. OpenF1's `session_result` does not directly include a fastest-lap flag, but `/v1/laps` provides individual lap durations from which the fastest lap can be derived.
- **Alternatives considered**:
  - Skip fastest lap entirely: Rejected — explicitly required by FR-003.
  - Hardcode from external sources: Rejected — violates the data residency principle.

## Decision 4: Session availability and status model

- **Decision**: Determine session availability by combining:
  1. Whether the session exists in the OpenF1 sessions response for the meeting
  2. Whether `session_result` data is available (non-empty) for that session
  3. Whether the session's `date_end` is in the past (completed) or future (not yet started)
  Map to four states: `completed`, `in_progress`, `not_available`, `upcoming`. Store as a `status` field on the Session document.
- **Rationale**: FR-008 requires distinguishing "no data yet" from "no results to show." The four-state model maps directly to frontend rendering logic (show results, show "in progress" indicator, show "not yet available" message, or show schedule).
- **Alternatives considered**:
  - Binary available/unavailable: Rejected — cannot distinguish in-progress from upcoming.
  - Derive status on every API call: Rejected — better to compute during ingestion to keep the API layer simple.

## Decision 5: Extending the existing poller vs. separate poller

- **Decision**: Extend the existing `OpenF1Poller` to also poll sessions and session results as a second phase within the same 5-minute poll cycle. After upserting meetings, iterate over meetings to fetch sessions, then for each session fetch results and drivers.
- **Rationale**: The spec assumption says "session ingestion piggybacks on the existing meeting poll cycle." A single poller simplifies lifecycle management and avoids duplicate HTTP client/logging configuration. Rate limiting (3 req/s free tier) is manageable: 24 meetings × 1 sessions call + up to 7 sessions × 2 calls (results + drivers) = ~24 + 14 = 38 calls per cycle, well within limits when paced at 3/s (~13 seconds).
- **Alternatives considered**:
  - Separate SessionPoller: Rejected — adds lifecycle complexity with no clear benefit; the spec explicitly says piggyback on existing poll.
  - Event-driven ingestion (on meeting change): Rejected — over-engineered for a polling architecture.

## Decision 6: API endpoint design for round detail

- **Decision**: Single endpoint `GET /api/v1/rounds/{round}?year=2026` returning a complete round detail response including meeting metadata and all session results (race, qualifying, practice, sprint). Each session is a typed entry in a `sessions` array.
- **Rationale**: The round detail page needs all session data in a single load. One API call is simpler for the frontend and reduces round-trip latency. The response includes `data_as_of_utc` per session for granular freshness (FR-007). A unified endpoint matches the UX pattern: one page = one fetch.
- **Alternatives considered**:
  - Separate endpoints per session type (FR-003, FR-004, FR-005 each): Rejected — would require 3+ API calls to load one page; the spec's functional requirements define data shape, not necessarily endpoint cardinality.
  - GraphQL: Rejected — adds unnecessary dependency; the query pattern is fixed and simple.

## Decision 7: Frontend routing strategy

- **Decision**: Introduce `react-router-dom` for client-side routing. Routes: `/` for calendar (default), `/standings` for standings, `/rounds/{round}` for round detail. The existing state-based page switching (`useState<Page>`) is replaced with router navigation.
- **Rationale**: FR-009 requires navigating from calendar to round detail and FR-014 requires navigating back. Direct URL navigation (FR, acceptance scenario 3: "navigates directly to a round detail URL via bookmark") requires real URL routing, not just state toggling. `react-router-dom` is the standard React routing solution.
- **Dependency justification**: `react-router-dom` is a widely-used, well-maintained library (MIT license, React team-adjacent). It is the de facto standard for React SPA routing and provides URL-based navigation, history management, and deep-linking — all required by the spec. No simpler alternative provides bookmark/URL-based navigation.
- **Alternatives considered**:
  - Continue with `useState<Page>`: Rejected — cannot support URL-based navigation or bookmarkable round detail pages.
  - Custom hash-based routing: Rejected — re-invents what react-router provides, higher maintenance cost.

## Decision 8: Cosmos DB partitioning for session data

- **Decision**: Use `season` as the partition key for both `Session` and `SessionResult` documents, consistent with existing `RaceMeeting` and standings documents.
- **Rationale**: All queries are season-scoped (get sessions for round N in season 2026). The `season` partition key keeps all data for a season co-located, enabling efficient cross-partition-free queries. Document IDs follow the pattern `{season}-{round}-{session_type}` for sessions and `{season}-{round}-{session_type}-{driver_number}` for results.
- **Alternatives considered**:
  - Partition by round: Rejected — queries fetch all sessions for a round in one season, which is always the same partition under season-based partitioning.
  - Partition by session_type: Rejected — would scatter data across partitions for a single page load.

## Decision 9: Points scoring approach

- **Decision**: Do not compute points in the backend. The spec assumption states "the backend does not need to compute points — it reads them from the source data or applies the standard table." OpenF1's `championship_drivers` endpoint provides cumulative points. For per-race points display, apply the standard 2026 FIA points table during ingestion based on finishing position and fastest lap.
- **Rationale**: Keeps the backend as a data pipeline rather than a rules engine. Points calculation is deterministic from position + fastest lap flag, so a simple lookup table suffices.
- **Alternatives considered**:
  - Compute points entirely from upstream cumulative data: Partially viable but doesn't give per-race breakdown.
  - Fetch points from a separate standings source: Already done for cumulative standings in feature 002.

## Decision 10: Sprint weekend session type handling

- **Decision**: Sprint weekends are identified by the presence of session types "Sprint Qualifying" and "Sprint" in the OpenF1 sessions response. The backend stores and serves all session types uniformly. The frontend renders whatever sessions exist for a round without requiring a separate "sprint format" flag.
- **Rationale**: FR-015 requires handling sprint formats. SC-006 requires displaying the correct session structure "without manual configuration per round." OpenF1 provides session_type metadata that distinguishes sprint sessions. The frontend simply iterates over available sessions.
- **Alternatives considered**:
  - Manual weekend-type configuration per round: Rejected — violates SC-006.
  - Hardcoded sprint round numbers: Rejected — fragile and violates data-driven principle.

## Open questions (resolved)

All NEEDS CLARIFICATION items have been resolved through OpenF1 API documentation research. No blocking unknowns remain.
