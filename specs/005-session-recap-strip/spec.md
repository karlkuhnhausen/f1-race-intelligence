# Feature Specification: Session Recap Strip

**Feature Branch**: `005-session-recap-strip`  
**Created**: 2026-05-02  
**Status**: Draft  
**Input**: User description: "Add a horizontal recap strip at the top of the round detail page showing one card per session in the round, summarizing each session's outcome at a glance."

## User Scenarios & Testing *(mandatory)*

### User Story 1 — View Race Recap Card (Priority: P1)

As an F1 fan viewing a completed round's detail page, I can see a Race recap card that tells me at a glance who won, by how much, who set the fastest lap, and whether there were any safety car or red flag periods — so I can absorb the race story in seconds without reading a full results table.

**Why this priority**: The Race card is the highest-value single piece of information a fan wants. It answers "what happened?" immediately and is the primary reason a fan lands on the round detail page.

**Independent Test**: Navigate to the detail page of any 2026 round whose race session has ended. A Race card appears in the recap strip. It shows winner name with team color, gap to P2, fastest lap holder and time, total laps completed, and a count of any safety car or virtual safety car periods.

**Acceptance Scenarios**:

1. **Given** a round's race session is completed, **When** a user loads the round detail page, **Then** a Race recap card is visible at the top of the page showing: winner name, winner's team color, gap to P2 (time or "laps behind"), fastest lap holder, fastest lap time, total laps, and a race-control event summary (e.g., "2 Safety Car periods", "Red flag — Lap 14").
2. **Given** a race had no safety cars or red flags, **When** the Race recap card is shown, **Then** no race-control event line appears on the card (not rendered as empty or "0 events").
3. **Given** a race had both a red flag and a safety car period, **When** the Race recap card is rendered, **Then** the red flag event is shown (highest-priority event wins; only the top-ranked event is displayed per card).
4. **Given** a race session is not yet completed (end timestamp is in the future relative to wall clock), **When** the round detail page loads, **Then** no Race recap card appears in the strip.

---

### User Story 2 — View Qualifying Recap Card (Priority: P2)

As an F1 fan, I can see a Qualifying recap card that tells me the pole sitter, pole time, how much faster they were than P2, the Q1 and Q2 cutoff times, and whether any red flags disrupted the session.

**Why this priority**: Qualifying sets the grid and is the second most important session narrative. Pole position and the gap to P2 are standard fan reference points.

**Independent Test**: Navigate to a round with a completed qualifying session. A Qualifying card appears showing pole sitter with team color, pole time, gap to P2, Q1/Q2 cutoff times, and red flag count (if any).

**Acceptance Scenarios**:

1. **Given** a round's qualifying session is completed, **When** a user loads the round detail page, **Then** a Qualifying recap card shows: pole sitter name, team color, pole lap time, gap to P2 qualifying time, Q1 elimination cutoff time, Q2 elimination cutoff time, and red flag count if any occurred.
2. **Given** a qualifying session had no red flags, **When** the Qualifying recap card is rendered, **Then** no red flag line appears on the card.
3. **Given** a sprint weekend qualifying (Sprint Qualifying), **When** the card is rendered, **Then** it uses the same Qualifying card layout and Q1/Q2 cutoff fields are omitted if the sprint qualifying format does not have Q1/Q2 elimination segments.

---

### User Story 3 — View Practice Recap Cards (Priority: P3)

As an F1 fan, I can see a card per practice session (FP1, FP2, FP3) that shows the quickest driver, their best lap time, total laps run in that session, and any red flags.

**Why this priority**: Practice results are useful context for hardcore fans but are secondary to race and qualifying summaries.

**Independent Test**: Navigate to a round with at least one completed practice session. A card for each completed practice session appears showing session-best driver with team color, best lap time, total laps, and red flag count (if any).

**Acceptance Scenarios**:

1. **Given** a round has FP1, FP2, and FP3 completed, **When** the detail page loads, **Then** three Practice recap cards appear, one per session, each showing: session label (e.g., "Practice 1"), session-best driver name, team color, best lap time, total laps, and red flag count if any.
2. **Given** a sprint weekend with only one practice session, **When** the detail page loads, **Then** only one Practice card appears and no placeholders appear for FP2 or FP3.
3. **Given** a practice session had no red flags, **When** the Practice card is rendered, **Then** no red flag line appears on the card.

---

### User Story 4 — Strip Ordering and Mobile/Desktop Layout (Priority: P2)

As a fan on any device, the recap cards appear in chronological session order (earliest to latest) and fit naturally into the page layout on both mobile and desktop.

**Why this priority**: Correct ordering and responsive layout are prerequisites for the strip to be usable on the dominant mobile viewing context.

**Independent Test**: Load a round detail page with all sessions completed on a 375px-wide viewport (mobile) and on a 1280px-wide viewport (desktop). Cards stack vertically on mobile and display as a horizontal scrollable row on desktop.

**Acceptance Scenarios**:

1. **Given** a round has FP1, FP2, FP3, Qualifying, and Race all completed, **When** the detail page loads, **Then** cards appear in the order FP1 → FP2 → FP3 → Qualifying → Race.
2. **Given** a user is on a mobile viewport (≤768px wide), **When** the strip is rendered, **Then** cards stack vertically as a single column with full-width cards.
3. **Given** a user is on a desktop viewport (>768px wide), **When** the strip is rendered, **Then** cards appear as a horizontal row; if cards overflow the container width, the row is horizontally scrollable.

---

### User Story 5 — Backfill Existing 2026 Sessions (Priority: P1)

As the operator, I can run a one-shot backfill that populates race-control summary data for all already-finalized 2026 sessions in Cosmos, so that the recap strip works for historical rounds immediately after deployment.

**Why this priority**: Without backfill, the recap strip shows empty for all rounds that finalized before this feature shipped — which would be all rounds in the season so far. This is a deployment-day blocker.

**Independent Test**: Run the backfill tool against the production Cosmos DB. Then load any 2026 round detail page for a round that finalized before deployment. The Race (and other session) recap cards appear with correct race-control event data.

**Acceptance Scenarios**:

1. **Given** a 2026 round finalized before this feature was deployed and its session documents lack race-control summaries, **When** the backfill tool is run, **Then** each session document is updated with the race-control summary fetched from OpenF1's historical endpoint.
2. **Given** the backfill tool is run and a session's race-control data cannot be fetched (e.g., OpenF1 returns empty), **When** the tool finishes, **Then** that session is logged as a warning and the tool continues processing remaining sessions rather than aborting.
3. **Given** the backfill has already been run and a session already has race-control data, **When** the tool is run again, **Then** the existing data is not overwritten (idempotent behavior).

---

### User Story 6 — Lazy-On-Read Gap Fill (Priority: P3)

As the system, any session that finalized but is missing race-control summary data is hydrated on first read, so no manual intervention is needed for edge cases missed by the backfill or occurring during a deployment gap.

**Why this priority**: This is a resilience measure for gaps; the backfill covers the primary case. Lazy fill prevents silent data holes in production.

**Independent Test**: Manually remove the race-control summary from one session document in Cosmos. Load the round detail page for that session. The recap card appears with correct event data fetched lazily from OpenF1.

**Acceptance Scenarios**:

1. **Given** a session is completed but its Cosmos document has no race-control summary, **When** the backend serves the round detail response for that session, **Then** the backend fetches race-control data from OpenF1, persists it to Cosmos, and includes it in the response.
2. **Given** the lazy fill fetch fails (OpenF1 error or timeout), **When** the backend serves the response, **Then** the response still returns without the event data (graceful degradation — recap card renders without the event line rather than failing).

---

### Edge Cases

- What happens when a session has no classified finishers (e.g., a red-flagged race with no laps completed)? The Race card shows whatever data is available; fields with no data are omitted from the card rather than shown as blank or zero.
- What happens when two drivers share the fastest lap time? The backend returns the first occurrence by lap number in the OpenF1 data; the frontend displays it without disambiguation needed.
- What happens when a qualifying session only has Q1 (no Q2/Q3)? Q1 cutoff time is shown; Q2/Q3 cutoff fields are omitted from the card.
- What happens when the round detail page is loaded for a future round with no completed sessions? The recap strip section is not rendered at all (no empty container visible).
- What happens when OpenF1's `/v1/race_control` returns duplicate messages for the same activation? Deduplication is applied by the backend: counts represent distinct activation events, not raw message count.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The backend MUST fetch race-control data for each session from OpenF1 `/v1/race_control` at session finalization time (at least 2 hours after session end timestamp).
- **FR-002**: The backend MUST persist a race-control summary object within the existing session document in Cosmos DB, including: red flag count, safety car activation count, virtual safety car activation count, and an ordered list of notable events (type, lap number).
- **FR-003**: The backend MUST deduplicate race-control messages so that activation counts reflect distinct events, not raw message count.
- **FR-004**: The backend MUST reuse the existing Feature 003 fastest-lap finalization path unchanged for all fastest-lap data displayed on recap cards.
- **FR-005**: The backend MUST provide a one-shot backfill mechanism (CLI tool or invocable script) that iterates all finalized 2026 session documents in Cosmos, fetches missing race-control data from OpenF1, and persists it — with idempotency (skip sessions that already have the data) and fault tolerance (log and skip sessions where OpenF1 returns no data).
- **FR-006**: The backend MUST implement lazy-on-read gap filling: when serving a round detail response, if a completed session lacks race-control summary data, the backend fetches and persists it before responding.
- **FR-007**: The round detail API response MUST include, for each completed session, a recap summary payload containing all fields needed to render the appropriate session type card.
- **FR-008**: A session's completion status MUST be derived at read time from its start and end timestamps compared to wall clock — not from a stored status field.
- **FR-009**: The recap strip MUST render one card per completed session in chronological order (FP1 → FP2 → FP3 → Sprint Qualifying → Sprint → Qualifying → Race, omitting any sessions not present in the round).
- **FR-010**: Race recap cards MUST display: winner name, winner team color, gap to P2 (as time delta or laps behind), fastest lap holder, fastest lap time, total laps completed, and the highest-priority notable race-control event (if any occurred).
- **FR-011**: Qualifying recap cards MUST display: pole sitter name, team color, pole lap time, gap to P2 qualifying time, Q1 and Q2 cutoff elimination times (where applicable), and red flag count (if any).
- **FR-012**: Practice recap cards MUST display: session label (e.g., "Practice 1"), session-best driver name, team color, best lap time, total laps completed in the session, and red flag count (if any).
- **FR-013**: Sprint sessions MUST use the Race card layout for Sprint races and the Qualifying card layout for Sprint Qualifying sessions.
- **FR-014**: The notable event shown on a card MUST follow priority ranking: red flag > safety car > virtual safety car > investigation. Only the single highest-priority event is shown per card; if multiple events of the same type occurred, a count is shown (e.g., "2 Safety Car periods").
- **FR-015**: When no notable events occurred in a session, the event line MUST be omitted from the card entirely.
- **FR-016**: The recap strip MUST render as a vertical single-column stack on viewports ≤768px wide, and as a horizontal row (with overflow scrolling if needed) on viewports >768px wide.
- **FR-017**: The recap strip MUST NOT auto-refresh; it represents a static post-session state loaded once with the page.
- **FR-018**: The frontend MUST NOT call OpenF1 directly; all data MUST be served from the backend API.

### Constitution Alignment *(mandatory)*

- **CA-001**: Feature runs entirely on Go backend + React frontend + Cosmos DB on AKS. No new infrastructure components are introduced.
- **CA-002**: The frontend calls only the existing backend round detail API endpoint. No direct calls to OpenF1 or any external service from the frontend.
- **CA-003**: Race-control data is fetched from OpenF1 exactly once per session (at finalization) and persisted to Cosmos DB. The round detail API serves data from Cosmos exclusively. No pass-through to OpenF1 at request time (except lazy fill, which also persists before responding).
- **CA-004**: No new secrets are required. The backfill tool accesses Cosmos DB via the same Managed Identity + Key Vault path used by the backend service.
- **CA-005**: No HTTPS ingress or Azure Firewall egress rule changes are needed. The backend already has egress to OpenF1 and ingress from the frontend.
- **CA-006**: No new Helm chart resources or Bicep changes are expected. The backfill tool runs as a one-shot job outside the normal deployment pipeline.
- **CA-007**: Feature delivery fits the existing CI/CD pipeline order: lint → test → build → push → deploy. The backfill is a post-deploy manual step.
- **CA-008**: Backfill runs and lazy-fill operations MUST emit structured JSON logs compatible with Azure Monitor ingestion.
- **CA-009**: No new external dependencies are introduced. Race-control data comes from OpenF1 (already an established dependency). Fastest-lap reuses Feature 003 code paths.
- **CA-010**: Live telemetry, lap-by-lap charts, head-to-head comparisons, championship implications, push notifications, new secrets, new external services, and new Helm/Bicep resources are explicitly out of scope.

### Key Entities *(include if feature involves data)*

- **RaceControlSummary**: Aggregated race-control state for a single session. Contains: red flag count, safety car activation count, virtual safety car activation count, and an ordered list of notable events each with type and lap number. Stored embedded within the session document in Cosmos.
- **NotableEvent**: A single race-control activation within a session. Attributes: event type (red flag, safety car, virtual safety car, investigation), lap number. Used to populate the event line on a recap card.
- **SessionRecapPayload**: The data shape returned by the round detail API for each completed session. The payload varies by session type (Race, Qualifying, Practice) and contains the specific fields each card type needs to render.
- **BackfillRecord**: A log entry produced by the backfill tool recording session key, outcome (skipped/updated/failed), and timestamp. Persisted as structured JSON logs; not stored in Cosmos.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: After the backfill runs, every 2026 completed round detail page shows the correct recap strip with session cards — verified by spot-checking at least 3 completed rounds before the feature is considered ready.
- **SC-002**: Recap cards load within the existing round detail page load time — no additional user-perceived delay introduced by the recap strip compared to the page without it.
- **SC-003**: The recap strip renders without horizontal overflow, clipping, or overlapping elements across viewport widths from 320px to 1920px.
- **SC-004**: Any session that finalizes after deployment has its race-control summary populated automatically within the normal finalization window, with no manual intervention required.
- **SC-005**: The backfill tool completes processing of all 2026 finalized sessions without error exits, and logs a per-session outcome for each session processed.
- **SC-006**: The notable event count shown on a card matches the count of distinct activation events in the OpenF1 race-control feed for that session (not raw message count).

## Assumptions

- The round detail page and its backend API endpoint (`/rounds/:id`) already exist from Feature 003 and will be extended, not rebuilt.
- The Feature 003 fastest-lap finalization path (querying OpenF1 `/v1/laps` at session finalization) is already in production and can be reused without modification.
- OpenF1's `/v1/race_control` endpoint provides reliable historical data for all 2026 sessions, including those that finalized before this feature was deployed.
- Team color data is available in the existing domain model and is already associated with driver entries in session results.
- Session documents in Cosmos DB have a stable identifier (session key: year + round number + session type) that the backfill tool can use to query and update them.
- The backfill tool will be run once manually by the operator after deployment, as a post-deploy step — it is not part of the automated CI/CD pipeline.
- Race-control deduplication logic will treat consecutive messages of the same event type within a short time window as a single activation (standard F1 race control message pattern where deployment/retraction pairs exist).
- No changes to the Cosmos DB container schema or partition key structure are needed — race-control summary is stored as a new nested object within the existing session document.
- The 2026 F1 calendar includes standard and sprint weekend formats; the spec covers both.
- OpenF1 free-tier rate limiting (~1 req/s) must be respected by both the backfill tool and the lazy-fill path; the backfill tool MUST introduce appropriate delays between requests.
