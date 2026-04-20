# Tasks: Race Calendar and Championship Standings

**Input**: Design documents from `/specs/002-race-calendar-standings/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/openapi.yaml

**Tests**: Include contract, integration, and targeted unit tests for countdown, cancellation, and standings transforms.

**Organization**: Tasks are grouped by user story for independent implementation and validation.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: User story label (US1, US2, US3)
- Every task lists concrete file paths

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Initialize repo structure and delivery pipeline baselines.

- [x] T001 Create service directories and module scaffolding in `backend/` and `frontend/` with `deploy/helm/backend` and `deploy/helm/frontend`
- [x] T002 Initialize Go module and React workspace with minimal dependencies in `backend/go.mod` and `frontend/package.json`
- [x] T003 [P] Configure lint/format for Go and React in `backend/.golangci.yml`, `frontend/eslint.config.js`, and `frontend/prettier.config.cjs`
- [x] T004 [P] Create GitHub Actions CI/CD pipeline with ordered stages lint -> test -> build -> push -> deploy in `.github/workflows/ci-cd.yml`

---

## Phase 2: Foundational (Blocking)

**Purpose**: Core architecture, data, and security controls required before story work.

- [x] T005 Implement backend HTTP server bootstrap and Chi routing in `backend/cmd/api/main.go` and `backend/internal/api/router.go`
- [x] T006 [P] Implement Cosmos DB client, containers, and repository interfaces in `backend/internal/storage/cosmos/client.go` and `backend/internal/storage/repository.go`
- [x] T007 [P] Implement OpenF1 polling scheduler (5-minute cadence) and ingest pipeline in `backend/internal/ingest/openf1_poller.go`
- [x] T008 [P] Implement Hyprace polling client and standings ingest in `backend/internal/standings/hyprace_client.go`
- [x] T009 Enforce frontend-to-backend-only data access via API base client in `frontend/src/services/apiClient.ts`
- [x] T010 Configure Key Vault + Managed Identity secret loading in `backend/internal/config/secrets.go`
- [x] T011 Configure structured JSON logging and metrics emitters in `backend/internal/observability/logger.go` and `backend/internal/observability/metrics.go`
- [x] T012 Define Helm charts for backend and frontend workloads in `deploy/helm/backend/` and `deploy/helm/frontend/`
- [x] T013 Configure HTTPS ingress and firewall-ready egress policy templates in `deploy/helm/backend/templates/ingress.yaml` and `deploy/helm/backend/templates/networkpolicy.yaml`
- [x] T014 Add dependency justification ledger for backend/frontend additions in `specs/002-race-calendar-standings/dependency-justification.md`

**Checkpoint**: Foundation complete, user stories can proceed.

---

## Phase 3: User Story 1 - View Full 2026 Race Calendar (Priority: P1) 🎯 MVP

**Goal**: Serve and render all 24 rounds of 2026 with required metadata.

**Independent Test**: Calendar API returns 24 unique rounds with required fields and frontend renders full list.

### Tests for User Story 1

- [x] T015 [P] [US1] Add contract tests for `GET /api/v1/calendar` in `backend/tests/contract/calendar_contract_test.go`
- [x] T016 [P] [US1] Add integration tests for poll-to-cache-to-API flow in `backend/tests/integration/calendar_cache_flow_test.go`
- [x] T017 [P] [US1] Add frontend component tests for calendar table rendering in `frontend/tests/calendar/CalendarTable.test.tsx`

### Implementation for User Story 1

- [x] T018 [P] [US1] Implement `RaceMeeting` domain model and status enum in `backend/internal/domain/race_meeting.go`
- [x] T019 [US1] Implement OpenF1 meetings normalization for 2026 in `backend/internal/ingest/meeting_transform.go`
- [x] T020 [US1] Implement calendar repository persistence/query logic in `backend/internal/storage/cosmos/calendar_repository.go`
- [x] T021 [US1] Implement calendar service and response shaping in `backend/internal/api/calendar/service.go`
- [x] T022 [US1] Implement `GET /api/v1/calendar` handler in `backend/internal/api/calendar/handler.go`
- [x] T023 [US1] Implement calendar page and race rows in `frontend/src/features/calendar/CalendarPage.tsx`
- [x] T024 [US1] Wire frontend calendar API service in `frontend/src/features/calendar/calendarApi.ts`

**Checkpoint**: US1 independently functional.

---

## Phase 4: User Story 2 - Track Upcoming Race Countdown (Priority: P1)

**Goal**: Highlight next non-cancelled race and display live countdown.

**Independent Test**: Backend exposes next-round/countdown fields; frontend displays one highlighted race and updates over time.

### Tests for User Story 2

- [x] T025 [P] [US2] Add unit tests for next-race selection and tie-break rules in `backend/tests/unit/next_race_selector_test.go`
- [x] T026 [P] [US2] Add integration tests for countdown target transitions in `backend/tests/integration/countdown_transition_test.go`
- [x] T027 [P] [US2] Add frontend timer tests for live countdown behavior in `frontend/tests/calendar/CountdownCard.test.tsx`

### Implementation for User Story 2

- [x] T028 [US2] Implement next-race selection service excluding past/cancelled rounds in `backend/internal/domain/next_race_selector.go`
- [x] T029 [US2] Extend calendar API contract fields (`next_round`, `countdown_target_utc`) in `backend/internal/api/calendar/service.go`
- [x] T030 [US2] Implement countdown card and highlighted race UI in `frontend/src/features/calendar/NextRaceCard.tsx`
- [x] T031 [US2] Implement client-side countdown refresh hook in `frontend/src/features/calendar/useCountdown.ts`

**Checkpoint**: US2 independently functional.

---

## Phase 5: User Story 3 - Cancelled Rounds and Championship Standings (Priority: P2)

**Goal**: Mark Bahrain R4 and Saudi Arabia R5 as cancelled and provide drivers/constructors standings tables.

**Independent Test**: Cancelled rounds always marked and excluded from next-race; standings endpoints return required columns.

### Tests for User Story 3

- [x] T032 [P] [US3] Add contract tests for standings endpoints in `backend/tests/contract/standings_contract_test.go`
- [x] T033 [P] [US3] Add integration tests for cancellation override behavior in `backend/tests/integration/cancelled_rounds_test.go`
- [x] T034 [P] [US3] Add frontend tests for cancellation badges and standings tables in `frontend/tests/standings/StandingsPage.test.tsx`

### Implementation for User Story 3

- [x] T035 [US3] Implement cancellation override policy for R4 Bahrain and R5 Saudi Arabia in `backend/internal/domain/cancellation_overrides.go`
- [x] T036 [US3] Add cancellation metadata fields in calendar DTOs in `backend/internal/api/calendar/dto.go`
- [x] T037 [US3] Implement drivers standings aggregation service in `backend/internal/standings/drivers_service.go`
- [x] T038 [US3] Implement constructors standings aggregation service in `backend/internal/standings/constructors_service.go`
- [x] T039 [US3] Implement standings handlers in `backend/internal/api/standings/handler.go`
- [x] T040 [US3] Implement standings UI page and tables in `frontend/src/features/standings/StandingsPage.tsx`
- [x] T041 [US3] Render cancelled indicators in calendar rows in `frontend/src/features/calendar/CancelledRaceBadge.tsx`

**Checkpoint**: US3 independently functional.

---

## Phase 6: Polish and Cross-Cutting

**Purpose**: Final hardening, compliance checks, and validation against success criteria.

- [ ] T042 [P] Add API latency and poll success dashboards/alerts configuration in `deploy/monitoring/azure-monitor-dashboard.json`
- [ ] T043 Validate structured JSON log schema consistency in `backend/tests/integration/log_schema_test.go`
- [ ] T044 [P] Validate no direct frontend calls to OpenF1/Hyprace in `frontend/tests/integration/network_boundary.test.ts`
- [ ] T045 Verify Helm values and environment overlays for AKS in `deploy/helm/backend/values.yaml` and `deploy/helm/frontend/values.yaml`
- [ ] T046 Verify GitHub Actions deployment guards and image promotion logic in `.github/workflows/ci-cd.yml`
- [ ] T047 Run quickstart end-to-end validation and update notes in `specs/002-race-calendar-standings/quickstart.md`

---

## Dependencies and Execution Order

### Phase Dependencies

- Phase 1 -> no dependencies.
- Phase 2 -> depends on Phase 1; blocks all user stories.
- Phase 3 (US1) -> depends on Phase 2.
- Phase 4 (US2) -> depends on Phase 2 and reuses US1 calendar contracts.
- Phase 5 (US3) -> depends on Phase 2; may run in parallel with US2 once shared calendar contract is stable.
- Phase 6 -> depends on completion of desired user stories.

### User Story Dependencies

- US1 is MVP and should be completed first.
- US2 depends on calendar data shape from US1 but remains independently testable.
- US3 depends on foundational ingest/storage and can proceed after Phase 2; cancellation badge rendering touches US1 UI.

### Parallel Opportunities

- T003 and T004 can run in parallel.
- T006, T007, T008 can run in parallel.
- Within each user story, test tasks marked [P] can run in parallel.
- Frontend and backend implementation tasks with separate files can run concurrently.

## Implementation Strategy

### MVP First

1. Complete Phase 1 and Phase 2.
2. Deliver US1 (Phase 3), validate 24-round calendar behavior.
3. Demo/deploy MVP increment.

### Incremental Delivery

1. Add US2 countdown behavior and validate transition logic.
2. Add US3 cancellation + standings behavior.
3. Execute Phase 6 hardening and compliance checks.

### Verification Checklist

- SC-001: 24 rounds present and unique.
- SC-002: R4/R5 cancelled and excluded from next-race selection.
- SC-003: Poll job success-rate instrumentation present.
- SC-004: `data_as_of_utc` freshness visible and tested.
- SC-005: API latency measured against targets.
- SC-006: Browser does not call external APIs directly.
- SC-007: Standings fields complete and populated.