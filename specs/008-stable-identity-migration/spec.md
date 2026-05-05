# Feature Specification: Stable Identity Migration

**Feature Branch**: `008-stable-identity-migration`  
**Created**: 2026-05-05  
**Status**: Draft (retroactive — Phases 1-4 already implemented)  
**Input**: User description: "Introduce immutable meeting_key and session_key fields as stable identity anchors for session-related Cosmos DB documents, eliminating data integrity issues caused by round-number shifts after mid-season cancellations."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Correct Data After Cancellation (Priority: P1)

A user navigates to the rounds detail page for a specific race weekend after an earlier round has been cancelled mid-season. Despite the round number shifting down by one, the page displays the correct session results, lap data, and analysis for the intended race weekend.

**Why this priority**: This is the core problem the feature solves. Without it, users see incorrect or misattributed data after any cancellation event.

**Independent Test**: Navigate to a rounds detail page whose round number changed due to a prior cancellation. Verify all displayed session data (results, positions, intervals, stints, pit stops, overtakes) matches the intended race weekend, not the one previously at that round number.

**Acceptance Scenarios**:

1. **Given** Round 5 (Australia) has been cancelled and Round 6 (Japan) has shifted to Round 5, **When** a user visits the Round 5 detail page, **Then** the system displays Japan session data, not stale Australia data.
2. **Given** a round detail page is loaded for a post-cancellation round number, **When** the system resolves documents, **Then** it uses meeting_key as the primary lookup and returns correct results.
3. **Given** multiple cancellations have occurred in a season, **When** a user navigates to any affected round, **Then** all rounds display their correct race data regardless of how many positions they shifted.

---

### User Story 2 - Pre-Migration Data Remains Accessible (Priority: P2)

A user navigates to rounds or analysis pages for race weekends whose data was ingested before the migration (before meeting_key was populated on documents). The system falls back to round-based queries and still returns correct historical data.

**Why this priority**: Ensures backward compatibility so that existing data is not broken by the migration. Without this, all pre-migration documents would become inaccessible.

**Independent Test**: Query a session result document that was written before meeting_key fields were added. Verify the system falls back to the round-based query path and returns the document successfully.

**Acceptance Scenarios**:

1. **Given** a session result document exists without a meeting_key field, **When** the rounds service queries for that session, **Then** it falls back to the round-based query and returns the document.
2. **Given** an analysis document exists without meeting_key or session_key, **When** the analysis service queries for that session's data, **Then** it falls back gracefully and returns correct results.

---

### User Story 3 - Backfill Stamps Existing Documents (Priority: P3)

An operator runs the backfill CLI tool with the `--stamp-meeting-keys` flag to retroactively populate meeting_key and session_key on all existing Cosmos DB documents that were written before those fields existed. After backfill, all documents are queryable via the meeting_key path.

**Why this priority**: Eliminates the need for the fallback path over time, ensuring all documents converge to the stable-identity query path. This is an operational concern that doesn't affect users directly but improves long-term data integrity.

**Independent Test**: Run the backfill CLI against a container with documents lacking meeting_key. Verify all documents are updated with correct keys and remain queryable by both meeting_key and round-based paths.

**Acceptance Scenarios**:

1. **Given** existing session result documents lack meeting_key, **When** the operator runs `backfill --stamp-meeting-keys`, **Then** each document is updated with the correct meeting_key derived from its season and round metadata.
2. **Given** existing analysis documents lack meeting_key and session_key, **When** the backfill runs, **Then** both fields are populated correctly on all analysis documents.
3. **Given** a document already has meeting_key populated, **When** the backfill encounters it, **Then** it skips or idempotently re-stamps without corrupting data.

---

### User Story 4 - End-to-End Validation (Priority: P4)

After migration is complete (all documents stamped), the system is validated to confirm that every session query resolves via meeting_key and returns consistent results matching the OpenF1 source of truth.

**Why this priority**: Provides confidence that the migration is complete and correct before the fallback path can be considered for removal in a future release.

**Independent Test**: Run a validation pass that compares query results for every known session against OpenF1 source data, confirming no mismatches.

**Acceptance Scenarios**:

1. **Given** all documents have been stamped with meeting_key, **When** a validation script queries every session via meeting_key, **Then** all queries return non-empty results matching the expected session.
2. **Given** a round-number-shifted scenario, **When** the validator queries both the meeting_key path and round-fallback path, **Then** both return the same correct data.

---

### Edge Cases

- What happens when a meeting_key is not found in OpenF1 data (e.g., a newly added sprint weekend with no historical key)? The system should skip stamping and log a warning.
- How does the system handle a document whose round metadata is ambiguous (e.g., round 0 or testing sessions)? Testing sessions are excluded from MeetingIndex; they are filtered out.
- What happens if the backfill is interrupted midway? The operation must be idempotent — re-running stamps only unstamped documents without duplicating or corrupting data.
- What happens if two meetings share the same round number in different seasons? Season year is part of the document partition key, so round-number collisions across seasons are impossible.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST store an immutable `meeting_key` field on all session-related documents (session results and all analysis types).
- **FR-002**: System MUST store an immutable `session_key` field on all analysis documents.
- **FR-003**: System MUST populate meeting_key and session_key during data ingestion from OpenF1 for all new documents going forward.
- **FR-004**: System MUST provide query methods that resolve sessions by meeting_key as the primary lookup path.
- **FR-005**: System MUST fall back to round-based queries when meeting_key is absent on a document (pre-migration data).
- **FR-006**: System MUST provide a `MeetingIndex` that maps between round numbers and meeting_keys, excluding cancelled and testing meetings.
- **FR-007**: System MUST provide a backfill CLI mode (`--stamp-meeting-keys`) that retroactively populates meeting_key and session_key on existing documents.
- **FR-008**: Backfill MUST be idempotent — re-running it produces no duplicate writes or data corruption.
- **FR-009**: System MUST NOT change any external API contracts — the frontend remains unaffected.
- **FR-010**: System MUST log structured warnings when a meeting_key cannot be resolved for a document during backfill.

### Constitution Alignment *(mandatory)*

- **CA-001**: Feature runs entirely on Go backend + Cosmos DB on AKS. No frontend changes required. ✅
- **CA-002**: UI continues to call backend APIs only. No new external API calls from frontend. ✅
- **CA-003**: OpenF1 data (meeting_key, session_key) is cached in Cosmos DB as part of document enrichment. All queries served from Cosmos DB. ✅
- **CA-004**: No new secrets required. Existing Key Vault + Managed Identity wiring unchanged. ✅
- **CA-005**: No new ingress or egress paths. Existing HTTPS ingress and firewall egress rules apply. ✅
- **CA-006**: Backfill CLI packaged as a container image deployable via existing Helm infrastructure. ✅
- **CA-007**: CI/CD unchanged — feature delivered via standard lint → test → build → push → deploy pipeline. ✅
- **CA-008**: All backfill operations and fallback decisions logged as structured JSON. ✅
- **CA-009**: No new dependencies introduced. ✅
- **CA-010**: All implementation traces to this specification. ✅

### Key Entities

- **MeetingKey**: A numeric identifier assigned by OpenF1 to a race meeting (weekend). Immutable regardless of schedule changes. Used as the primary stable anchor for all session-related documents.
- **SessionKey**: A numeric identifier assigned by OpenF1 to an individual session within a meeting (e.g., FP1, Qualifying, Race). Immutable and unique across sessions.
- **MeetingIndex**: A bidirectional lookup structure mapping season round numbers to meeting_keys for non-cancelled, non-testing meetings sorted chronologically. Rebuilt from calendar data on each request.
- **RaceMeeting**: Enriched with meeting_key field to serve as the authoritative source for round↔meeting_key mapping.
- **SessionResult**: Enriched with meeting_key to enable stable lookups independent of round position.
- **Analysis Documents** (Position, Interval, Stint, Pit, Overtake): Enriched with both meeting_key and session_key for precise session-level stable identity.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All session detail queries return correct data for a round whose number shifted due to a prior cancellation — verified by navigating to affected rounds and confirming data matches the intended race weekend.
- **SC-002**: 100% of pre-migration documents (those without meeting_key) remain accessible via the round-based fallback path with no query failures.
- **SC-003**: After backfill completes, 100% of session-related documents in Cosmos DB have meeting_key populated.
- **SC-004**: After backfill completes, 100% of analysis documents have both meeting_key and session_key populated.
- **SC-005**: The backfill operation completes without errors on the full production dataset and is re-runnable without side effects.
- **SC-006**: Zero breaking changes to external API contracts — all existing frontend functionality continues working without modification.

## Assumptions

- OpenF1 provides stable, immutable meeting_key and session_key values that never change for a given meeting/session once assigned.
- Cancelled meetings still have a meeting_key in OpenF1 historical data (they existed before cancellation).
- Testing sessions (pre-season tests) do not have meaningful meeting_keys for production use and are excluded from the MeetingIndex.
- The backfill CLI has access to the same Cosmos DB connection and calendar data as the main backend service.
- The number of existing documents requiring backfill is manageable within a single CLI run (no need for distributed batch processing).
- Round-based fallback path will remain indefinitely as a safety net; removal would be a separate future feature decision.
