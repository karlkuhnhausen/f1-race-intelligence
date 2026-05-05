# Tasks: Stable Identity Migration

**Input**: Design documents from `/specs/008-stable-identity-migration/`  
**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, quickstart.md ✅

**Organization**: Tasks grouped by user story. Phases 1-4 are complete (domain types, storage layer, API query paths). Remaining work is Phase 5 (backfill CLI) and Phase 6 (end-to-end validation).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US3, US4)

---

## Phase 1: Setup — Domain Types (COMPLETE)

**Purpose**: MeetingIndex type and BuildMeetingIndex function

- [x] T001 Create MeetingIndex, MeetingIndexEntry, MeetingForIndex types in backend/internal/domain/meeting_index.go
- [x] T002 Implement BuildMeetingIndex function with cancelled/testing exclusion in backend/internal/domain/meeting_index.go
- [x] T003 Add MeetingKeyForRound helper method on MeetingIndex in backend/internal/domain/meeting_index.go

---

## Phase 2: Storage Layer — Fields & Query Methods (COMPLETE)

**Purpose**: Add MeetingKey/SessionKey fields to storage types and meeting_key-based query methods

- [x] T004 [P] Add MeetingKey field to storage.Session and storage.SessionResult types in backend/internal/storage/repository.go
- [x] T005 [P] Add MeetingKey and SessionKey fields to all analysis document types in backend/internal/storage/repository.go
- [x] T006 Implement GetSessionsByMeetingKey query method in backend/internal/storage/cosmos/sessions.go
- [x] T007 Implement GetSessionResultsByMeetingKey query method in backend/internal/storage/cosmos/sessions.go
- [x] T008 Implement GetSessionAnalysisByMeetingKey query method in backend/internal/storage/cosmos/analysis.go

---

## Phase 3: API — Rounds Service meeting_key-first Query (COMPLETE)

**Purpose**: Rounds service resolves meeting_key via MeetingIndex, queries by meeting_key with round fallback

- [x] T009 [US1] Build MeetingIndex from calendar data in rounds service in backend/internal/api/rounds/service.go
- [x] T010 [US1] Implement meeting_key-first query for sessions with round fallback in backend/internal/api/rounds/service.go
- [x] T011 [US1] Implement meeting_key-first query for session results with round fallback in backend/internal/api/rounds/service.go
- [x] T012 [US2] Verify round-based fallback returns pre-migration documents correctly in backend/internal/api/rounds/service.go

---

## Phase 4: API — Analysis Service meeting_key-first Query (COMPLETE)

**Purpose**: Analysis service resolves meeting_key via MeetingIndex, queries by meeting_key with round fallback

- [x] T013 [US1] Build MeetingIndex from calendar data in analysis service in backend/internal/api/analysis/service.go
- [x] T014 [US1] Implement meeting_key-first query for analysis data with round fallback in backend/internal/api/analysis/service.go
- [x] T015 [US2] Verify round-based fallback returns pre-migration analysis documents in backend/internal/api/analysis/service.go

---

## Phase 5: User Story 3 — Backfill CLI `--stamp-meeting-keys` Mode (Priority: P3)

**Goal**: Operator runs backfill CLI to retroactively populate meeting_key and session_key on all pre-migration documents in Cosmos DB.

**Independent Test**: Run `go run ./cmd/backfill --season=2026 --stamp-meeting-keys --dry-run` and verify it logs correct meeting_key resolutions for all unstamped documents without writing.

### Implementation for User Story 3

- [x] T016 [US3] Add `--stamp-meeting-keys` flag to CLI flag parsing in backend/cmd/backfill/main.go
- [x] T017 [US3] Implement stampMeetingKeys function: fetch all RaceMeetings via GetMeetingsBySeason, build MeetingIndex in backend/cmd/backfill/main.go
- [x] T018 [US3] Query all Session documents with meeting_key=0 for the season and stamp each with resolved meeting_key via MeetingIndex in backend/cmd/backfill/main.go
- [x] T019 [US3] Query all SessionResult documents with meeting_key=0 for the season and stamp each with resolved meeting_key via MeetingIndex in backend/cmd/backfill/main.go
- [x] T020 [US3] Query all analysis documents (positions, intervals, stints, pits, overtakes) with meeting_key=0 for the season in backend/cmd/backfill/main.go
- [x] T021 [US3] For each unstamped analysis document, resolve session_key by querying sessions container for matching session (season+round+session_type) in backend/cmd/backfill/main.go
- [x] T022 [US3] Upsert stamped documents back to Cosmos DB (skip if meeting_key already > 0 for idempotency) in backend/cmd/backfill/main.go
- [x] T023 [US3] Support --dry-run: log what would be stamped without writing, including resolved meeting_key and session_key values in backend/cmd/backfill/main.go
- [x] T024 [US3] Add structured slog warnings for documents where meeting_key cannot be resolved (round not in MeetingIndex) in backend/cmd/backfill/main.go
- [x] T025 [US3] Add completion summary log (stamped count, skipped count, failed count, dry_run flag) in backend/cmd/backfill/main.go
- [x] T026 [US3] Verify `go build ./...` and `go test ./...` pass with the new CLI mode in backend/

---

## Phase 6: User Story 4 — End-to-End Validation (Priority: P4)

**Goal**: Validate that after backfill, all session queries resolve via meeting_key and return correct results.

**Independent Test**: Run backfill in dry-run mode, verify logs show correct meeting_key for every round. Then query rounds API for a post-cancellation round and confirm correct data.

### Implementation for User Story 4

- [x] T027 [US4] Run backfill --stamp-meeting-keys --dry-run against live Cosmos DB and verify all rounds resolve to expected meeting_keys in backend/cmd/backfill/main.go
- [x] T028 [US4] Run backfill --stamp-meeting-keys (actual) against live Cosmos DB and verify completion log shows 0 failures
- [x] T029 [US4] Query rounds API for multiple rounds and verify session data is returned via meeting_key path (check structured logs for query method used)
- [x] T030 [US4] Verify analysis API returns correct data for stamped sessions (positions, intervals, stints, pits, overtakes present)
- [x] T031 [US4] Verify idempotency: re-run backfill --stamp-meeting-keys and confirm 0 documents updated (all skipped)
- [x] T032 [US4] Run full backend test suite: `cd backend && go test ./...` passes with no regressions

---

## Dependencies

```
Phase 1 (T001-T003) ──► Phase 2 (T004-T008) ──► Phase 3 (T009-T012)
                                                       │
                                                       ▼
                                                 Phase 4 (T013-T015)
                                                       │
                                                       ▼
                                                 Phase 5 (T016-T026)
                                                       │
                                                       ▼
                                                 Phase 6 (T027-T032)
```

Within Phase 5: T016 → T017 → (T018, T019, T020 can be parallel) → T021 → T022 → T023 → T024 → T025 → T026

## Parallel Execution Opportunities

**Phase 5** (within the story):
- T018, T019, T020 can be implemented in parallel (different document types, same pattern)
- T023, T024, T025 are independent logging concerns (can be woven in during T018-T022 implementation)

**Phase 6** (validation):
- T029 and T030 can run in parallel (different API endpoints)

## Implementation Strategy

1. **MVP**: Phase 5 complete → operator can stamp all existing documents
2. **Validation**: Phase 6 confirms correctness end-to-end
3. **Incremental delivery**: Each task within Phase 5 builds on the previous, but the entire stampMeetingKeys function can be implemented as one cohesive unit if preferred
4. **No frontend changes**: This is entirely backend CLI work
5. **Existing patterns**: Follow the established backfill patterns in `main.go` (analysis mode, championship mode) for consistency
