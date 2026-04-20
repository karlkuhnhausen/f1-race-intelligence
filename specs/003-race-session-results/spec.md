# Feature Specification: Race Results & Session Data

**Feature Branch**: `003-race-session-results`  
**Created**: 2026-04-19  
**Status**: Draft  
**Input**: User description: "Add race results and session data to the F1 Race Intelligence Dashboard"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - View Race Results for a Completed Round (Priority: P1)

As an F1 fan, I can click on a completed race in the calendar and see the race finishing order — including positions, driver names, teams, time gaps, points scored, and fastest lap indicator — so I can review how each grand prix played out.

**Why this priority**: Race results are the core value of this feature and the primary reason users navigate to a round detail page. Without race results, session data has limited standalone value.

**Independent Test**: Can be fully tested by navigating to a completed round detail page and verifying that the race results table displays all finishers with correct positions, gaps, points, and fastest lap marker.

**Acceptance Scenarios**:

1. **Given** a round has completed race session data available, **When** a user clicks that round in the calendar table, **Then** the user navigates to a detail page showing race results with position, driver name, team, finishing gap, points awarded, and fastest lap indicator for each classified finisher.
2. **Given** a round's race session data includes retirements and non-classified entries, **When** the detail page loads, **Then** retired drivers appear at the bottom of the results table with their retirement status (e.g., "DNF", "DNS") instead of a time gap.
3. **Given** race results data has been ingested from OpenF1, **When** the backend race results endpoint is called for that round, **Then** the response includes `data_as_of_utc` freshness metadata consistent with the last successful poll.

---

### User Story 2 - View Qualifying Results for a Round (Priority: P2)

As an F1 fan, I can view qualifying results for a round — showing qualifying positions, driver names, teams, and best lap times across Q1/Q2/Q3 — so I can understand how the grid was set.

**Why this priority**: Qualifying determines the starting grid and is the second most important session after the race. It provides essential pre-race context.

**Independent Test**: Can be fully tested by navigating to a round with qualifying data and verifying the qualifying results table shows positions with best times per qualifying segment.

**Acceptance Scenarios**:

1. **Given** a round has completed qualifying session data, **When** a user views the round detail page, **Then** a qualifying results section shows each driver's grid position, name, team, and best lap time in each qualifying segment they participated in (Q1, Q2, Q3).
2. **Given** a driver was eliminated in Q1, **When** qualifying results are displayed, **Then** that driver shows a Q1 time but no Q2 or Q3 times, and their final qualifying position is shown correctly.
3. **Given** qualifying data is not yet available for a future round, **When** the user views that round's detail page, **Then** the qualifying section displays a clear "Not yet available" message rather than an empty or broken table.

---

### User Story 3 - View Practice Session Results (Priority: P3)

As an F1 fan, I can view practice session times — showing driver names, teams, best lap times, number of laps completed, and time gap to the fastest driver — so I can track pace trends across the weekend.

**Why this priority**: Practice data enriches the weekend picture but is less critical than race or qualifying results. Many fans skip practice details entirely.

**Independent Test**: Can be fully tested by navigating to a round with practice data and verifying practice session tables show driver times and lap counts.

**Acceptance Scenarios**:

1. **Given** a round has one or more completed practice sessions (FP1, FP2, FP3), **When** the user views the round detail page, **Then** each practice session appears as a separate section with drivers listed by fastest lap time, showing name, team, best time, gap to fastest, and laps completed.
2. **Given** a round uses a sprint weekend format with only one practice session, **When** the detail page loads, **Then** only the available practice session is shown and no empty placeholder sections appear for missing sessions.

---

### User Story 4 - Navigate Between Calendar and Round Detail (Priority: P1)

As a user browsing the calendar, I can click any round to see its details and easily return to the calendar, so I can explore multiple rounds without losing my place.

**Why this priority**: Navigation is a foundational user experience requirement that enables all other stories. Without it, session data is inaccessible.

**Independent Test**: Can be fully tested by clicking a round in the calendar, verifying navigation to the detail page, then clicking the back/calendar link and verifying return to the calendar view.

**Acceptance Scenarios**:

1. **Given** the user is on the calendar page, **When** they click on any round row, **Then** they navigate to a round detail page identified by round number.
2. **Given** the user is on a round detail page, **When** they click the navigation element to return to the calendar, **Then** they return to the calendar view with their previous scroll position or state preserved.
3. **Given** the user navigates directly to a round detail URL (e.g., via bookmark or shared link), **When** the page loads, **Then** the detail page renders correctly with all available session data for that round.

---

### Edge Cases

- A session is in progress when the user loads the detail page: the backend serves the latest ingested snapshot and the frontend shows a "Session in progress — data may be partial" indicator with `data_as_of_utc` timestamp.
- OpenF1 returns no session data for a round (e.g., cancelled round): the detail page displays the round header information with a clear message that no session data is available.
- A round has some sessions completed but others not yet started: each session section independently shows results or a "not yet available" state.
- OpenF1 session data arrives out of order or with corrections: the backend upserts on the unique session+driver key, so later ingestion overwrites earlier data and the most recent snapshot is always served.
- A driver participates as a reserve/substitute for a single session: the driver appears in that session's results with whatever team association the source data provides.
- The user navigates to a round number that does not exist in the season: the detail page returns a clear "Round not found" message with a link back to the calendar.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Backend MUST ingest session data from OpenF1 `GET /v1/sessions?meeting_key={key}` for each known meeting, polling on the existing 5-minute interval alongside meeting ingestion.
- **FR-002**: Backend MUST ingest session results from OpenF1 position and lap endpoints for each session, storing per-driver results including position, driver name, team, lap times, gap, and status.
- **FR-003**: Backend MUST provide a race results endpoint that returns classified finishers and non-classified entries for a given round, including position, driver name, team, time/gap to leader, points scored, fastest lap flag, and finishing status.
- **FR-004**: Backend MUST provide a qualifying results endpoint that returns qualifying positions with driver name, team, and best lap times for each qualifying segment (Q1, Q2, Q3) for a given round.
- **FR-005**: Backend MUST provide a practice results endpoint that returns driver times for each practice session of a given round, including driver name, team, best lap time, gap to fastest, and laps completed.
- **FR-006**: Backend MUST persist all ingested session and result data in Cosmos DB with appropriate partitioning alongside existing meeting data.
- **FR-007**: Backend session results endpoints MUST return `data_as_of_utc` freshness metadata, consistent with the existing pattern established in the calendar endpoints.
- **FR-008**: Backend MUST handle sessions that have not yet occurred by returning an explicit "not available" status rather than empty results, so the frontend can distinguish between "no data yet" and "no results to show."
- **FR-009**: Frontend MUST add a round detail page accessible via client-side routing from the calendar table, displaying all available session data for the selected round.
- **FR-010**: Frontend round detail page MUST show race results, qualifying results, and practice results in clearly separated sections, rendering only sections for which data is available.
- **FR-011**: Frontend MUST indicate partial or in-progress session data when the backend signals that a session is ongoing.
- **FR-012**: Frontend MUST consume only backend endpoints and MUST NOT call OpenF1 directly from browser code, consistent with the existing architecture boundary.
- **FR-013**: Calendar table rows MUST be clickable, navigating the user to the corresponding round detail page.
- **FR-014**: Round detail page MUST include a navigation element to return to the calendar view.
- **FR-015**: Backend MUST handle sprint weekend session formats (sprint qualifying, sprint race) as distinct session types alongside standard practice, qualifying, and race sessions.
- **FR-016**: Backend race results MUST include explicit status values for non-standard finishes: "Finished", "DNF" (did not finish), "DNS" (did not start), "DSQ" (disqualified).

### Constitution Alignment *(mandatory)*

- **CA-001**: Feature runs on existing Go backend + React frontend + Cosmos DB on AKS infrastructure deployed in feature 002.
- **CA-002**: Three-tier boundary enforced: frontend calls backend API only; backend owns all OpenF1 integration.
- **CA-003**: OpenF1-derived session and result data is persisted in Cosmos DB and served from cache-first behavior with explicit freshness metadata.
- **CA-004**: No new secrets are required for this feature; OpenF1 is free-tier with no API key. Existing Key Vault + Managed Identity patterns are preserved.
- **CA-005**: Existing AKS HTTPS ingress and network egress controls apply. No new external API endpoints are introduced beyond OpenF1 (already allowed).
- **CA-006**: Deployment artifacts extend existing Helm charts; no new Kubernetes resource types required.
- **CA-007**: Feature delivery follows existing GitHub Actions pipeline: lint → test → build → push → deploy.
- **CA-008**: Session ingestion and result API operations include structured JSON logging to Azure Monitor, extending existing log patterns.
- **CA-009**: New dependencies (if any) require explicit justification in implementation artifacts.
- **CA-010**: Any work not traceable to this spec is out of scope until approved.

### Key Entities

- **Session**: Represents one session within a race weekend (FP1, FP2, FP3, Sprint Qualifying, Sprint, Qualifying, Race), identified by meeting key and session type, with session status and timing metadata.
- **RaceResult**: Represents one driver's finishing record in a race session — position, driver name, team, time/gap, points scored, fastest lap indicator, and finishing status (Finished/DNF/DNS/DSQ).
- **QualifyingResult**: Represents one driver's qualifying record — grid position, driver name, team, and best lap time per segment (Q1, Q2, Q3), with elimination round indicated.
- **PracticeResult**: Represents one driver's practice session record — driver name, team, best lap time, gap to session fastest, and laps completed.
- **SessionAvailability**: Represents whether a given session's data is available, not yet started, in progress, or complete — used to drive frontend rendering decisions.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can navigate from the calendar to any round's detail page and back in under 2 seconds total navigation time.
- **SC-002**: Race results for completed rounds display all classified and non-classified finishers with 100% positional accuracy relative to the ingested source data.
- **SC-003**: Session data freshness (`data_as_of_utc`) is no older than 10 minutes for at least 95% of detail page requests during normal upstream availability, consistent with the calendar freshness target.
- **SC-004**: Round detail pages correctly show "not yet available" messaging for sessions that have not occurred, with zero instances of empty or broken tables displayed to users.
- **SC-005**: All session types for a completed standard race weekend (FP1, FP2, FP3, Qualifying, Race) are available on the detail page within 15 minutes of the final session ending.
- **SC-006**: Sprint weekend formats display the correct session structure (Practice, Sprint Qualifying, Sprint, Qualifying, Race) without manual configuration per round.
- **SC-007**: Frontend network inspection shows zero direct browser requests to OpenF1 on the round detail page in production mode.

## Assumptions

- OpenF1 provides session-level and result-level data (sessions, positions, laps) sufficient to construct practice times, qualifying segment results, and race finishing orders.
- The existing 5-minute polling infrastructure can be extended to poll session data without requiring a separate poller — session ingestion piggybacks on the existing meeting poll cycle.
- Sprint weekend formats are identifiable via session type metadata from OpenF1 (e.g., distinct session names for "Sprint", "Sprint Qualifying").
- Points scoring follows the standard 2026 FIA points system; the backend does not need to compute points — it reads them from the source data or applies the standard table.
- Fastest lap attribution is available in the source data or can be derived from lap time data for the race session.
- The existing Cosmos DB serverless account has sufficient throughput for the additional session data storage and queries without tier changes.
- No authentication or authorization is required for accessing round detail pages — the dashboard remains publicly accessible consistent with feature 002.
- Mobile-responsive layout for the detail page follows the same approach as the existing calendar page (responsive table/card patterns).
