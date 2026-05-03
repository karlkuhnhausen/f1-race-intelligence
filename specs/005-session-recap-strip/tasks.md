# Tasks: Session Recap Strip

**Input**: Design documents from `/specs/005-session-recap-strip/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/rounds-recap-api.md

**Tests**: Contract, backend unit, and frontend unit tests are included as specified in plan.md §1.4.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story. The six user stories are implemented in priority order (P1 → P2 → P3).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: User story label — US1 through US6 per spec.md; setup and foundational tasks carry no story label
- Every task includes exact file paths

## Implementation Order

`contract tests RED → storage types → cosmos impl → DTO extension → ingest race_control → poller extension → API service extension → frontend types → card components → strip container + page insert → backfill CLI → lazy fill wiring → tests GREEN`

---

## Phase 1: Setup

**Purpose**: Create the one new directory and file entry-point that does not exist yet.

- [x] T001 Create `backend/cmd/backfill/` directory and stub `backend/cmd/backfill/main.go` as a `package main` placeholder (empty `func main(){}`)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Storage-layer types, repository interface extension, DTO extension, and contract tests written RED — all of which every user story phase depends on.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [x] T002 Add `RaceControlSummary` and `NotableEvent` storage structs; add `RaceControlSummary *RaceControlSummary` and `FastestLapTimeSeconds *float64` fields to `Session` struct; add `GetFinalizedSessions(ctx context.Context, season int) ([]Session, error)` method to `SessionRepository` interface in `backend/internal/storage/repository.go`
- [x] T003 Implement `GetFinalizedSessions` in `backend/internal/storage/cosmos/sessions.go` — query `SELECT * FROM c WHERE c.season = @season AND c.type = 'session' AND c.finalized = true` using the standard partition-key pager pattern already present in the file
- [x] T004 [P] Add `NotableEventDTO` and `SessionRecapDTO` structs (all fields `omitempty`; race/sprint, qualifying, and practice field groups per plan.md §1.1); add `RecapSummary *SessionRecapDTO` to `SessionDetailDTO` in `backend/internal/api/rounds/dto.go`
- [x] T005 [P] Extend `backend/tests/contract/rounds_contract_test.go` with three RED tests: `TestRoundDetail_RecapSummary_CompletedRace` (response includes `recap_summary` with winner fields), `TestRoundDetail_RecapSummary_UpcomingSession` (`recap_summary` absent), `TestRoundDetail_RecapSummary_NilHydrator_Degrades` (nil hydrator → recap omits events but returns winner) — confirm these fail before implementation

**Checkpoint**: Storage types compile; DTO compiles; contract tests are present and failing; all later phases can proceed.

---

## Phase 3: User Story 1 — View Race Recap Card (Priority: P1) 🎯 MVP

**Goal**: A fan viewing a completed round detail page sees a Race recap card at the top showing winner, gap to P2, fastest lap, total laps, and the highest-priority race-control event.

**Independent Test**: Navigate to a 2026 completed round detail page. A Race card appears in the recap strip with winner name and team color, gap to P2, fastest lap holder and time, total laps, and a safety car / red flag event line (if any occurred). No card appears for sessions that have not yet ended.

### Tests for User Story 1

> **Write these first, run `go test ./...` and `npx vitest run` to confirm RED before implementing T010–T014.**

- [x] T006 [P] [US1] Write backend unit tests for `SummarizeRaceControl` in `backend/tests/unit/race_control_test.go` — cases: single RED flag, safety car deduplication (two messages same lap → count 1), safety car two distinct laps (count 2), VSC ignores ending messages, empty message slice returns zero-value summary, priority red flag over SC
- [x] T007 [P] [US1] Write backend unit tests for `deriveRecapSummary` race/sprint path in `backend/tests/unit/rounds_recap_test.go` — cases: P1 result maps to winner fields, no classified finishers omits winner gracefully, `FastestLap == true` result maps to fastest lap holder, `FastestLapTimeSeconds` propagated from `sess.FastestLapTimeSeconds`
- [x] T008 [P] [US1] Write frontend unit tests for `RaceRecapCard` in `frontend/tests/rounds/RecapCards.test.tsx` — cases: renders winner name and team, renders gap to P2, omits fastest lap row when absent, renders top_event safety car line, renders top_event red flag line

### Implementation for User Story 1

- [x] T009 [US1] Create `backend/internal/ingest/race_control.go` — implement `FetchRaceControlMsgs(ctx, client, sessionKey)` calling `GET https://api.openf1.org/v1/race_control?session_key={key}`; implement `SummarizeRaceControl(msgs)` with deduplication by activation type + lap_number per plan.md §1.2 (red flag: flag=="RED"; safety car: message starts with "SAFETY CAR DEPLOYED"; VSC: "VIRTUAL SAFETY CAR DEPLOYED"; investigation: category=="Other" && message contains "UNDER INVESTIGATION"); define `RaceControlHydrator` struct with `NewRaceControlHydrator(repo, logger)` constructor and `Hydrate(ctx, sess)` method (fetch → summarize → upsert → return)
- [x] T010 [US1] Extend finalization block in `backend/internal/ingest/session_poller.go` — after `fetchAndUpsertResults` succeeds: (1) sleep 500ms, call `FetchRaceControlMsgs`, call `SummarizeRaceControl`, assign to `sess.RaceControlSummary`; (2) derive `FastestLapTimeSeconds` from the laps slice already in memory (find lap where driver == `DeriveFastestLap` result, minimum non-nil `LapDuration`); store both in the same `UpsertSession` call that sets `sess.Finalized = true`; do NOT bump `SessionSchemaVersion`
- [x] T011 [US1] Extend `backend/internal/api/rounds/service.go` — define `RaceControlHydrator` interface (`Hydrate(ctx, sess) (*storage.RaceControlSummary, error)`); add `rcHydrator RaceControlHydrator` and `logger *slog.Logger` fields to `Service` struct; add `NewServiceWithHydrator(sessionRepo, calendarRepo, hydrator, logger) *Service` constructor; add `deriveTopEvent` helper (priority: red_flag > safety_car > vsc > investigation; nil if no events); add `deriveRecapSummary(sess, results) *SessionRecapDTO` function implementing the race/sprint path per plan.md §1.2; call `deriveRecapSummary` inside the per-session loop when `status == statusCompleted`
- [x] T012 [P] [US1] Add `NotableEvent` and `SessionRecapSummary` TypeScript interfaces matching the contract in `contracts/rounds-recap-api.md`; add `recap_summary?: SessionRecapSummary` to the `SessionDetail` interface in `frontend/src/features/rounds/roundApi.ts`
- [x] T013 [P] [US1] Create `frontend/src/features/rounds/RaceRecapCard.tsx` — props: `{ session: SessionDetail }` (parent guarantees `recap_summary` non-null); render session label (`session.session_name`), winner row with team-color left border using `getTeamColor`, gap to P2 (omit row if absent), fastest lap row with `LapTimeDisplay` (omit if absent), `{total_laps} laps` (omit if 0), top event row using emoji labels per plan.md §1.3 (omit if no top_event)

**Checkpoint**: Backend compiles; `SummarizeRaceControl` and `deriveRecapSummary` race-path unit tests pass; `RaceRecapCard` renders correctly in isolation; contract tests still RED (hydrator not wired to main yet).

---

## Phase 4: User Story 5 — Backfill Existing 2026 Sessions (Priority: P1)

**Goal**: A one-shot CLI populates `RaceControlSummary` for all already-finalized 2026 sessions that are missing it, with idempotency and fault tolerance.

**Independent Test**: Run `backfill --season=2026 --dry-run --rate-limit-ms=1000` against Cosmos. Verify structured JSON log lines per session (skipped/would-update) and a summary log line at completion. No writes occur in dry-run mode.

### Implementation for User Story 5

- [x] T014 [US5] Implement `backend/cmd/backfill/main.go` — flags: `--season` (required int), `--dry-run` (bool), `--rate-limit-ms` (default 1000); wiring: read `COSMOS_ENDPOINT` from env, create Cosmos client via same path as `backend/internal/config/`; call `repo.GetFinalizedSessions(ctx, season)`; loop: skip if `sess.RaceControlSummary != nil` (idempotent), call `FetchRaceControlMsgs`, if error/empty log WARN + continue, call `SummarizeRaceControl`, if not dry-run patch and `repo.UpsertSession`, sleep `rate-limit-ms`; emit `log/slog` structured JSON per session (`{"level":"INFO","msg":"backfill: updated","session_id":"...","outcome":"updated|skipped|failed"}`); emit summary line at end (`{"level":"INFO","msg":"backfill: complete","updated":N,"skipped":N,"failed":N}`)

**Checkpoint**: Backfill binary compiles and runs with `--dry-run`; emits correct per-session and summary JSON log lines; skips sessions that already have `RaceControlSummary`.

---

## Phase 5: User Story 2 — View Qualifying Recap Card (Priority: P2)

**Goal**: A fan viewing a round with a completed qualifying session sees a Qualifying recap card showing pole sitter, pole time, gap to P2, Q1/Q2 cutoff times (standard format), and red flag count.

**Independent Test**: Navigate to a round with a completed qualifying session. A Qualifying card appears showing pole sitter with team color border, pole time, gap to P2, Q1 and Q2 cutoff times (omitted on sprint qualifying format), and red flag count (omitted if none).

### Tests for User Story 2

- [x] T015 [P] [US2] Extend `backend/tests/unit/rounds_recap_test.go` with qualifying/sprint qualifying tests — cases: pole and Q1/Q2 cutoff derivation (last P15 Q1 time, last P10 Q2 time), sprint qualifying format omits Q1/Q2 cutoffs, gap_to_p2 formatted as "+X.XXX"
- [x] T016 [P] [US2] Extend `frontend/tests/rounds/RecapCards.test.tsx` with `QualifyingRecapCard` tests — cases: renders pole sitter and pole time, renders Q1/Q2 cutoffs, omits cutoff rows when absent, renders red flag count, omits red flag row when zero

### Implementation for User Story 2

- [x] T017 [US2] Add qualifying/sprint qualifying branch to `deriveRecapSummary` in `backend/internal/api/rounds/service.go` — pole sitter from P1 result; pole time from Q3Time (fallback Q2Time, Q1Time); gap to P2 as formatted delta string; Q1 cutoff from last result with non-nil Q1Time and nil Q2Time; Q2 cutoff from last result with non-nil Q2Time and nil Q3Time; race-control fields from `sess.RaceControlSummary`
- [x] T018 [P] [US2] Create `frontend/src/features/rounds/QualifyingRecapCard.tsx` — props: `{ session: SessionDetail }`; render session label, pole sitter with team-color border, pole time via `LapTimeDisplay`, gap to P2, Q1 cutoff row (omit if nil), Q2 cutoff row (omit if nil), red flag count row (omit if 0 or absent)

**Checkpoint**: Qualifying unit tests pass; `QualifyingRecapCard` renders correctly in isolation; backend qualifying recap fields appear in API response for completed qualifying sessions.

---

## Phase 6: User Story 4 — Strip Ordering and Mobile/Desktop Layout (Priority: P2)

**Goal**: All recap cards appear in chronological session order in a responsive container — vertical stack on mobile (≤768px), horizontal scrollable row on desktop (>768px).

**Independent Test**: Load a round detail page with all sessions completed at 375px and 1280px viewport widths. Cards are ordered FP1 → FP2 → FP3 → Qualifying → Race. Mobile: full-width vertical stack. Desktop: horizontal row, overflow scrolls if needed. No cards appear when no completed sessions with recap data exist.

### Tests for User Story 4

- [x] T019 [P] [US4] Create `frontend/tests/rounds/SessionRecapStrip.test.tsx` — cases: renders null when no completed sessions, renders one card per completed session with recap_summary, cards are in chronological order by `date_start_utc`, dispatches correct card type per `session_type` (race/sprint → RaceRecapCard, qualifying/sprint_qualifying → QualifyingRecapCard, practice* → PracticeRecapCard)

### Implementation for User Story 4

- [x] T020 [US4] Create `frontend/src/features/rounds/SessionRecapStrip.tsx` — props: `{ sessions: SessionDetail[] }`; filter to sessions where `status === 'completed' && recap_summary != null`; sort ascending by `date_start_utc`; render strip container with responsive CSS (`flex flex-col md:flex-row md:overflow-x-auto`; cards full-width on mobile, fixed ~280px width on desktop); dispatch per session type per plan.md §1.3; return null if filtered list is empty
- [x] T021 [US4] Insert `<SessionRecapStrip sessions={data.sessions} />` between the round header section and the session cards list in `frontend/src/features/rounds/RoundDetailPage.tsx`

**Checkpoint**: Strip renders all card types in correct order; responsive layout verified at target breakpoints; no strip rendered when no completed sessions.

---

## Phase 7: User Story 3 — View Practice Recap Cards (Priority: P3)

**Goal**: Each completed practice session (FP1, FP2, FP3) renders a card showing the session-best driver with team color, best lap time, total laps run, and red flag count.

**Independent Test**: Navigate to a round with at least one completed practice session. A card per completed practice appears (no placeholders for absent sessions), each showing session label, session-best driver with team color border, best lap time, total laps, and red flag count (omitted if none).

### Tests for User Story 3

- [x] T022 [P] [US3] Extend `backend/tests/unit/rounds_recap_test.go` with practice tests — cases: P1 result maps to best driver and best lap time, total laps is sum of `NumberOfLaps` across all results, red flag count propagated from `RaceControlSummary`, no red flag row when count is zero
- [x] T023 [P] [US3] Extend `frontend/tests/rounds/RecapCards.test.tsx` with `PracticeRecapCard` tests — cases: renders session label, renders best driver and team, renders best lap time, renders total laps, omits red flag row when zero

### Implementation for User Story 3

- [x] T024 [US3] Add practice branch to `deriveRecapSummary` in `backend/internal/api/rounds/service.go` — best driver from P1 result (lowest BestLapTime, already sorted); best lap time from P1 result; total laps as `sum(r.NumberOfLaps for r in results)`; red flag count from `sess.RaceControlSummary`
- [x] T025 [P] [US3] Create `frontend/src/features/rounds/PracticeRecapCard.tsx` — props: `{ session: SessionDetail }`; render session label, best driver with team-color left border, best lap time via `LapTimeDisplay`, `{total_laps} laps`, red flag count row (omit if 0 or absent)

**Checkpoint**: Practice unit tests pass; `PracticeRecapCard` renders correctly; all three card types visible in the strip for a round with all sessions completed.

---

## Phase 8: User Story 6 — Lazy-On-Read Gap Fill (Priority: P3)

**Goal**: Any session that finalized but is missing `RaceControlSummary` is hydrated on first read, persisted to Cosmos, and included in the response — with graceful degradation if the fetch fails.

**Independent Test**: Manually remove `race_control_summary` from one session document in Cosmos. Load the round detail page. The recap card appears with correct event data. On a second load, no OpenF1 call is made (data was persisted). If OpenF1 returns an error, the card renders without the event line rather than failing.

### Implementation for User Story 6

- [x] T026 [US6] Add lazy fill block in `GetRoundDetail` per-session loop in `backend/internal/api/rounds/service.go` — before calling `deriveRecapSummary`, if `status == statusCompleted && sess.RaceControlSummary == nil && s.rcHydrator != nil`: call `s.rcHydrator.Hydrate(ctx, sess)`; on success set `sess.RaceControlSummary = summary`; on error log `s.logger.Warn("lazy race control fill failed — recap rendered without events", "session_id", sess.ID, "error", err)` and continue (graceful degradation)
- [x] T027 [US6] Wire `NewServiceWithHydrator` with a real `*ingest.RaceControlHydrator` in `backend/cmd/api/main.go`, replacing the existing `NewService` call; inject the Cosmos session repository and the `slog` logger already present in main

**Checkpoint**: Contract tests pass GREEN (hydrator wired); lazy fill hydrates missing summaries on read and persists them; failure path returns partial response without error; `go test ./...` passes.

---

## Final Phase: Polish & Cross-Cutting Concerns

**Purpose**: Verify all tests green, lint clean, backfill operational, and `omitempty` behaviour confirmed end-to-end.

- [x] T028 [P] Verify all backend tests pass: `cd backend && go test ./...` — confirm contract, unit, and any integration tests all green
- [x] T029 [P] Verify all frontend tests pass: `cd frontend && npx vitest run` — confirm `SessionRecapStrip`, `RecapCards`, and `RoundDetailPage` tests all green
- [x] T030 [P] Run `export PATH="$HOME/go/bin:$PATH" && cd backend && golangci-lint run ./...` and resolve any lint issues in new or modified files
- [x] T031 Validate `omitempty` behaviour end-to-end: call `GET /api/v1/rounds/{round}?year=2026` for a completed round and confirm zero-value race-control fields (`red_flag_count: 0`, `top_event: null`) are absent from the JSON response body
- [x] T032 Run post-deploy backfill dry-run: `./backfill --season=2026 --dry-run --rate-limit-ms=1000` and confirm one structured JSON log line per finalized session plus the summary line; then run without `--dry-run` in production and spot-check at least 3 completed round detail pages for correct recap cards (SC-001)

---

## Dependency Graph

```
T001 (setup)
  └─► T002 (storage types)
        ├─► T003 (cosmos GetFinalizedSessions)
        ├─► T004 (DTO extension)          [P with T005]
        └─► T005 (contract tests RED)     [P with T004]
              ├─► T006 (unit: SummarizeRaceControl)   [P]
              ├─► T007 (unit: deriveRecapSummary race) [P]
              ├─► T008 (unit: RaceRecapCard)           [P]
              ├─► T009 (race_control.go)
              │     └─► T010 (poller extension)
              ├─► T011 (service: interface + race derive)
              │     └─► T017 (service: qualifying derive)
              │     └─► T024 (service: practice derive)
              │     └─► T026 (lazy fill logic)
              │           └─► T027 (wire hydrator in main.go)
              ├─► T012 (frontend types)   [P with T013]
              ├─► T013 (RaceRecapCard)    [P with T012]
              ├─► T014 (backfill CLI)     [needs T003, T009]
              ├─► T018 (QualifyingRecapCard) [P; needs T012]
              ├─► T020 (SessionRecapStrip)   [needs T013, T018, T025]
              │     └─► T021 (RoundDetailPage insert)
              └─► T025 (PracticeRecapCard)   [P; needs T012]
```

## Parallel Execution Examples

**Within Phase 3 (US1)**:
- T006, T007, T008 (test stubs) can be written in parallel
- T012 (frontend types) and T013 (RaceRecapCard) can be written in parallel with T009 (race_control.go)
- T010 (poller) must follow T009; T011 (service) must follow T004

**Within Phase 5 (US2)**:
- T015 (backend tests), T016 (frontend tests), T018 (QualifyingRecapCard) can all be written in parallel with T017 (service qualifying)

**Within Phase 7 (US3)**:
- T022 (backend tests), T023 (frontend tests), T025 (PracticeRecapCard) can all be written in parallel with T024 (service practice)

**Final Phase**:
- T028 (backend tests), T029 (frontend tests), T030 (lint) can run in parallel after T027 completes

## Implementation Strategy

**MVP scope**: Complete Phases 1–4 (T001–T014). This delivers:
- Race recap cards for newly-finalized sessions (via poller)
- Backfill for all pre-existing 2026 sessions
- The `RaceRecapCard` component ready to display once the strip container (Phase 6) is wired in

**Increment 2**: Complete Phase 5 + Phase 6 (T015–T021). This adds qualifying cards and makes all cards visible on the page with correct layout.

**Increment 3**: Complete Phases 7–8 (T022–T027). Adds practice cards and the resilience lazy-fill path.

**Total tasks**: 32 (T001–T032)
**Tasks per story**: US1: 8 · US2: 4 · US3: 4 · US4: 3 · US5: 1 · US6: 2 · Foundational: 4 · Setup: 1 · Polish: 5
**Parallel opportunities**: 12 tasks marked [P]
