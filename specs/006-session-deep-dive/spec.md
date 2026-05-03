# Feature Specification: Session Deep Dive Page

**Feature Branch**: `006-session-deep-dive`  
**Created**: 2026-05-02  
**Status**: Draft  
**Input**: User description: "A dedicated post-session analysis page accessible from the round detail page providing rich visualizations of race/sprint session data using historical OpenF1 data."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Position Battle Chart (Priority: P1)

A fan visits the round detail page for a completed race and clicks "View Analysis." They navigate to a dedicated analysis page showing a lap-by-lap position chart with all 20 drivers as individual colored lines. The user can immediately see the race narrative: who led, where position swaps happened, where drivers retired, and how the field evolved over the full race distance.

**Why this priority**: The position chart is the "hero visualization" — it tells the race story at a glance and is the single most valuable chart for post-session analysis. Without it, the page has no reason to exist.

**Independent Test**: Can be fully tested by navigating to a completed race's analysis page and verifying a multi-line chart renders with correct driver positions per lap. Delivers standalone value as a race story overview.

**Acceptance Scenarios**:

1. **Given** a finalized 2026 race session with position data in the database, **When** a user clicks "View Analysis" on the round detail page, **Then** they see a line chart with one line per driver showing position (Y-axis, inverted 1 at top) vs. lap number (X-axis)
2. **Given** a race where a driver retired mid-race (DNF), **When** the analysis page loads, **Then** that driver's line ends at their retirement lap without extending further
3. **Given** a race with a red flag restart, **When** the analysis page loads, **Then** the position chart shows a visual gap or annotation at the red-flag lap
4. **Given** the analysis page is viewed on a mobile device (≤768px), **When** the page renders, **Then** the chart displays full-width and is scrollable/zoomable for readability
5. **Given** a session that has not yet been finalized, **When** a user navigates to the analysis URL, **Then** they see an "Analysis not yet available" message with context about when data will be ready

---

### User Story 2 - Backfill Existing 2026 Sessions (Priority: P1)

A system operator runs a CLI command to backfill analysis data for all existing finalized 2026 race and sprint sessions. This ensures that on deployment day, users can immediately access analysis for past races without waiting for data to trickle in.

**Why this priority**: Without backfill, the feature launches with empty pages for all historical sessions. This is a deployment-day blocker — the feature must have data on day one.

**Independent Test**: Can be tested by running the backfill CLI against a known set of finalized sessions and verifying that position, interval, stint, pit, and overtake data are persisted in the database for each session.

**Acceptance Scenarios**:

1. **Given** multiple finalized 2026 race/sprint sessions exist, **When** the operator runs the backfill command, **Then** all five data types (position, intervals, stints, pits, overtakes) are fetched and stored for each session
2. **Given** the backfill is re-run for sessions that already have data, **When** execution completes, **Then** existing data is not duplicated (idempotent operation)
3. **Given** one session fails to fetch from the external source, **When** the backfill runs, **Then** it logs the error, skips that session, and continues with remaining sessions
4. **Given** rate limiting constraints on the data source, **When** the backfill runs, **Then** it applies appropriate delays between requests to avoid being throttled

---

### User Story 3 - Tire Strategy Swimlane (Priority: P2)

A user viewing the analysis page sees a horizontal swimlane chart showing each driver's tire compound choices across their race laps. Each stint is displayed as a colored block (indicating compound type) spanning the lap range of that stint, making strategy decisions immediately visible.

**Why this priority**: Tire strategy is a fundamental element of race analysis — it explains many position changes visible in the position chart. Showing compound choices side-by-side for all drivers reveals strategic patterns (one-stop vs. two-stop, offset strategies, undercuts).

**Independent Test**: Can be tested by loading a session analysis page and verifying that each driver's row shows correctly-colored blocks corresponding to their stint data, with accurate lap ranges.

**Acceptance Scenarios**:

1. **Given** a finalized race with stint data, **When** the analysis page loads, **Then** a swimlane chart shows one row per driver with colored blocks representing each stint's compound (Soft, Medium, Hard, Intermediate, Wet)
2. **Given** a driver who made two pit stops, **When** viewing their swimlane row, **Then** three distinct compound blocks are shown with correct start/end lap boundaries
3. **Given** the tire strategy chart and position chart are both visible, **When** comparing them, **Then** a user can correlate a driver's tire change with their position changes at that lap

---

### User Story 4 - Pit Stop Timeline (Priority: P2)

A user sees a timeline visualization showing when each driver pitted and how long each stop took. Pit windows become obvious, and undercut/overcut strategies are immediately visible by comparing stop timing between drivers.

**Why this priority**: Pit stops are pivotal race moments. Showing their timing and duration across all drivers reveals strategy execution quality and explains net position changes that aren't obvious from the position chart alone.

**Independent Test**: Can be tested by loading a session with pit data and verifying that each pit stop is displayed with correct lap number, driver, and duration.

**Acceptance Scenarios**:

1. **Given** a finalized race with pit stop data, **When** the analysis page loads, **Then** a timeline shows pit stop markers for each driver at their respective lap numbers with stop duration visible
2. **Given** a driver who made a slow pit stop (>5 seconds), **When** viewing the timeline, **Then** the long stop is visually distinguishable from normal-duration stops
3. **Given** two drivers who pitted on consecutive laps, **When** viewing the timeline, **Then** the timing difference (undercut window) is clearly visible

---

### User Story 5 - Gap-to-Leader Progression (Priority: P2)

A user can see how the time gap between selected drivers and the race leader evolved over the course of the race. This reveals pace differentials — who was catching the leader, who was falling back, and when Virtual Safety Cars or Safety Cars compressed the field.

**Why this priority**: Gap data adds the pace dimension that pure position data cannot show. A driver holding P3 throughout may have been catching P1 or falling away — gap progression reveals which.

**Independent Test**: Can be tested by loading a session with interval data and verifying a line chart shows gap-to-leader values over laps for all drivers.

**Acceptance Scenarios**:

1. **Given** a finalized race with interval data, **When** the analysis page loads, **Then** a chart shows gap-to-leader (seconds) on the Y-axis and lap number on the X-axis, with one line per driver
2. **Given** a safety car period compressed gaps, **When** viewing the chart, **Then** gap values visibly converge toward zero during that window
3. **Given** a driver who lapped other drivers, **When** viewing the chart, **Then** lapped drivers' gap values continue increasing naturally without wrapping or resetting

---

### User Story 6 - Overtake Annotations (Priority: P3)

A user sees markers on the position chart indicating where overtakes occurred — showing which driver passed which other driver at which lap. This enriches the position chart narrative by highlighting the key on-track battles.

**Why this priority**: Overtake data enriches the position chart but is supplementary. The position chart already shows position changes; overtake annotations add "who vs. whom" context. Also, external source overtake data may be incomplete for some sessions.

**Independent Test**: Can be tested by loading a session with overtake data and verifying markers/annotations appear on the position chart at the correct lap positions.

**Acceptance Scenarios**:

1. **Given** a finalized race with overtake data, **When** the analysis page loads, **Then** the position chart shows annotation markers at laps where overtakes occurred
2. **Given** a session where overtake data is incomplete or unavailable, **When** the analysis page loads, **Then** the position chart renders normally without annotations and no error is shown (graceful degradation)
3. **Given** an overtake annotation is visible, **When** a user interacts with it (hover/tap), **Then** they see details: overtaking driver name, overtaken driver name, and new position

---

### Edge Cases

- **Red flag restart**: Position chart shows a visible gap or annotation at the red-flag lap; positions resume from restart grid
- **DNS (Did Not Start)**: Driver does not appear in charts at all — they have zero lap data
- **DNF (Did Not Finish)**: Driver's lines end at their retirement lap; pit/tire data only shows up to that point
- **Sprint race (~20 laps)**: Same layout and chart types with fewer data points; no special handling needed
- **Incomplete overtake data**: Overtake annotations section gracefully degrades — chart renders without annotations, no error message
- **Session not yet finalized**: Full page shows "Analysis not yet available" state with explanation that data appears approximately 2 hours after session end
- **No interval data available**: Gap-to-leader chart shows "Data not available" placeholder while other charts render normally
- **Driver with no pit stops** (rare edge case in sprint): Tire swimlane shows a single block spanning full race distance

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide a dedicated analysis page at `/rounds/:round/sessions/:sessionType/analysis` for Race and Sprint sessions only
- **FR-002**: System MUST display a "View Analysis" button on the round detail page for each completed Race and Sprint session, linking to the analysis page
- **FR-003**: System MUST NOT show "View Analysis" buttons for Qualifying or Practice sessions
- **FR-004**: System MUST render a lap-by-lap position chart showing all drivers as individually colored lines with position (1-20, inverted) on Y-axis and lap number on X-axis
- **FR-005**: System MUST render a gap-to-leader progression chart showing gap (seconds) vs. lap number for all drivers
- **FR-006**: System MUST render a tire strategy swimlane showing compound usage (Soft, Medium, Hard, Intermediate, Wet) per driver across laps as colored blocks
- **FR-007**: System MUST render a pit stop timeline showing when each driver pitted with stop duration indicated
- **FR-008**: System MUST render overtake annotations on the position chart showing who passed whom at which lap
- **FR-009**: System MUST use team colors for driver lines/markers across all charts (consistent color coding)
- **FR-010**: System MUST aggregate raw position data server-side to 1 data point per driver per lap before serving to the frontend
- **FR-011**: System MUST fetch and cache position, interval, stint, pit, and overtake data from the external source after session finalization (minimum 2-hour buffer)
- **FR-012**: System MUST cache analysis data indefinitely once fetched (post-session data is immutable)
- **FR-013**: System MUST provide a single combined API endpoint returning all analysis data for a given session
- **FR-014**: System MUST display an "Analysis not yet available" state when data has not been ingested for a session
- **FR-015**: System MUST gracefully degrade when individual data types are unavailable (e.g., no overtake data should not prevent position chart from rendering)
- **FR-016**: System MUST provide a one-shot backfill mechanism to populate data for all existing finalized 2026 Race and Sprint sessions
- **FR-017**: The backfill mechanism MUST be idempotent — re-running does not duplicate data
- **FR-018**: System MUST stack charts vertically on mobile viewports (≤768px) and allow full-width display on desktop
- **FR-019**: System MUST provide back-navigation from the analysis page to the round detail page
- **FR-020**: System MUST apply appropriate rate limiting when fetching data from the external source (delays between requests to avoid throttling)

### Constitution Alignment *(mandatory)*

- **CA-001**: Feature runs on Go backend (session analysis data ingestion, aggregation, API) + React frontend (chart rendering) + Cosmos DB (data persistence) on AKS. No exceptions.
- **CA-002**: Frontend MUST call only the backend analysis API endpoint. Frontend MUST NOT call OpenF1 or any third-party API directly.
- **CA-003**: All OpenF1 analysis data (position, intervals, stints, pits, overtakes) MUST be cached in Cosmos DB before being served to the frontend. No pass-through to OpenF1.
- **CA-004**: No new secrets required for this feature — OpenF1 is free-tier with no API key. Existing Key Vault + Managed Identity patterns apply for Cosmos DB access.
- **CA-005**: Analysis API endpoint served via existing HTTPS ingress. No new egress rules needed (OpenF1 already allowlisted).
- **CA-006**: No new Helm chart resources required — backend and frontend deployments already exist. New API routes and frontend pages are code changes within existing deployments.
- **CA-007**: Feature delivery follows existing CI/CD stages: lint → test → build → push → deploy. Backfill CLI runs as a one-shot job post-deploy.
- **CA-008**: Analysis data ingestion MUST log structured JSON (session key, data type, row count, duration) to Azure Monitor.
- **CA-009**: New frontend charting dependency (recharts) MUST include explicit justification documenting why it's needed and why alternatives were not chosen.
- **CA-010**: All implementation work must trace to this specification. Out-of-scope enhancements require separate approval.

### Key Entities

- **SessionPosition**: Per-driver position at each lap for a given session — attributes: session key, driver number, lap number, position
- **SessionInterval**: Per-driver gap-to-leader and interval at each sample point — attributes: session key, driver number, lap number, gap to leader (seconds), interval (seconds)
- **SessionStint**: Tire compound usage per driver per stint — attributes: session key, driver number, stint number, compound, lap start, lap end, tyre age at start
- **SessionPit**: Pit stop event — attributes: session key, driver number, lap number, pit lane duration, stop duration
- **SessionOvertake**: Overtake event — attributes: session key, overtaking driver number, overtaken driver number, lap number, resulting position

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can navigate from a round detail page to a session analysis page in one click and see all visualizations rendered within 3 seconds of page load
- **SC-002**: All finalized 2026 Race and Sprint sessions have analysis data available on feature deployment day (backfill complete)
- **SC-003**: Position chart accurately reflects the final classification — P1 through P20 at the final lap matches official results for every session
- **SC-004**: Analysis page is usable on mobile devices (≤768px): all charts are visible, scrollable, and legible without horizontal overflow
- **SC-005**: When one data type is unavailable (e.g., overtakes), the remaining charts render without error — zero full-page failures due to partial data
- **SC-006**: System handles the largest expected session payload (5000 raw position rows aggregated to ~1200 points) without frontend performance degradation (no jank, smooth scrolling)
- **SC-007**: Backfill CLI completes processing of all existing 2026 sessions within 30 minutes (respecting rate limits)

## Assumptions

- OpenF1 free-tier historical endpoints (/position, /intervals, /stints, /pit, /overtakes) remain available and structurally stable for 2026 session data
- Position and interval data is immutable once a session is finalized — caching indefinitely is safe
- The existing driver data in Cosmos DB (from Feature 003) includes team color information usable for chart coloring
- The existing round detail page (Feature 003) provides a suitable location for the "View Analysis" button
- The existing session poller (Feature 005 pattern) provides the mechanism to detect session finalization and trigger data ingestion with a 2-hour buffer
- Sprint races produce the same data structure from OpenF1 endpoints as full races, just with fewer laps
- A single new charting library (recharts) is sufficient for all five visualization types; no additional visualization dependencies are needed
- The 2-hour post-session buffer is sufficient for OpenF1 to have complete data available for all five endpoints
- Desktop users have viewports ≥1024px; tablet users (769-1023px) receive a layout similar to desktop
