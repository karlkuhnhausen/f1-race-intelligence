# Tasks: Session Deep Dive Page

**Input**: Design documents from `/specs/006-session-deep-dive/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/analysis-api.md

**Tests**: Unit tests for backend domain/storage/ingestion layers, contract tests for the API endpoint, and component tests for frontend chart components — included as specified by the user.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story. The six user stories are implemented in priority order (P1 → P2 → P3).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: User story label — US1 through US6 per spec.md; setup and foundational tasks carry no story label
- Every task includes exact file paths

## Implementation Order

`domain types → storage interface → cosmos impl → ingest fetchers + aggregation → API handler/service/DTO → frontend types + API client → page shell + routing → chart components → round detail integration → backfill CLI extension → tests GREEN`

---

## Phase 1: Setup

**Purpose**: Install new dependency and create directory structure for the analysis feature.

- [ ] T001 Install `recharts` dependency in `frontend/` — run `npm install recharts` (justification: [dependency-justification.md](dependency-justification.md))
- [ ] T002 [P] Create directory `frontend/src/features/analysis/` and `frontend/tests/analysis/`
- [ ] T003 [P] Create directory `backend/internal/api/analysis/`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Backend domain types, storage interface, Cosmos implementation, and ingest layer — the shared infrastructure that ALL user stories depend on.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

### Domain Types

- [ ] T004 Define domain types (`AnalysisPosition`, `PositionLap`, `AnalysisInterval`, `IntervalLap`, `AnalysisStint`, `AnalysisPit`, `AnalysisOvertake`, `AnalysisData`) in `backend/internal/domain/analysis.go` per data-model.md §Domain Layer
- [ ] T005 [P] Write unit tests for domain type construction helpers (if any) in `backend/tests/unit/analysis_domain_test.go`

### Storage Layer

- [ ] T006 Add `AnalysisRepository` interface to `backend/internal/storage/repository.go` — 7 methods: `UpsertSessionPositions`, `UpsertSessionIntervals`, `UpsertSessionStints`, `UpsertSessionPits`, `UpsertSessionOvertakes`, `GetSessionAnalysis`, `HasAnalysisData` per data-model.md §Storage Layer
- [ ] T007 Define Cosmos document types (`SessionPosition`, `SessionInterval`, `SessionStint`, `SessionPit`, `SessionOvertake`, `SessionAnalysisData`) with JSON tags and composite document IDs in `backend/internal/storage/cosmos/analysis.go`
- [ ] T008 Implement `UpsertSessionPositions`, `UpsertSessionIntervals`, `UpsertSessionStints`, `UpsertSessionPits`, `UpsertSessionOvertakes` in `backend/internal/storage/cosmos/analysis.go` — use container `sessions`, partition key `season`, idempotent upsert by composite document ID
- [ ] T009 Implement `GetSessionAnalysis` in `backend/internal/storage/cosmos/analysis.go` — single partition query with `STARTSWITH(c.type, 'analysis_')` filter, deserialize into `SessionAnalysisData`
- [ ] T010 Implement `HasAnalysisData` in `backend/internal/storage/cosmos/analysis.go` — `COUNT` query on `analysis_position` documents for the session
- [ ] T011 [P] Write unit tests for Cosmos document ID generation and type mapping in `backend/tests/unit/analysis_storage_test.go`

### Ingest Layer

- [ ] T012 Create `backend/internal/ingest/analysis.go` — define raw OpenF1 response structs (`RawPosition`, `RawInterval`, `RawStint`, `RawPit`, `RawOvertake`) and `AnalysisFetchResult` struct
- [ ] T013 Implement `FetchPositionData(ctx, client, sessionKey)` in `backend/internal/ingest/analysis.go` — GET `/position?session_key={key}`, return raw entries
- [ ] T014 Implement `AggregatePositions(raw []RawPosition, drivers map[int]DriverInfo) []domain.AnalysisPosition` in `backend/internal/ingest/analysis.go` — deduplicate keeping LAST entry per (driver_number, lap_number), attach driver metadata, produce 1 point per driver per lap (FR-010)
- [ ] T015 [P] Implement `FetchIntervalData` and `AggregateIntervals` in `backend/internal/ingest/analysis.go` — same deduplication pattern for interval data
- [ ] T016 [P] Implement `FetchStintData` in `backend/internal/ingest/analysis.go` — map directly to domain type, normalize compound to uppercase
- [ ] T017 [P] Implement `FetchPitData` in `backend/internal/ingest/analysis.go` — map `pit_duration` and `duration` fields to domain type
- [ ] T018 [P] Implement `FetchOvertakeData` in `backend/internal/ingest/analysis.go` — handle empty array gracefully (not an error per FR-015)
- [ ] T019 Implement `FetchAllAnalysisData(ctx, client, sessionKey, drivers)` in `backend/internal/ingest/analysis.go` — orchestrate all 5 fetches with 500ms delays between each, position data required (error if fails), other 4 non-fatal on failure, structured JSON logging per fetch
- [ ] T020 [P] Write unit tests for `AggregatePositions` in `backend/tests/unit/analysis_ingest_test.go` — cases: multiple entries per driver per lap (keep last), single entry per lap (passthrough), empty input, driver not in drivers map (skip gracefully)
- [ ] T021 [P] Write unit tests for `AggregateIntervals` in `backend/tests/unit/analysis_ingest_test.go` — cases: deduplication, empty input, leader always 0 gap
- [ ] T022 [P] Write unit tests for `FetchAllAnalysisData` orchestration in `backend/tests/unit/analysis_ingest_test.go` — cases: all succeed, position fails (returns error), intervals fail (non-fatal, logged), rate limiting delays applied

**Checkpoint**: Domain types compile; storage interface compiles; Cosmos implementation handles upsert and query; ingest fetchers and aggregation are tested. All user story phases can now proceed.

---

## Phase 3: User Story 1 — Position Battle Chart (Priority: P1) 🎯 MVP

**Goal**: A fan clicks "View Analysis" on a completed race round detail page and sees a lap-by-lap position chart with all 20 drivers as individually colored lines showing the race narrative.

**Independent Test**: Navigate to `/rounds/4/sessions/race/analysis?year=2026`. A multi-line position chart renders with correct driver positions per lap, inverted Y-axis (1 at top), team-colored lines, and handles DNF (line ends at retirement lap). Shows "Analysis not yet available" for sessions without data.

### Tests for User Story 1

- [ ] T023 [P] [US1] Write contract test `TestGetSessionAnalysis_200` in `backend/tests/contract/analysis_contract_test.go` — verify 200 response shape matches contract (positions non-null, intervals/stints/pits/overtakes nullable)
- [ ] T024 [P] [US1] Write contract test `TestGetSessionAnalysis_404_NoData` in `backend/tests/contract/analysis_contract_test.go` — verify 404 with `analysis_not_available` error when no data exists
- [ ] T025 [P] [US1] Write contract test `TestGetSessionAnalysis_400_InvalidRound` in `backend/tests/contract/analysis_contract_test.go` — verify 400 for non-integer round param
- [ ] T026 [P] [US1] Write frontend component test for `PositionChart` in `frontend/tests/analysis/PositionChart.test.tsx` — cases: renders SVG with 20 lines, Y-axis inverted (P1 at top), handles DNF (fewer laps), empty data shows placeholder
- [ ] T027 [P] [US1] Write frontend component test for `AnalysisPage` in `frontend/tests/analysis/AnalysisPage.test.tsx` — cases: loading state, renders all chart sections when data present, shows "Analysis not yet available" on 404, back-navigation link present

### Implementation for User Story 1

- [ ] T028 [US1] Create `backend/internal/api/analysis/dto.go` — define `SessionAnalysisDTO`, `PositionDriverDTO`, `PositionLapDTO`, `IntervalDriverDTO`, `IntervalLapDTO`, `StintDTO`, `PitDTO`, `OvertakeDTO` per data-model.md §API Layer
- [ ] T029 [US1] Create `backend/internal/api/analysis/service.go` — `NewService(repo AnalysisRepository, logger)`, `GetSessionAnalysis(ctx, season, round, sessionType)` that calls `repo.GetSessionAnalysis` and maps domain types to DTOs
- [ ] T030 [US1] Create `backend/internal/api/analysis/handler.go` — Chi-compatible `GetSessionAnalysis` handler; extract `round` and `type` path params, `year` query param (default 2026); validate session type is "race" or "sprint"; return 404 with `analysis_not_available` error if no data; return 400 for invalid params
- [ ] T031 [US1] Register analysis route in `backend/internal/api/router.go` — `r.Get("/rounds/{round}/sessions/{type}/analysis", analysisHandler.GetSessionAnalysis)` with service/handler wiring
- [ ] T032 [P] [US1] Create `frontend/src/features/analysis/analysisTypes.ts` — TypeScript interfaces matching `SessionAnalysisDTO` contract response shape
- [ ] T033 [P] [US1] Create `frontend/src/features/analysis/analysisApi.ts` — `fetchSessionAnalysis(round, sessionType, year)` function calling `GET /api/v1/rounds/{round}/sessions/{type}/analysis?year={year}`, handle 404 → return null
- [ ] T034 [US1] Create `frontend/src/features/analysis/AnalysisPage.tsx` — page shell with route params extraction, data fetching on mount, loading/error/empty states, back-navigation link ("← Back to Round {N}"), session header, vertically-stacked chart section placeholders
- [ ] T035 [US1] Add analysis route to `frontend/src/app/routes.tsx` — `<Route path="/rounds/:round/sessions/:sessionType/analysis" element={<AnalysisPage />} />`
- [ ] T036 [US1] Create `frontend/src/features/analysis/PositionChart.tsx` — recharts `LineChart` with `ResponsiveContainer`, inverted Y-axis (1 top, 20 bottom), one `Line` per driver colored by team, X = lap number; handles empty/null position data with placeholder message
- [ ] T037 [US1] Integrate `PositionChart` into `AnalysisPage.tsx` — pass position data from fetched response, show chart section only when positions array is non-empty

**Checkpoint**: User can navigate to `/rounds/{round}/sessions/race/analysis` and see a position chart. Backend API returns 200 with position data, 404 when not available. Contract tests pass.

---

## Phase 4: User Story 2 — Backfill Existing 2026 Sessions (Priority: P1)

**Goal**: A system operator runs the backfill CLI with `--analysis` flag to populate analysis data for all existing finalized 2026 Race and Sprint sessions, ensuring the feature has data on deployment day.

**Independent Test**: Run `backfill --season=2026 --analysis --dry-run` and verify structured JSON log lines per session (skipped/would-update). Run without `--dry-run` against Cosmos and verify analysis documents are created. Re-run and verify all sessions are skipped (idempotent).

### Tests for User Story 2

- [ ] T038 [P] [US2] Write unit test for backfill analysis logic in `backend/tests/unit/backfill_analysis_test.go` — cases: skips sessions with existing data (HasAnalysisData=true), processes sessions without data, continues on individual session failure, respects rate limiting, dry-run mode does not write

### Implementation for User Story 2

- [ ] T039 [US2] Extend `backend/cmd/backfill/main.go` — add `--analysis` flag (bool, default false); when set: query `GetFinalizedSessions` filtered to Race and Sprint types, for each check `HasAnalysisData` (skip if true), call `FetchAllAnalysisData`, persist via `AnalysisRepository` upsert methods, rate limit 1000ms between sessions + 500ms between endpoints
- [ ] T040 [US2] Add structured JSON logging to backfill analysis path — per-session log line (`session_key`, `round`, `session_type`, `outcome: updated|skipped|failed`, counts per data type), summary line at completion (`updated`, `skipped`, `failed` counts)
- [ ] T041 [US2] Wire `AnalysisRepository` (Cosmos implementation) into backfill CLI — same config/client initialization pattern as existing backfill; import ingest analysis functions

**Checkpoint**: Operator can run `backfill --season=2026 --analysis` and all finalized Race/Sprint sessions get analysis data populated. Idempotent on re-run.

---

## Phase 5: User Story 3 — Tire Strategy Swimlane (Priority: P2)

**Goal**: A user viewing the analysis page sees a horizontal swimlane chart showing each driver's tire compound choices across their race laps as colored blocks.

**Independent Test**: Load a session analysis page with stint data and verify each driver's row shows correctly-colored blocks (Soft=red, Medium=yellow, Hard=white, Inter=green, Wet=blue) with accurate lap ranges.

### Tests for User Story 3

- [ ] T042 [P] [US3] Write frontend component test for `TireStrategyChart` in `frontend/tests/analysis/TireStrategyChart.test.tsx` — cases: renders one row per driver, correct compound colors, handles multi-stint drivers (2-3 stints), empty stint data shows placeholder, single-stint sprint (one block full width)

### Implementation for User Story 3

- [ ] T043 [US3] Create `frontend/src/features/analysis/TireStrategyChart.tsx` — recharts `BarChart` with horizontal stacked bars; compound-colored (SOFT=red, MEDIUM=yellow, HARD=white, INTERMEDIATE=green, WET=blue); one row per driver sorted by finishing position; `ResponsiveContainer` for mobile
- [ ] T044 [US3] Integrate `TireStrategyChart` into `AnalysisPage.tsx` — show section only when stints array is non-null and non-empty (FR-015 graceful degradation)

**Checkpoint**: Tire strategy swimlane renders alongside position chart when stint data is present. Omitted gracefully when data is null.

---

## Phase 6: User Story 4 — Pit Stop Timeline (Priority: P2)

**Goal**: A user sees a timeline visualization showing when each driver pitted and how long each stop took, making pit windows and undercut/overcut strategies visible.

**Independent Test**: Load a session with pit data and verify each pit stop displays with correct lap number, driver, and duration. Slow stops (>5s) are visually distinct.

### Tests for User Story 4

- [ ] T045 [P] [US4] Write frontend component test for `PitStopTimeline` in `frontend/tests/analysis/PitStopTimeline.test.tsx` — cases: renders scatter dots for each pit stop, slow stops (>5s) have different styling, correct lap placement on X-axis, empty pit data shows placeholder

### Implementation for User Story 4

- [ ] T046 [US4] Create `frontend/src/features/analysis/PitStopTimeline.tsx` — recharts `ScatterChart` with lap on X-axis, drivers on Y-axis (categorical), dot size proportional to stop duration; distinguish slow stops (>5s) with different color/size; `ResponsiveContainer`; tooltip showing driver, lap, pit duration, stop duration
- [ ] T047 [US4] Integrate `PitStopTimeline` into `AnalysisPage.tsx` — show section only when pits array is non-null and non-empty

**Checkpoint**: Pit stop timeline renders alongside other charts. Slow stops are visually distinguishable.

---

## Phase 7: User Story 5 — Gap-to-Leader Progression (Priority: P2)

**Goal**: A user can see how the time gap between each driver and the race leader evolved over the course of the race, revealing pace differentials and safety car compressions.

**Independent Test**: Load a session with interval data and verify a line chart shows gap-to-leader values over laps for all drivers. Leader line stays at 0.

### Tests for User Story 5

- [ ] T048 [P] [US5] Write frontend component test for `GapToLeaderChart` in `frontend/tests/analysis/GapToLeaderChart.test.tsx` — cases: renders one line per driver, leader line at 0, handles safety car compression (gaps converge), empty interval data shows placeholder

### Implementation for User Story 5

- [ ] T049 [US5] Create `frontend/src/features/analysis/GapToLeaderChart.tsx` — recharts `LineChart` with gap (seconds) on Y-axis, lap number on X-axis, one `Line` per driver colored by team; `ResponsiveContainer`; tooltip showing driver, lap, gap value
- [ ] T050 [US5] Integrate `GapToLeaderChart` into `AnalysisPage.tsx` — show section only when intervals array is non-null and non-empty

**Checkpoint**: Gap-to-leader chart renders showing pace differentials over race distance.

---

## Phase 8: User Story 6 — Overtake Annotations (Priority: P3)

**Goal**: A user sees markers on the position chart indicating where overtakes occurred, enriching the race narrative with "who passed whom" context.

**Independent Test**: Load a session with overtake data and verify annotation markers appear on the position chart at correct laps. When overtake data is unavailable, position chart renders normally without annotations.

### Tests for User Story 6

- [ ] T051 [P] [US6] Write frontend component test for overtake annotations in `frontend/tests/analysis/PositionChart.test.tsx` (extend existing) — cases: renders annotation markers when overtake data present, no markers when overtakes is null, tooltip shows overtaking/overtaken driver names

### Implementation for User Story 6

- [ ] T052 [US6] Extend `frontend/src/features/analysis/PositionChart.tsx` — add `ReferenceDot` markers at overtake laps on the relevant driver's line; tooltip on hover/tap shows overtaking driver, overtaken driver, resulting position; gracefully skip when overtakes prop is null/empty (FR-015)

**Checkpoint**: Overtake annotations enrich the position chart when data is available. No errors when data is absent.

---

## Phase 9: Integration & Session Poller Extension

**Purpose**: Wire analysis ingestion into the live session poller and add "View Analysis" navigation from round detail page.

- [ ] T053 Extend `backend/internal/ingest/session_poller.go` — after `fetchAndUpsertResults` succeeds for Race/Sprint sessions at finalization (2h buffer elapsed): call `FetchAllAnalysisData` with session key and driver info, persist via `AnalysisRepository` upsert methods; 500ms delay between endpoints; log error and continue on failure (non-blocking for session finalization)
- [ ] T054 [P] Write unit test for session poller analysis integration in `backend/tests/unit/session_poller_analysis_test.go` — cases: triggers analysis fetch for race sessions, triggers for sprint sessions, does NOT trigger for qualifying/practice, continues on analysis fetch failure (session still finalized)
- [ ] T055 Extend `frontend/src/features/rounds/RoundDetailPage.tsx` — add "View Analysis →" link/button for completed Race and Sprint sessions only (FR-002, FR-003); link targets `/rounds/{round}/sessions/{sessionType}/analysis?year={year}`
- [ ] T056 [P] Write frontend test for "View Analysis" button visibility in `frontend/tests/rounds/RoundDetailAnalysisLink.test.tsx` — cases: shows for completed race, shows for completed sprint, hidden for qualifying, hidden for practice, hidden for upcoming sessions

**Checkpoint**: Live sessions get analysis data automatically after finalization. Users can navigate from round detail to analysis page.

---

## Phase 10: Polish & Cross-Cutting Concerns

**Purpose**: Final validation, mobile responsiveness, caching headers, and documentation.

- [ ] T057 Add `Cache-Control: public, max-age=86400` header to 200 responses in `backend/internal/api/analysis/handler.go` (immutable post-session data per contract)
- [ ] T058 [P] Verify all charts render mobile-responsive (≤768px full-width, stacked vertically) — manual check against `ResponsiveContainer` usage in all 4 chart components (FR-018)
- [ ] T059 [P] Run full backend test suite: `cd backend && go test ./...` — all tests pass including new analysis tests
- [ ] T060 [P] Run full frontend test suite: `cd frontend && npx vitest run` — all tests pass including new analysis component tests
- [ ] T061 [P] Run linter: `cd backend && golangci-lint run ./...` and `cd frontend && npx tsc --noEmit` — no errors
- [ ] T062 Run `specs/006-session-deep-dive/quickstart.md` validation scenarios end-to-end

**Checkpoint**: Feature complete. All tests green. Linters pass. Ready for deployment.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 (recharts installed for frontend tests) — BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Phase 2 — delivers MVP (position chart + API endpoint)
- **User Story 2 (Phase 4)**: Depends on Phase 2 — can run in parallel with Phase 3
- **User Story 3 (Phase 5)**: Depends on Phase 3 (AnalysisPage shell exists) — parallel with Phases 4, 6, 7
- **User Story 4 (Phase 6)**: Depends on Phase 3 (AnalysisPage shell exists) — parallel with Phases 4, 5, 7
- **User Story 5 (Phase 7)**: Depends on Phase 3 (AnalysisPage shell exists) — parallel with Phases 4, 5, 6
- **User Story 6 (Phase 8)**: Depends on Phase 3 (PositionChart exists) — after Phase 3
- **Integration (Phase 9)**: Depends on Phase 2 (ingest + storage layers) and Phase 3 (frontend page) — can overlap with Phases 5-8
- **Polish (Phase 10)**: Depends on all previous phases complete

### Within Each User Story

- Tests written FIRST, confirmed to FAIL before implementation
- Backend: domain → storage → ingest → API handler
- Frontend: types → API client → page shell → chart components → integration
- Story complete before moving to next priority (unless parallelizing)

### Parallel Opportunities

- **Phase 2**: T005, T011, T020, T021, T022 (unit tests) can run in parallel once their target code exists
- **Phase 2**: T015, T16, T017, T018 (individual fetchers) are independent files/functions
- **Phase 3**: T023-T027 (all tests) can be written in parallel; T032, T033 (frontend types/API) parallel with backend work
- **Phase 4**: Can run entirely in parallel with Phase 3 (different files)
- **Phases 5, 6, 7**: All three chart components are independent — can be implemented in parallel
- **Phase 9**: T054, T056 (tests) parallel; T053, T055 (implementation) on different files

---

## Summary

| Metric | Value |
|--------|-------|
| Total tasks | 62 |
| Phase 1 (Setup) | 3 tasks |
| Phase 2 (Foundational) | 19 tasks |
| Phase 3 (US1 - Position Chart) | 15 tasks |
| Phase 4 (US2 - Backfill) | 4 tasks |
| Phase 5 (US3 - Tire Strategy) | 3 tasks |
| Phase 6 (US4 - Pit Timeline) | 3 tasks |
| Phase 7 (US5 - Gap-to-Leader) | 3 tasks |
| Phase 8 (US6 - Overtake Annotations) | 2 tasks |
| Phase 9 (Integration) | 4 tasks |
| Phase 10 (Polish) | 6 tasks |
| Parallelizable tasks | 28 (45%) |
| **MVP scope** | Phases 1-3 (Position Chart + API) |
| **Deployment-ready scope** | Phases 1-4 (MVP + Backfill) |
