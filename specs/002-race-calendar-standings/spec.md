# Feature Specification: Race Calendar & Championship Standings

**Feature Branch**: `[002-race-calendar-standings]`  
**Created**: 2026-04-19  
**Status**: Draft  
**Input**: User description: "Feature: Race Calendar & Championship Standings"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - View Full 2026 Race Calendar (Priority: P1)

As an F1 fan, I can view all 24 rounds of the 2026 season with race name, circuit, country, and date range so I can track the complete season schedule.

**Why this priority**: This is the primary value of the feature and the foundation for all race-weekend awareness use cases.

**Independent Test**: Can be fully tested by requesting the backend race-calendar endpoint and loading the frontend race calendar page, then validating exactly 24 rounds with required fields.

**Acceptance Scenarios**:

1. **Given** 2026 calendar data is cached in Cosmos DB, **When** a user opens the calendar page, **Then** the UI shows 24 rounds with round number, race name, circuit, country, and event date range.
2. **Given** OpenF1 has updated meeting metadata, **When** the backend polling cycle runs, **Then** the updated metadata is persisted in Cosmos DB and returned by backend API responses without frontend changes.

---

### User Story 2 - Track Upcoming Race Countdown (Priority: P1)

As a fan preparing for race weekend, I can immediately identify the next upcoming race and see a live countdown so I know how long remains until race start.

**Why this priority**: Time-to-next-race awareness is a top user task and drives repeated engagement.

**Independent Test**: Can be fully tested by stubbing current time before and after known race start times and confirming one race is highlighted and countdown behavior is correct.

**Acceptance Scenarios**:

1. **Given** at least one future race exists in the season, **When** the user views the calendar, **Then** exactly one race is visually highlighted as "Next Race" and includes a countdown timer to scheduled race start.
2. **Given** the current time has passed the highlighted race start, **When** countdown refresh logic runs, **Then** the next chronologically valid race becomes highlighted and countdown recalculates.

---

### User Story 3 - Understand Cancelled Events & Current Standings (Priority: P2)

As a user following championship progression, I can identify cancelled rounds and view up-to-date driver and constructor standings from backend-provided data.

**Why this priority**: Standings and cancellation status provide context for how the season unfolds, but can ship after the core calendar view.

**Independent Test**: Can be fully tested by calling standings endpoints and calendar endpoints, verifying Bahrain (R4) and Saudi Arabia (R5) are marked cancelled and excluded from next-race countdown eligibility.

**Acceptance Scenarios**:

1. **Given** Bahrain R4 and Saudi Arabia R5 are configured as cancelled races, **When** calendar data is returned, **Then** both rounds have a cancelled indicator and cancellation reason/status metadata in the backend response.
2. **Given** championship data is available from Hyprace and OpenF1-derived identity data, **When** the user opens standings, **Then** drivers table shows position, driver name, team, points, wins and constructors table shows position, team, points.
3. **Given** a cancelled round date is in the future, **When** upcoming-race logic evaluates candidates, **Then** cancelled rounds are never selected as the next highlighted race.

---

### Edge Cases

- OpenF1 `/v1/meetings?year=2026` temporarily fails or times out: backend serves last successful Cosmos snapshot with stale-data timestamp.
- Fewer than 24 meetings are returned upstream: backend records quality warning and returns available meetings with `completeness_status` indicating partial data.
- Two races have identical start timestamps due to data error: backend deterministic tie-break uses lower round number.
- Countdown reaches zero while user is on page: frontend transitions to next valid (non-cancelled) upcoming race without full page reload.
- Hyprace standings are unavailable but OpenF1 driver identities are available: backend returns standings endpoint with explicit partial-source status and non-200 only when no standings rows can be produced.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Backend MUST ingest 2026 race meetings from OpenF1 `GET /v1/meetings?year=2026` on a 5-minute polling interval and persist normalized records in Cosmos DB serverless.
- **FR-002**: Backend race calendar API MUST return all 24 expected rounds for 2026 with fields: `round`, `race_name`, `circuit_name`, `country_name`, `start_datetime_utc`, `end_datetime_utc`, and `status`.
- **FR-003**: Backend MUST mark Bahrain round 4 and Saudi Arabia round 5 as `cancelled`, include a visual-indicator hint field (for UI rendering), and include cancellation metadata (`is_cancelled=true`, `cancelled_label`).
- **FR-004**: Next-upcoming-race selection MUST exclude cancelled rounds and any rounds with start time earlier than current time; backend MUST expose the selected `next_round` and `countdown_target_utc`.
- **FR-005**: Frontend MUST consume only backend endpoints for calendar and standings and MUST NOT call OpenF1 or Hyprace directly from browser code.
- **FR-006**: Backend MUST provide a drivers standings endpoint containing rows with `position`, `driver_name`, `team_name`, `points`, and `wins`, using Hyprace standings data plus OpenF1 driver metadata where needed.
- **FR-007**: Backend MUST provide a constructors standings endpoint containing rows with `position`, `team_name`, and `points`, sourced from Hyprace standings data.
- **FR-008**: Backend MUST cache raw upstream payload fingerprints and last-refresh timestamps in Cosmos DB and return `data_as_of_utc` in API responses for freshness transparency.
- **FR-009**: Backend MUST expose health/metrics signals for poll success rate, upstream latency, cache age, and error counts, and MUST log structured JSON events for poll cycles and API responses.
- **FR-010**: System MUST run as two containers on AKS (Go/Chi backend and React frontend), with backend-to-Cosmos and backend-to-upstream network paths only.
- **FR-011**: Backend outbound egress MUST be restricted to required upstream APIs (OpenF1, Hyprace) through configured firewall allow-list controls.
- **FR-012**: Secrets and API credentials for Hyprace (if required) MUST be retrieved via Managed Identity from Azure Key Vault and MUST NOT be stored in source, image layers, or plaintext config.
- **FR-013**: Dependency additions MUST be minimized; any new package for polling, HTTP clients, or time handling MUST include brief justification and owner in implementation artifacts.
- **FR-014**: Backend API contract MUST include explicit `status` enum values (`scheduled`, `cancelled`, `completed`, `unknown`) and frontend MUST render a distinct cancelled visual treatment for cancelled rounds.

### Constitution Alignment *(mandatory)*

- **CA-001**: Feature runs on Go backend + React frontend + Cosmos DB on AKS.
- **CA-002**: Three-tier boundary enforced: frontend calls backend API only; backend owns third-party integration.
- **CA-003**: OpenF1-derived data is persisted in Cosmos DB and served from cache-first behavior with explicit freshness metadata.
- **CA-004**: Hyprace secrets use Key Vault + Managed Identity; no static secrets allowed.
- **CA-005**: AKS deployment and ingress controls enforce HTTPS and network egress restrictions.
- **CA-006**: Operational requirements include structured JSON logging and observable polling/cache metrics.
- **CA-007**: Dependency discipline maintained with explicit justification for any added library.

### Key Entities *(include if feature involves data)*

- **RaceMeeting**: Represents one F1 round with round number, venue, country, event date range, canonical status, cancellation metadata, and source refresh timestamps.
- **UpcomingRaceSnapshot**: Represents computed next race context with selected round, countdown target timestamp, generation time, and exclusion reasons for skipped rounds.
- **DriverStandingRow**: Represents one driver ranking row with position, driver display name, team, points, wins, and source freshness metadata.
- **ConstructorStandingRow**: Represents one constructor ranking row with position, team, points, and source freshness metadata.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Calendar API returns exactly 24 unique rounds for year 2026 in 99.9% of successful requests over a rolling 7-day window.
- **SC-002**: Bahrain R4 and Saudi Arabia R5 are marked `cancelled` in 100% of calendar responses and are never selected as `next_round`.
- **SC-003**: Backend polling job executes every 5 minutes with at least 95% successful poll cycles per day.
- **SC-004**: `data_as_of_utc` freshness for calendar and standings is no older than 10 minutes for at least 95% of requests during normal upstream availability.
- **SC-005**: P95 backend response time is under 300 ms for calendar and under 350 ms for standings, measured at API gateway over a 24-hour sample.
- **SC-006**: Frontend network inspection shows 0 direct browser requests to OpenF1 or Hyprace in production mode.
- **SC-007**: Standings endpoints return complete required columns (drivers: 5 fields, constructors: 3 fields) in 99% of successful requests.

## Assumptions

- OpenF1 `meetings` data for 2026 includes enough metadata to map race name, circuit, country, and start/end datetimes.
- Hyprace provides standings data for drivers and constructors compatible with required table columns.
- Cancellation status for Bahrain R4 and Saudi Arabia R5 is a product-defined override and remains in effect unless explicitly revised.
- Countdown target uses UTC race start time from normalized backend meeting data.
- Existing AKS, Cosmos DB serverless account, Key Vault, and Azure Monitor foundations are available to this feature without re-platforming.
