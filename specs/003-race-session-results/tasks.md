# Tasks: Race Results & Session Data

**Input**: Design documents from `/specs/003-race-session-results/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/openapi.yaml

**Tests**: Include contract, integration, and unit tests for session ingestion, round detail API, and frontend result components.

**Organization**: Tasks are grouped by user story for independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: User story label (US1, US2, US3, US4)
- Every task lists concrete file paths

## Phase 1: Setup

**Purpose**: Create new package directories and add the one new frontend dependency.

- [ ] T001 Create backend and frontend directory scaffolding for new packages: `backend/internal/api/rounds/`, `backend/internal/domain/`, `frontend/src/features/rounds/`, `frontend/tests/rounds/`
- [ ] T002 Add `react-router-dom` dependency in `frontend/package.json` with justification (required for URL-based routing per FR-009, FR-013, FR-014; see research.md Decision 7)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Backend domain types, storage layer, and ingestion pipeline that MUST be complete before any user story work.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [ ] T003 [P] Define Session and SessionResult domain types, SessionType enum, and SessionStatus enum in `backend/internal/domain/session.go`
- [ ] T004 [P] Add SessionRepository interface with UpsertSession, UpsertSessionResult, GetSessionsByRound, GetSessionResultsByRound methods in `backend/internal/storage/repository.go`
- [ ] T005 Implement Cosmos DB session repository with upsert and query-by-round for Session and SessionResult documents in `backend/internal/storage/cosmos/session_repository.go`
- [ ] T006 [P] Implement OpenF1 session, session_result, and driver data transforms to domain/storage types (including session type slug mapping and fastest lap derivation) in `backend/internal/ingest/session_transform.go`
- [ ] T007 Extend OpenF1 poller to fetch sessions via `/v1/sessions?meeting_key={key}`, results via `/v1/session_result?session_key={key}`, and drivers via `/v1/drivers?session_key={key}` within the existing 5-minute poll cycle in `backend/internal/ingest/openf1_poller.go`
- [ ] T008 Wire SessionRepository into Cosmos client and inject into poller and router in `backend/cmd/api/main.go`

**Checkpoint**: Session data ingestion pipeline operational; Cosmos DB populated with session and result documents.

---

## Phase 3: User Story 4 - Navigate Between Calendar and Round Detail (Priority: P1) 🎯 MVP

**Goal**: Users can click any round in the calendar to view its detail page and navigate back. Round detail page serves as the shell for all session result stories.

**Independent Test**: Click a round in the calendar, verify navigation to `/rounds/{round}` with meeting metadata and session status sections. Click back link, verify return to calendar. Direct URL navigation works via bookmark.

### Tests for User Story 4

- [ ] T009 [P] [US4] Add contract test for `GET /api/v1/rounds/{round}?year=2026` validating RoundDetailResponse schema, 400 for invalid params, and 404 for unknown round in `backend/tests/contract/rounds_contract_test.go`
- [ ] T010 [P] [US4] Add frontend test for round detail page rendering, navigation from calendar, and back link in `frontend/tests/rounds/RoundDetailPage.test.tsx`

### Implementation for User Story 4

- [ ] T011 [P] [US4] Implement RoundDetailResponse, SessionDetailDTO, and SessionResultDTO structs matching openapi.yaml contract in `backend/internal/api/rounds/dto.go`
- [ ] T012 [US4] Implement round detail service that queries SessionRepository and assembles RoundDetailResponse with meeting metadata and per-session results in `backend/internal/api/rounds/service.go`
- [ ] T013 [US4] Implement HTTP handler for `GET /api/v1/rounds/{round}` with year query param, input validation, 400/404 error responses, and structured logging in `backend/internal/api/rounds/handler.go`
- [ ] T014 [US4] Register `/api/v1/rounds/{round}` route and inject session repository into router in `backend/internal/api/router.go`
- [ ] T015 [US4] Replace `useState<Page>` page-switching with react-router-dom `BrowserRouter`, adding routes for `/` (calendar), `/standings`, and `/rounds/:round` in `frontend/src/App.tsx`
- [ ] T016 [US4] Make calendar table rows clickable with `<Link to={/rounds/${round}}>` navigation in `frontend/src/features/calendar/CalendarPage.tsx`
- [ ] T017 [US4] Create round detail page shell displaying meeting header (race name, circuit, country, dates, status), session sections with status indicators (completed/in-progress/upcoming/not available), `data_as_of_utc` freshness, and back-to-calendar navigation link in `frontend/src/features/rounds/RoundDetailPage.tsx`
- [ ] T018 [P] [US4] Implement `fetchRoundDetail(year, round)` API client function in `frontend/src/features/rounds/roundApi.ts`

**Checkpoint**: US4 independently functional — calendar-to-detail navigation works with session metadata visible.

---

## Phase 4: User Story 1 - View Race Results for a Completed Round (Priority: P1)

**Goal**: Display race finishing order with positions, driver names, teams, time gaps, points scored, fastest lap indicator, and DNF/DNS/DSQ statuses.

**Independent Test**: Navigate to a completed round detail page and verify race results table shows all classified and non-classified finishers with correct positions, gaps, points, fastest lap marker, and retirement statuses.

### Tests for User Story 1

- [ ] T019 [P] [US1] Add frontend test for race results table rendering with classified finishers, DNF/DNS/DSQ statuses, fastest lap indicator, and points column in `frontend/tests/rounds/RaceResults.test.tsx`

### Implementation for User Story 1

- [ ] T020 [US1] Implement race results table component rendering position, driver name/acronym, team, gap to leader, race time (P1), points, fastest lap badge, and finishing status for DNF/DNS/DSQ entries at bottom of table in `frontend/src/features/rounds/RaceResults.tsx`
- [ ] T021 [US1] Integrate RaceResults component into RoundDetailPage session sections for `race` and `sprint` session types in `frontend/src/features/rounds/RoundDetailPage.tsx`

**Checkpoint**: US1 independently functional — race results table visible on round detail page.

---

## Phase 5: User Story 2 - View Qualifying Results for a Round (Priority: P2)

**Goal**: Display qualifying positions with driver names, teams, and best lap times across Q1/Q2/Q3 segments, with clear indication of elimination rounds.

**Independent Test**: Navigate to a round with qualifying data and verify the qualifying table shows positions, Q1/Q2/Q3 times, and correctly omits times for segments a driver did not reach.

### Implementation for User Story 2

- [x] T022 [US2] Implement qualifying results table component rendering grid position, driver name/acronym, team, Q1 time, Q2 time (null for Q1-eliminated), Q3 time (null for Q2-eliminated), and "Not yet available" state when session status is `upcoming` in `frontend/src/features/rounds/QualifyingResults.tsx`
- [x] T023 [US2] Integrate QualifyingResults component into RoundDetailPage session sections for `qualifying` and `sprint_qualifying` session types in `frontend/src/features/rounds/RoundDetailPage.tsx`

**Checkpoint**: US2 independently functional — qualifying results visible alongside race results.

---

## Phase 6: User Story 3 - View Practice Session Results (Priority: P3)

**Goal**: Display practice session times showing driver names, teams, best lap times, gap to fastest, and laps completed for each available practice session.

**Independent Test**: Navigate to a round with practice data and verify each practice session (FP1, FP2, FP3) renders a separate table with driver times and lap counts. Sprint weekends show only the available practice session with no empty placeholders.

### Implementation for User Story 3

- [x] T024 [US3] Implement practice results table component rendering position, driver name/acronym, team, best lap time, gap to fastest, and laps completed in `frontend/src/features/rounds/PracticeResults.tsx`
- [x] T025 [US3] Integrate PracticeResults component into RoundDetailPage session sections for `practice1`, `practice2`, `practice3` session types in `frontend/src/features/rounds/RoundDetailPage.tsx`

**Checkpoint**: US3 independently functional — all session types now render on round detail page.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Testing, observability validation, deployment checks, and success criteria verification.

- [x] T026 [P] Add unit tests for session transform logic covering session type slug mapping, qualifying Q1/Q2/Q3 extraction, fastest lap derivation, and DNF/DNS/DSQ status mapping in `backend/tests/unit/session_transform_test.go`
- [x] T027 [P] Add integration test for session ingestion flow verifying poll → transform → upsert → query round-trip in `backend/tests/integration/session_ingestion_test.go`
- [x] T028 [P] Extend network boundary test to validate no direct frontend calls to OpenF1 on the round detail page in `frontend/tests/integration/network_boundary.test.ts`
- [x] T029 Validate structured JSON logging for session ingestion and round detail API operations in `backend/tests/integration/log_schema_test.go`
- [x] T030 Verify Helm chart values and environment configuration support session data workloads in `deploy/helm/backend/values.yaml` and `deploy/helm/frontend/values.yaml`
- [x] T031 Run quickstart end-to-end validation and update notes in `specs/003-race-session-results/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies — can start immediately.
- **Phase 2 (Foundational)**: Depends on Phase 1; **BLOCKS all user stories**.
- **Phase 3 (US4 Navigation)**: Depends on Phase 2. Establishes the round detail page shell that US1–US3 extend.
- **Phase 4 (US1 Race Results)**: Depends on Phase 3 (needs RoundDetailPage shell and round API client).
- **Phase 5 (US2 Qualifying)**: Depends on Phase 3. Can run in parallel with US1 (different component files).
- **Phase 6 (US3 Practice)**: Depends on Phase 3. Can run in parallel with US1 and US2 (different component files).
- **Phase 7 (Polish)**: Depends on completion of desired user stories.

### User Story Dependencies

- **US4 (Navigation)** is the MVP foundation — establishes routing, round detail page, and API endpoint. Must complete first.
- **US1 (Race Results)**, **US2 (Qualifying)**, and **US3 (Practice)** are independent frontend components that integrate into the US4 page shell. They can proceed in parallel after US4.
- US1 is the highest-value content story and should be prioritized if sequential.

### Parallel Opportunities

**Phase 2**: T003 and T004 can run in parallel (different files). T006 can run in parallel with T005 after T003 completes.

**Phase 3**: T009, T010, T011, T018 can all run in parallel. Backend (T012–T014) and frontend (T015–T017) tracks can run concurrently.

**Phases 4–6**: US1, US2, and US3 implementation tasks can run fully in parallel since each creates/modifies different component files.

**Phase 7**: T026, T027, T028 can all run in parallel.

## Implementation Strategy

### MVP First

1. Complete Phase 1 and Phase 2 (setup + data ingestion pipeline).
2. Deliver US4 (Phase 3) — validate calendar-to-detail navigation and round API endpoint.
3. Deliver US1 (Phase 4) — validate race results display.
4. Demo/deploy MVP increment (navigation + race results).

### Incremental Delivery

1. Add US2 (Phase 5) qualifying results component.
2. Add US3 (Phase 6) practice results component.
3. Execute Phase 7 hardening and compliance checks.

### Verification Checklist

- **SC-001**: Calendar-to-detail-to-calendar navigation under 2 seconds total.
- **SC-002**: Race results show 100% positional accuracy relative to ingested source data.
- **SC-003**: `data_as_of_utc` freshness no older than 10 minutes for 95% of requests.
- **SC-004**: "Not yet available" messaging for future sessions; zero empty/broken tables.
- **SC-005**: All session types for a completed weekend available within 15 minutes of final session.
- **SC-006**: Sprint weekend formats display correct session structure without manual config.
- **SC-007**: Zero direct browser requests to OpenF1 on round detail page.
