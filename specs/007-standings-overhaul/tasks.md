# Tasks: Standings Overhaul

**Input**: Design documents from `/specs/007-standings-overhaul/`  
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/api.md

**Tests**: Included where tests already exist and need updating.

**Organization**: Tasks grouped by user story. US1 and US2 are both P1 and share foundational ingestion work.

## Format: `[ID] [P?] [Story?] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to (US1–US6)
- File paths are relative to repository root

---

## Phase 1: Setup (Hyprace Removal & Storage Schema)

**Purpose**: Remove dead code and prepare storage layer for new data types

- [ ] T001 Delete `backend/internal/standings/hyprace_client.go` entirely
- [ ] T002 Remove Hyprace poller instantiation and goroutine launch from `backend/cmd/api/main.go`
- [ ] T003 [P] Remove `source = 'hyprace'` filter from Cosmos queries in `backend/internal/storage/cosmos/client.go` — replace with `type` discriminator-based queries
- [ ] T004 [P] Add new storage types (`DriverChampionshipSnapshot`, `TeamChampionshipSnapshot`, `SessionResult`, `StartingGrid`) to `backend/internal/storage/repository.go` per data-model.md
- [ ] T005 [P] Add `ChampionshipRepository` interface methods to `backend/internal/storage/repository.go`: `UpsertDriverChampionshipSnapshots`, `GetDriverChampionshipSnapshots`, `UpsertTeamChampionshipSnapshots`, `GetTeamChampionshipSnapshots`, `UpsertSessionResults`, `GetSessionResults`, `UpsertStartingGrids`, `GetStartingGrids`
- [ ] T006 Implement new Cosmos DB methods for championship snapshots and session results in `backend/internal/storage/cosmos/client.go`
- [ ] T007 [P] Remove any Hyprace egress firewall rules from `infra/bicep/` and Helm values in `deploy/helm/backend/`

**Checkpoint**: Hyprace is fully removed. Storage layer supports new document types. Backend still compiles and passes existing tests.

---

## Phase 2: Foundational (Championship Ingestion Engine)

**Purpose**: Core ingestion module that fetches and transforms OpenF1 championship data — MUST complete before user stories

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T008 Create `backend/internal/standings/championship_ingester.go` with `ChampionshipIngester` struct that fetches `/v1/championship_drivers?session_key={key}` and `/v1/championship_teams?session_key={key}` from OpenF1, transforms responses to `DriverChampionshipSnapshot` and `TeamChampionshipSnapshot`, and upserts to Cosmos via repository
- [ ] T009 [P] Create `backend/internal/standings/stats_aggregator.go` with `StatsAggregator` that queries session results and starting grids from storage and computes per-driver stats (wins, podiums, DNFs, poles) and per-team stats (wins, podiums, DNFs)
- [ ] T010 Add OpenF1 session result ingestion to `backend/internal/standings/championship_ingester.go`: fetch `/v1/session_result?session_key={key}` and transform to `SessionResult` documents
- [ ] T011 Add OpenF1 starting grid ingestion to `backend/internal/standings/championship_ingester.go`: fetch `/v1/starting_grid?meeting_key={meeting_key}` and transform to `StartingGrid` documents
- [ ] T012 Extend session poller finalization hook in `backend/internal/ingest/session_poller.go` to trigger `ChampionshipIngester.IngestSession(ctx, sessionKey, meetingKey)` after Race and Sprint sessions are finalized
- [ ] T013 Wire `ChampionshipIngester` instantiation in `backend/cmd/api/main.go` — inject into session poller
- [ ] T014 [P] Add `--championship` flag to `backend/cmd/backfill/main.go` that iterates all completed Race and Sprint sessions for a given season and calls `ChampionshipIngester.IngestSession` for each with rate limiting (500ms between requests)
- [ ] T015 [P] Create unit tests in `backend/tests/unit/championship_ingester_test.go` — test transformation of OpenF1 JSON responses to storage types, null handling for `position_start`/`points_start`, and team name resolution for null team names
- [ ] T016 [P] Create unit tests in `backend/tests/unit/stats_aggregator_test.go` — test wins/podiums/DNFs/poles computation from mock session results and starting grids

**Checkpoint**: Championship data can be ingested from OpenF1 and stored in Cosmos. Backfill CLI can populate historical data. Stats can be computed from stored results.

---

## Phase 3: User Story 1 — Real Championship Standings (Priority: P1) 🎯 MVP

**Goal**: Standings tables display real data from OpenF1 championship endpoints with team colors

**Independent Test**: Navigate to `/standings` — Drivers and Constructors tabs show real rows with position, name, team color, and points for the 2026 season.

### Implementation

- [ ] T017 [US1] Update `backend/internal/api/standings/dto.go` — expand `DriverStandingDTO` to include `driver_number`, `team_color`; expand `ConstructorStandingDTO` to include `team_color`; add `data_as_of_utc` to response envelopes
- [ ] T018 [US1] Update `backend/internal/api/standings/service.go` — rewrite `GetDrivers(ctx, season)` to read latest `DriverChampionshipSnapshot` per driver for the season, join with driver identity for names/colors, and return expanded DTO; rewrite `GetConstructors(ctx, season)` similarly using `TeamChampionshipSnapshot`
- [ ] T019 [US1] Update `backend/internal/api/standings/handler.go` — update `parseYear` validation to enforce range 2023–current year; default year to current when not provided
- [ ] T020 [P] [US1] Update `frontend/src/features/standings/standingsApi.ts` — align TypeScript interfaces with expanded backend DTOs (`driver_number`, `team_color` fields)
- [ ] T021 [P] [US1] Update `frontend/src/features/design-system/StandingsTable.tsx` — render `team_color` as left-border accent on each row
- [ ] T022 [US1] Update `frontend/src/features/standings/StandingsPage.tsx` — pass `team_color` through to `StandingsTable`
- [ ] T023 [US1] Update contract tests in `backend/tests/contract/standings_contract_test.go` — verify new response shapes for `/standings/drivers` and `/standings/constructors` (expanded fields, `data_as_of_utc`)
- [ ] T024 [US1] Update frontend tests in `frontend/tests/standings/StandingsPage.test.tsx` — verify table renders with team color accents and all expected columns

**Checkpoint**: Standings page shows real championship data. MVP is usable.

---

## Phase 4: User Story 2 — Expanded Statistics Columns (Priority: P1)

**Goal**: Standings tables include wins, podiums, DNFs, and poles alongside points

**Independent Test**: Navigate to `/standings` — each driver row shows numeric values for wins, podiums, DNFs, poles. Constructor rows show wins, podiums, DNFs.

### Implementation

- [ ] T025 [US2] Extend `backend/internal/api/standings/dto.go` — add `Wins`, `Podiums`, `DNFs`, `Poles` to `DriverStandingDTO`; add `Wins`, `Podiums`, `DNFs` to `ConstructorStandingDTO`
- [ ] T026 [US2] Extend `backend/internal/api/standings/service.go` — call `StatsAggregator.GetDriverStats(ctx, season)` and `GetTeamStats(ctx, season)` to populate stats fields in standings responses
- [ ] T027 [P] [US2] Update `frontend/src/features/standings/standingsApi.ts` — add `wins`, `podiums`, `dnfs`, `poles` to TypeScript interfaces
- [ ] T028 [P] [US2] Update `frontend/src/features/design-system/StandingsTable.tsx` — add optional columns for podiums, DNFs, poles (extend existing `columns` prop to support all stat types)
- [ ] T029 [US2] Update `frontend/src/features/standings/StandingsPage.tsx` — configure `StandingsTable` to show all stats columns for drivers (`wins`, `podiums`, `dnfs`, `poles`) and constructors (`wins`, `podiums`, `dnfs`)
- [ ] T030 [US2] Update contract tests in `backend/tests/contract/standings_contract_test.go` — verify stats fields are present and correctly typed in driver and constructor responses
- [ ] T031 [US2] Update frontend tests in `frontend/tests/standings/StandingsPage.test.tsx` — verify stats columns render with "0" for zero values (not blank/dash)

**Checkpoint**: Standings tables show full statistics. US1 + US2 together form the complete P1 deliverable.

---

## Phase 5: User Story 3 — Standings Progression Charts (Priority: P2)

**Goal**: Interactive line charts showing cumulative points per race for drivers and constructors

**Independent Test**: Navigate to `/standings` and toggle to chart view — line chart renders with one line per driver, team-colored, with tooltips on hover.

### Implementation

- [ ] T032 [US3] Add `GetDriverProgression(ctx, season)` and `GetConstructorProgression(ctx, season)` methods to `backend/internal/api/standings/service.go` — query all championship snapshots for the season ordered by session_key, group by driver/team, return per-round points arrays
- [ ] T033 [US3] Add progression DTOs to `backend/internal/api/standings/dto.go` — `ProgressionResponse` with `rounds` array and `drivers`/`teams` array each containing `points_by_round`
- [ ] T034 [US3] Add `GetDriversProgression` and `GetConstructorsProgression` handlers in `backend/internal/api/standings/handler.go`
- [ ] T035 [US3] Register new routes in `backend/internal/api/router.go`: `GET /api/v1/standings/drivers/progression` and `GET /api/v1/standings/constructors/progression`
- [ ] T036 [P] [US3] Add `fetchDriverProgression(year)` and `fetchConstructorProgression(year)` to `frontend/src/features/standings/standingsApi.ts`
- [ ] T037 [US3] Create `frontend/src/features/standings/ProgressionChart.tsx` — recharts `<LineChart>` with `<ResponsiveContainer>`, one `<Line>` per competitor colored by `team_color`, `<Tooltip>` showing name + race + points, `<XAxis>` with round names, `<YAxis>` with points
- [ ] T038 [US3] Update `frontend/src/features/standings/StandingsPage.tsx` — add table/chart toggle, render `<ProgressionChart>` in chart mode for both drivers and constructors tabs
- [ ] T039 [P] [US3] Create `frontend/tests/standings/ProgressionChart.test.tsx` — test chart renders with mock progression data, tooltip interaction, empty state (single round)

**Checkpoint**: Users can toggle between table and chart views. Progression chart shows championship narrative visually.

---

## Phase 6: User Story 4 — Historical Season Selector (Priority: P2)

**Goal**: Year picker allows browsing standings for any season 2023–current

**Independent Test**: Navigate to `/standings`, select 2024 from year picker — tables and charts update to show 2024 data.

### Implementation

- [ ] T040 [US4] Ensure backend standings endpoints already support `year` query parameter (verify from T019 — range 2023–current)
- [ ] T041 [US4] Add on-demand backfill logic in `backend/internal/api/standings/service.go` — when a season has no cached data, trigger synchronous ingestion from OpenF1 (with 30s timeout), cache results, return them; if timeout or no data, return empty response with message
- [ ] T042 [P] [US4] Create `frontend/src/features/standings/YearPicker.tsx` — select/dropdown component showing years 2023–current, defaults to current year, emits `onYearChange(year)` callback
- [ ] T043 [US4] Update `frontend/src/features/standings/StandingsPage.tsx` — integrate `<YearPicker>`, pass selected year to all API calls (standings, progression), re-fetch on year change
- [ ] T044 [US4] Update `frontend/src/features/standings/standingsApi.ts` — ensure all fetch functions accept and pass `year` parameter
- [ ] T045 [P] [US4] Update `frontend/tests/standings/StandingsPage.test.tsx` — test year picker renders, changing year triggers data reload

**Checkpoint**: Historical seasons are accessible. Combined with US3, users can see progression for any season 2023+.

---

## Phase 7: User Story 5 — Head-to-Head Comparisons (Priority: P3)

**Goal**: Select two drivers or constructors for side-by-side stats and overlay progression chart

**Independent Test**: On `/standings`, select two drivers — comparison panel shows stats deltas and two-line progression overlay.

### Implementation

- [ ] T046 [US5] Add `GetDriverComparison(ctx, season, driver1, driver2)` and `GetConstructorComparison(ctx, season, team1, team2)` methods to `backend/internal/api/standings/service.go` — fetch both competitors' stats and progression, compute deltas
- [ ] T047 [US5] Add comparison DTOs to `backend/internal/api/standings/dto.go` — `ComparisonResponse` with `driver1`/`driver2` (or `team1`/`team2`) stats, `deltas`, and `progression` overlay data
- [ ] T048 [US5] Add `GetDriversCompare` and `GetConstructorsCompare` handlers in `backend/internal/api/standings/handler.go` — validate `driver1`/`driver2` or `team1`/`team2` params, return 400 for missing/same values, 404 for not found
- [ ] T049 [US5] Register new routes in `backend/internal/api/router.go`: `GET /api/v1/standings/drivers/compare` and `GET /api/v1/standings/constructors/compare`
- [ ] T050 [P] [US5] Add `fetchDriverComparison(year, driver1, driver2)` and `fetchConstructorComparison(year, team1, team2)` to `frontend/src/features/standings/standingsApi.ts`
- [ ] T051 [US5] Create `frontend/src/features/standings/ComparisonPanel.tsx` — side-by-side stats display with delta badges ("+15 pts", "+2 wins"), plus a `<LineChart>` overlay with exactly 2 lines
- [ ] T052 [US5] Update `frontend/src/features/standings/StandingsPage.tsx` — add comparison selection UI (checkboxes or click-to-select on table rows), show `<ComparisonPanel>` when two competitors selected, clear on year change if competitors don't exist in new season
- [ ] T053 [P] [US5] Create `frontend/tests/standings/ComparisonPanel.test.tsx` — test renders stats, deltas, overlay chart; test clearing on season change

**Checkpoint**: Power users can compare championship rivals directly. All P1+P2 features remain working.

---

## Phase 8: User Story 6 — Constructor Driver Breakdown (Priority: P3)

**Goal**: Expand a constructor row to see individual driver contributions

**Independent Test**: On Constructors tab, click a team row — inline detail shows each driver's points, wins, podiums. Points sum to team total.

### Implementation

- [ ] T054 [US6] Add `GetConstructorDriverBreakdown(ctx, season, teamName)` method to `backend/internal/api/standings/service.go` — query driver standings for that team's drivers, compute `points_percentage`
- [ ] T055 [US6] Add breakdown DTO to `backend/internal/api/standings/dto.go` — `ConstructorBreakdownResponse` with `team_name`, `team_points`, `drivers` array
- [ ] T056 [US6] Add `GetConstructorDrivers` handler in `backend/internal/api/standings/handler.go` — parse team path param, validate year
- [ ] T057 [US6] Register new route in `backend/internal/api/router.go`: `GET /api/v1/standings/constructors/{team}/drivers`
- [ ] T058 [P] [US6] Add `fetchConstructorDriverBreakdown(year, teamName)` to `frontend/src/features/standings/standingsApi.ts`
- [ ] T059 [US6] Create `frontend/src/features/standings/ConstructorBreakdown.tsx` — expandable inline component showing driver rows with position, points, wins, podiums, and percentage bar
- [ ] T060 [US6] Update `frontend/src/features/standings/StandingsPage.tsx` — make constructor table rows expandable (click to toggle), render `<ConstructorBreakdown>` inline when expanded
- [ ] T061 [P] [US6] Create `frontend/tests/standings/ConstructorBreakdown.test.tsx` — test expand/collapse, driver points sum to team total, percentage display

**Checkpoint**: Constructor drill-down complete. All 6 user stories are functional.

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Integration testing, observability, and deployment readiness

- [ ] T062 [P] Verify structured JSON logging in `ChampionshipIngester` — log session_key, data type, row count, duration on each ingestion; log errors with context
- [ ] T063 [P] Validate no direct frontend calls to OpenF1 — update `frontend/tests/integration/network_boundary.test.ts` to cover new standings API client functions
- [ ] T064 Run backfill for 2026 completed sessions: `go run cmd/backfill/main.go --season=2026 --championship`
- [ ] T065 [P] Run backfill for historical seasons: `--season=2025 --championship`, `--season=2024 --championship`, `--season=2023 --championship`
- [ ] T066 Verify all backend tests pass: `cd backend && go test ./...`
- [ ] T067 Verify all frontend tests pass: `cd frontend && npx vitest run`
- [ ] T068 [P] Verify lint passes: `cd backend && golangci-lint run ./...` and `cd frontend && npx tsc --noEmit`

**Checkpoint**: Feature is complete, tested, and deployment-ready.

---

## Dependencies

```
Phase 1 (T001–T007) → Phase 2 (T008–T016) → Phase 3 (T017–T024) → Phase 4 (T025–T031)
                                                                         ↓
                                                          Phase 5 (T032–T039) ←──┘
                                                                         ↓
                                                          Phase 6 (T040–T045)
                                                                         ↓
                                                          Phase 7 (T046–T053)
                                                                         ↓
                                                          Phase 8 (T054–T061)
                                                                         ↓
                                                          Phase 9 (T062–T068)
```

**Story independence notes**:
- US1 and US2 share foundational ingestion (Phase 2) but their API/frontend work (Phases 3–4) can overlap
- US3 (progression) depends on US1 data being in Cosmos
- US4 (year picker) is orthogonal to US3 — can be done in parallel after US1
- US5 (comparison) depends on US2 stats + US3 progression being available
- US6 (breakdown) depends on US2 stats being available

**Parallel opportunities within phases**:
- Phase 1: T003, T004, T005, T007 can all run in parallel
- Phase 2: T009, T014, T015, T016 can run in parallel once T008 is started
- Phase 3: T020, T021 can run in parallel
- Phase 5: T036, T039 can run in parallel
- Phase 9: T062, T063, T065, T068 can all run in parallel

---

## Implementation Strategy

**MVP scope**: Phases 1–4 (US1 + US2) — real data with expanded stats. This alone replaces the broken Hyprace integration and delivers immediate value.

**Incremental delivery**:
1. Phase 1–2: Backend infrastructure (Hyprace removal + OpenF1 ingestion). Deploy backend, run backfill.
2. Phase 3–4: P1 stories (real standings tables with full stats). Deploy frontend.
3. Phase 5–6: P2 stories (progression charts + year picker). Deploy both.
4. Phase 7–8: P3 stories (comparison + breakdown). Deploy both.
5. Phase 9: Polish and verify production readiness.

**Total tasks**: 68  
**Tasks per story**: US1: 8, US2: 7, US3: 8, US4: 6, US5: 8, US6: 8  
**Setup/foundational**: 16, Polish: 7
