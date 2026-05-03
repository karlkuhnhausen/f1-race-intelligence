# Implementation Plan: Session Recap Strip

**Branch**: `005-session-recap-strip` | **Date**: 2026-05-02 | **Spec**: [spec.md](spec.md)  
**Input**: Feature specification from `/specs/005-session-recap-strip/spec.md`

---

## Summary

Add a session recap strip to the round detail page. Each completed session renders a summary card — Race, Qualifying, or Practice — showing key outcomes at a glance. The backend extends the existing round detail endpoint with a pre-computed `recap_summary` per session. Race-control data (flags, safety cars, VSC) is fetched from OpenF1 `/v1/race_control` once at session finalization, persisted into the existing Cosmos session document, and served from Cosmos at read time. Fastest-lap data reuses the existing Feature 003 laps-fetch path. A one-shot backfill CLI populates race-control summaries for all pre-existing 2026 sessions. Lazy-on-read gap fill handles edge cases.

---

## Technical Context

**Language/Version**: Go 1.25 (backend), TypeScript 5.6 / React 18 (frontend)  
**Primary Dependencies**: Chi v5 router, Azure Cosmos DB SDK for Go, Vitest 4.1 — no new dependencies  
**Storage**: Azure Cosmos DB serverless — `sessions` container, `season` partition key, existing schema extended (additive fields only)  
**Testing**: `go test ./...` (backend), `npx vitest run` (frontend, pool: "threads")  
**Target Platform**: AKS 1.33 (existing deployment)  
**Performance Goals**: No additional user-perceived latency vs. baseline round detail load; lazy fill completes within existing request timeout (30s)  
**Constraints**: No new Helm/Bicep resources; no new secrets; no pass-through to OpenF1 at request time (except lazy fill which persists before responding); OpenF1 rate limit ≤1 req/s respected by both poller and backfill  
**Scale/Scope**: ~20 sessions per 2026 season requiring backfill; ~7 sessions per round detail request (worst case, all completed)

---

## Constitution Check

| Gate | Status | Notes |
|------|--------|-------|
| **Stack gate** | ✅ PASS | Go backend, React frontend, Cosmos DB serverless, AKS — no new platform components |
| **Architecture gate** | ✅ PASS | Frontend calls only the existing `/api/v1/rounds/{round}` backend endpoint; OpenF1 calls are backend-only (ingest poller, backfill CLI, lazy-fill hydrator) |
| **Data gate** | ✅ PASS | Race-control data fetched once at finalization, persisted to Cosmos. Lazy fill also persists before responding. Backfill covers pre-existing sessions. No pass-through at request time. |
| **Security gate** | ✅ PASS | No new secrets. Backfill CLI uses same Managed Identity / Key Vault path as main service. |
| **Network gate** | ✅ PASS | No changes to ingress TLS or egress firewall rules. Backend already has egress to `api.openf1.org`. |
| **Delivery gate** | ✅ PASS | No new Helm/Bicep resources. CI/CD pipeline order unchanged: lint → test → build → push → deploy. Backfill is a manual post-deploy step. |
| **Observability gate** | ✅ PASS | Backfill and lazy-fill emit structured JSON logs (per FR/CA-008). Per-session outcome logged. |
| **Dependency gate** | ✅ PASS | Zero new external dependencies. Race-control data comes from OpenF1 (already allowed). Fastest-lap reuses existing Feature 003 code. |
| **Spec authority gate** | ✅ PASS | All work items trace to FR-001–FR-018 and CA-001–CA-010 in spec.md. |

---

## Project Structure

### Documentation (this feature)

```
specs/005-session-recap-strip/
├── plan.md              ← This file
├── research.md          ← Phase 0 output (resolved)
├── data-model.md        ← Phase 1 output
├── quickstart.md        ← Phase 1 output
├── contracts/
│   └── rounds-recap-api.md  ← Phase 1 output
└── tasks.md             ← Phase 2 output (/speckit.tasks — NOT created here)
```

### Source Code Changes

```
backend/
├── internal/
│   ├── storage/
│   │   └── repository.go          ← EXTEND: add RaceControlSummary, NotableEvent types;
│   │                                          extend Session struct; add GetFinalizedSessions
│   │   cosmos/
│   │   └── sessions.go            ← EXTEND: implement GetFinalizedSessions
│   ├── ingest/
│   │   ├── race_control.go        ← NEW: OpenF1 race_control fetcher, deduplication,
│   │   │                                     RaceControlHydrator
│   │   └── session_poller.go      ← EXTEND: call race_control fetch at finalization;
│   │                                          store FastestLapTimeSeconds
│   └── api/
│       └── rounds/
│           ├── dto.go             ← EXTEND: add SessionRecapDTO, NotableEventDTO;
│           │                                 add RecapSummary to SessionDetailDTO
│           └── service.go         ← EXTEND: derive recap payload per session type;
│                                             inject RaceControlHydrator for lazy fill
└── cmd/
    └── backfill/
        └── main.go                ← NEW: one-shot backfill CLI

frontend/
├── src/
│   └── features/
│       └── rounds/
│           ├── roundApi.ts        ← EXTEND: add recap types to SessionDetail
│           ├── SessionRecapStrip.tsx  ← NEW: responsive strip container
│           ├── RaceRecapCard.tsx      ← NEW: race/sprint card
│           ├── QualifyingRecapCard.tsx ← NEW: qualifying card
│           ├── PracticeRecapCard.tsx   ← NEW: practice card
│           └── RoundDetailPage.tsx    ← EXTEND: insert <SessionRecapStrip>
└── tests/
    └── rounds/
        ├── SessionRecapStrip.test.tsx ← NEW
        └── RecapCards.test.tsx        ← NEW
```

---

## Phase 0: Research

**Status**: Complete. See [research.md](research.md).

All NEEDS CLARIFICATION items resolved:
1. OpenF1 `/v1/race_control` field mapping — resolved (category, flag, message, lap_number)
2. Race-control deduplication strategy — resolved (group by activation type + lap_number)
3. Fastest lap time storage — resolved (new `FastestLapTimeSeconds` field on `Session`)
4. Recap payload computation location — resolved (server-side in rounds service)
5. Lazy fill architecture — resolved (injected `RaceControlHydrator` interface)
6. Backfill repository access — resolved (new `GetFinalizedSessions` repo method)
7. Q1/Q2 cutoff time derivation — resolved (last driver with only Q1/Q2 time)
8. Practice "total laps" — resolved (sum across all drivers)
9. Session ordering — resolved (chronological ascending by `DateStartUTC`)
10. `SessionSchemaVersion` handling — resolved (do NOT bump; additive nil-safe fields)

---

## Phase 1: Design & Contracts

### 1.1 Data Model

See [data-model.md](data-model.md) for full type definitions.

#### Storage layer additions

`storage.Session` gets two new optional fields (additive, nil-safe):
- `RaceControlSummary *RaceControlSummary` — persisted at finalization and by backfill
- `FastestLapTimeSeconds *float64` — persisted at finalization by session poller

`storage.RaceControlSummary` (new struct):
- `RedFlagCount int`, `SafetyCarCount int`, `VSCCount int`
- `NotableEvents []NotableEvent` — ordered by priority (highest first)
- `FetchedAtUTC time.Time`

`storage.NotableEvent` (new struct):
- `EventType string` — `"red_flag"`, `"safety_car"`, `"vsc"`, `"investigation"`
- `LapNumber int` — first occurrence; 0 for pre-race events
- `Count int` — distinct activations

`storage.SessionRepository` interface gains one method:
- `GetFinalizedSessions(ctx, season) ([]Session, error)`

#### API layer additions

`rounds.SessionDetailDTO` gains:
- `RecapSummary *SessionRecapDTO` — nil for non-completed sessions

`rounds.SessionRecapDTO` (new struct, all fields `omitempty`):
- Race/Sprint fields: `WinnerName`, `WinnerTeam`, `GapToP2`, `FastestLapHolder`, `FastestLapTeam`, `FastestLapTimeSeconds *float64`, `TotalLaps`
- Qualifying fields: `PoleSitterName`, `PoleSitterTeam`, `PoleTime *float64`, `Q1CutoffTime *float64`, `Q2CutoffTime *float64`
- Practice fields: `BestDriverName`, `BestDriverTeam`, `BestLapTime *float64`
- All sessions: `RedFlagCount`, `SafetyCarCount`, `VSCCount` (int, omitempty=0), `TopEvent *NotableEventDTO`

### 1.2 Backend Implementation Details

#### `backend/internal/ingest/race_control.go` (new file)

**Purpose**: Encapsulate all OpenF1 race_control interaction.

```
Functions:
  FetchRaceControlMsgs(ctx, client, sessionKey) ([]openF1RaceControlMsg, error)
    → GET https://api.openf1.org/v1/race_control?session_key={key}
    → Returns raw message slice; caller handles deduplication

  SummarizeRaceControl(msgs []openF1RaceControlMsg) storage.RaceControlSummary
    → Deduplication: group by activation type + lap_number
    → Red flag: flag == "RED"
    → Safety Car: message starts with "SAFETY CAR DEPLOYED"
    → VSC: message starts with "VIRTUAL SAFETY CAR DEPLOYED"
    → Investigation: category == "Other" && message contains "UNDER INVESTIGATION"
    → Counts = distinct lap_number groups per type (lap_number==0 grouped by ±60s time window)
    → NotableEvents: sorted by priority descending, one entry per type with count > 0
    → Returns summary with FetchedAtUTC = time.Now().UTC()

Types:
  RaceControlHydrator struct
    → Depends on: *http.Client, storage.SessionRepository, *slog.Logger
    → Constructor: NewRaceControlHydrator(repo, logger) *RaceControlHydrator
    → Implements: Hydrate(ctx, sess storage.Session) (*storage.RaceControlSummary, error)
       1. Call FetchRaceControlMsgs
       2. Call SummarizeRaceControl
       3. Patch sess.RaceControlSummary = &summary
       4. Call repo.UpsertSession(ctx, sess) (read-modify-write)
       5. Return &summary
       On error: return nil, err (caller handles graceful degradation)
```

#### `backend/internal/ingest/session_poller.go` extensions

At the point where `sess.Finalized` is set to `true` (after `fetchAndUpsertResults` succeeds and `time.Since(sess.DateEndUTC) >= finalizationBuffer`), add:

1. **Fetch race_control**: Call `FetchRaceControlMsgs(ctx, p.client, raw.SessionKey)`, then `SummarizeRaceControl`. Store in `sess.RaceControlSummary`. Rate-limit delay (500ms) before this call.

2. **Store fastest lap time**: Extract from the `laps` slice already fetched in `fetchAndUpsertResults`. Find the lap of the driver returned by `DeriveFastestLap` with the minimum non-nil `LapDuration`. Store in `sess.FastestLapTimeSeconds`. No additional API call needed.

3. Both stored in the same `UpsertSession` call that sets `sess.Finalized = true`.

`SessionSchemaVersion` stays at 3 (NOT bumped). No re-fetch of existing sessions.

#### `backend/internal/api/rounds/service.go` extensions

**Inject hydrator**:

```go
type RaceControlHydrator interface {
    Hydrate(ctx context.Context, sess storage.Session) (*storage.RaceControlSummary, error)
}

type Service struct {
    sessionRepo  storage.SessionRepository
    calendarRepo storage.CalendarRepository
    rcHydrator   RaceControlHydrator // may be nil (tests; hydration skipped gracefully)
    now          func() time.Time
    logger       *slog.Logger        // add for lazy-fill warning logs
}
```

Add `NewServiceWithHydrator(sessionRepo, calendarRepo, hydrator, logger) *Service` constructor.

**Lazy fill in `GetRoundDetail`** (inside the per-session loop, before building recap):

```go
if status == statusCompleted && sess.RaceControlSummary == nil && s.rcHydrator != nil {
    if summary, err := s.rcHydrator.Hydrate(ctx, sess); err != nil {
        s.logger.Warn("lazy race control fill failed — recap rendered without events",
            "session_id", sess.ID, "error", err)
    } else {
        sess.RaceControlSummary = summary
    }
}
```

**Recap derivation** (`deriveRecapSummary(sess, results) *SessionRecapDTO`):

```
Race / Sprint:
  Find P1 result (position == 1, not DNF/DNS/DSQ) → winner
  Find P2 result → gap_to_p2 (from GapToLeader field)
  Find result where FastestLap == true → fastest_lap_holder, fastest_lap_team
  FastestLapTimeSeconds from sess.FastestLapTimeSeconds
  TotalLaps from P1 result's NumberOfLaps
  Race-control event fields from sess.RaceControlSummary

Qualifying / Sprint Qualifying:
  P1 result → pole_sitter; pole_time = Q3Time (or Q2Time or Q1Time, first non-nil from Q3→Q1)
  P2 result → gap_to_p2 = pole_time_secs - p2_pole_time_secs (formatted as "+X.XXX")
  Q1 cutoff: last result with non-nil Q1Time AND nil Q2Time → Q1Time
  Q2 cutoff: last result with non-nil Q2Time AND nil Q3Time → Q2Time
  Race-control event fields from sess.RaceControlSummary (RedFlagCount primarily)

Practice:
  P1 result (lowest BestLapTime, already sorted) → best_driver, best_lap_time
  TotalLaps = sum(r.NumberOfLaps for r in results)
  Race-control event fields from sess.RaceControlSummary (RedFlagCount primarily)

TopEvent derivation (all types):
  priority order: red_flag > safety_car > vsc > investigation
  Pick highest-priority event type where count > 0
  TopEvent = {EventType, LapNumber (first occurrence), Count}
  If no events, TopEvent = nil
```

#### `backend/cmd/backfill/main.go` (new file)

```
Package: main
Binary: backfill

Flags:
  --season=2026    (required)
  --dry-run        (skip writes, log what would change)
  --rate-limit-ms=1000  (delay between OpenF1 fetches)

Wiring:
  1. Read COSMOS_ENDPOINT from env (same as main service)
  2. Create Cosmos client (Managed Identity / key, same as config/secrets.go)
  3. Create storage.SessionRepository (cosmos.Client)
  4. Call repo.GetFinalizedSessions(ctx, season)
  5. Filter to sessions where RaceControlSummary == nil
  6. For each session:
     a. Log: {"level":"INFO","msg":"backfill: fetching","session_id":"...","session_key":...}
     b. Call FetchRaceControlMsgs(ctx, httpClient, session.SessionKey)
     c. If error or empty: log WARN and continue
     d. Call SummarizeRaceControl(msgs)
     e. If not dry-run: patch session.RaceControlSummary, call repo.UpsertSession
     f. Log outcome: {"level":"INFO","msg":"backfill: updated","session_id":"...","outcome":"updated"}
     g. Sleep rate-limit-ms
  7. Log summary: {"level":"INFO","msg":"backfill: complete","updated":N,"skipped":N,"failed":N}

Notes:
  - Idempotent: sessions already with RaceControlSummary are skipped without modification
  - Does NOT fetch fastest lap time (only race_control, per FR-005)
  - Uses structured JSON logs (log/slog, same as main service)
  - Runs as a one-shot local binary or kubectl exec; not a K8s Job
```

#### `backend/internal/storage/cosmos/sessions.go` extension

Add `GetFinalizedSessions` implementation:

```go
func (c *Client) GetFinalizedSessions(ctx context.Context, season int) ([]storage.Session, error) {
    pk := azcosmos.NewPartitionKeyNumber(float64(season))
    query := `SELECT * FROM c WHERE c.season = @season AND c.type = 'session' AND c.finalized = true`
    // ... standard pager pattern, same as existing queries ...
}
```

### 1.3 Frontend Implementation Details

#### `frontend/src/features/rounds/roundApi.ts`

Add `NotableEvent`, `SessionRecapSummary` interfaces and `recap_summary?: SessionRecapSummary` to `SessionDetail`. See [contracts/rounds-recap-api.md](contracts/rounds-recap-api.md).

#### `frontend/src/features/rounds/SessionRecapStrip.tsx` (new)

```
Props: { sessions: SessionDetail[] }

Behavior:
  - Filter to sessions where status === 'completed' && recap_summary != null
  - Sort chronologically by date_start_utc ascending (FP1 → Race)
  - Render a strip container:
    - Mobile (≤768px): flex-col (vertical stack), full-width cards
    - Desktop (>768px): flex-row, overflow-x-auto, fixed-width cards (~280px)
  - For each session, render the appropriate card by session_type:
    - race / sprint → <RaceRecapCard>
    - qualifying / sprint_qualifying → <QualifyingRecapCard>
    - practice1 / practice2 / practice3 → <PracticeRecapCard>
  - If no completed sessions with recap data, render nothing (null)

Styling:
  - Reuse existing design system atoms: getTeamColor from teamColors.ts,
    LapTimeDisplay for lap times, no TireCompound (not applicable to recap)
  - Session type label from session_name field (e.g., "Practice 1")
  - Team color as left border accent (same pattern as DriverCard)
```

#### `frontend/src/features/rounds/RaceRecapCard.tsx` (new)

```
Props: { session: SessionDetail }  (recap_summary guaranteed non-null by parent)

Renders:
  - Session label: session.session_name (e.g., "Race" or "Sprint")
  - Winner row: team-color left border, winner_name, winner_team
  - Gap to P2: gap_to_p2 string (omit row if absent)
  - Fastest lap row: fastest_lap_holder, LapTimeDisplay for fastest_lap_time_seconds (omit if absent)
  - Total laps: "{total_laps} laps" (omit if 0)
  - Top event row: render if top_event exists:
    - red_flag → "🚩 Red Flag — Lap {lap_number}" (or count if > 1)
    - safety_car → "SC × {count}" (or "Safety Car — Lap {lap_number}" if count == 1)
    - vsc → "VSC × {count}"
  - Note: emojis per design spec; no additional icon library required
```

#### `frontend/src/features/rounds/QualifyingRecapCard.tsx` (new)

```
Props: { session: SessionDetail }

Renders:
  - Session label
  - Pole sitter: pole_sitter_name, pole_sitter_team (team color border)
  - Pole time: LapTimeDisplay for pole_time
  - Gap to P2: gap_to_p2
  - Q1 cutoff: LapTimeDisplay for q1_cutoff_time (omit row if absent)
  - Q2 cutoff: LapTimeDisplay for q2_cutoff_time (omit row if absent)
  - Red flags: "{red_flag_count} Red Flag(s)" (omit if 0)
```

#### `frontend/src/features/rounds/PracticeRecapCard.tsx` (new)

```
Props: { session: SessionDetail }

Renders:
  - Session label (e.g., "Practice 1")
  - Best driver: best_driver_name, best_driver_team (team color border)
  - Best lap time: LapTimeDisplay for best_lap_time
  - Total laps: "{total_laps} laps"
  - Red flags: (omit if 0)
```

#### `frontend/src/features/rounds/RoundDetailPage.tsx` extension

Insert `<SessionRecapStrip sessions={data.sessions} />` between the round header section and the session cards list. Renders nothing automatically if no completed sessions.

### 1.4 Test Plan

#### Backend unit tests

**`backend/tests/unit/race_control_test.go`** (new):
- `TestSummarizeRaceControl_RedFlag`: single RED flag message → count=1
- `TestSummarizeRaceControl_SafetyCar_Deduplicated`: two messages with same lap_number → count=1
- `TestSummarizeRaceControl_SafetyCar_TwoDistinctLaps`: two messages with different lap_numbers → count=2
- `TestSummarizeRaceControl_VSC_IgnoresEndingMessages`: deployment + ending = count=1
- `TestSummarizeRaceControl_EmptyMessages`: returns zero-value summary
- `TestSummarizeRaceControl_Priority_RedFlagOverSC`: red flag + SC → TopEvent is red_flag

**`backend/tests/unit/rounds_recap_test.go`** (new, testing `deriveRecapSummary`):
- `TestDeriveRecap_Race_Winner`: P1 result maps to winner fields
- `TestDeriveRecap_Race_NoClassifiedFinishers`: omits winner fields gracefully
- `TestDeriveRecap_Race_FastestLap`: FastestLap=true result maps to holder
- `TestDeriveRecap_Qualifying_PoleAndCutoffs`: Q1/Q2 cutoff derivation
- `TestDeriveRecap_Qualifying_SprintFormat_NoCutoffs`: nil Q1/Q2 cutoff
- `TestDeriveRecap_Practice_BestDriver`: P1 result maps to best driver
- `TestDeriveRecap_Practice_TotalLaps`: sums NumberOfLaps

#### Backend contract tests

**`backend/tests/contract/rounds_contract_test.go`** (extend existing):
- `TestRoundDetail_RecapSummary_CompletedRace`: response includes recap_summary with winner
- `TestRoundDetail_RecapSummary_UpcomingSession`: recap_summary absent for upcoming session
- `TestRoundDetail_RecapSummary_NilHydrator_Degrades`: nil hydrator → recap omits events but still renders winner

#### Frontend unit tests

**`frontend/tests/rounds/SessionRecapStrip.test.tsx`** (new):
- Strip renders nothing when no completed sessions
- Strip renders one card per completed session in chronological order
- Strip renders correct card type per session type

**`frontend/tests/rounds/RecapCards.test.tsx`** (new):
- RaceRecapCard renders winner, gap, fastest lap
- RaceRecapCard omits fastest lap time when absent
- RaceRecapCard renders top_event for safety car
- QualifyingRecapCard renders pole, Q1/Q2 cutoffs
- QualifyingRecapCard omits Q1/Q2 cutoff when absent (sprint qualifying)
- PracticeRecapCard renders best driver and total laps

---

## Phase 2: Constitution Re-check (Post-Design)

After Phase 1 design, re-verify all gates:

| Gate | Post-Design Status | Notes |
|------|-------------------|-------|
| Stack gate | ✅ PASS | No new platform components introduced |
| Architecture gate | ✅ PASS | `RaceControlHydrator` lives in `ingest/` package, not in `api/` layer; injected via interface |
| Data gate | ✅ PASS | Race-control data persisted before response served (lazy fill read-modify-write confirmed); backfill is idempotent |
| Security gate | ✅ PASS | Backfill CLI reads Cosmos credentials identically to main service; no new secret surface |
| Network gate | ✅ PASS | Outbound call to `api.openf1.org` already in Azure Firewall allow-list |
| Delivery gate | ✅ PASS | `cmd/backfill` is a standalone binary, not a new K8s Job or Helm resource |
| Observability gate | ✅ PASS | All new code paths use `log/slog` structured JSON; backfill logs per-session outcome |
| Dependency gate | ✅ PASS | Zero new Go modules or npm packages required |
| Spec authority gate | ✅ PASS | All 18 FRs and 10 CAs traced to concrete implementation steps above |

---

## Requirement Traceability

| Req | Implementation location |
|-----|------------------------|
| FR-001 | `session_poller.go` — race_control fetch at finalization |
| FR-002 | `storage/repository.go` `RaceControlSummary`; `ingest/race_control.go` `SummarizeRaceControl` |
| FR-003 | `ingest/race_control.go` `SummarizeRaceControl` deduplication by lap_number |
| FR-004 | `session_poller.go` — existing `DeriveFastestLap` call unchanged; `FastestLapTimeSeconds` derived from same laps slice |
| FR-005 | `cmd/backfill/main.go` |
| FR-006 | `api/rounds/service.go` lazy fill via `RaceControlHydrator.Hydrate` |
| FR-007 | `api/rounds/dto.go` `SessionRecapDTO`; `api/rounds/service.go` `deriveRecapSummary` |
| FR-008 | `api/rounds/service.go` `deriveSessionStatus` (unchanged Day 12 pattern) |
| FR-009 | `frontend/src/features/rounds/SessionRecapStrip.tsx` — chronological sort |
| FR-010 | `api/rounds/service.go` race recap derivation + `RaceRecapCard.tsx` |
| FR-011 | `api/rounds/service.go` qualifying recap derivation + `QualifyingRecapCard.tsx` |
| FR-012 | `api/rounds/service.go` practice recap derivation + `PracticeRecapCard.tsx` |
| FR-013 | `SessionRecapStrip.tsx` — sprint → RaceRecapCard, sprint_qualifying → QualifyingRecapCard |
| FR-014 | `api/rounds/service.go` `deriveTopEvent` priority logic |
| FR-015 | `SessionRecapDTO` omitempty + nil TopEvent |
| FR-016 | `SessionRecapStrip.tsx` responsive CSS (flex-col on mobile, flex-row overflow-x-auto on desktop) |
| FR-017 | No auto-refresh, no polling, no WebSocket — static load only |
| FR-018 | `roundApi.ts` uses `apiClient.get` (backend endpoint) — no direct OpenF1 calls |

---

## Decisions Log

| Decision | Rationale |
|----------|-----------|
| Embed `RaceControlSummary` in `Session` document, not a new container | Constitution: no new infrastructure; Cosmos document size for race_control messages is small (~<5KB) |
| Do NOT bump `SessionSchemaVersion` | Prevents unnecessary re-fetch of all finalized sessions' results/drivers/laps; new fields are nil-safe |
| `RaceControlHydrator` in `ingest/`, not `api/` | Keeps HTTP client code out of the API layer; maintains clean tier separation |
| Backfill as CLI binary, not K8s Job | Constitution: no new Helm resources; one-shot manual step as specified in CA-006 |
| Derive `FastestLapTimeSeconds` from laps already in memory | Reuses Feature 003 laps fetch without additional API call per FR-004 |
| Backend computes recap payload (not frontend) | Keeps derivation logic testable, server-side, and independent of frontend state |
| Use GapToLeader string from existing results for gap_to_p2 | Already formatted by Feature 003 ingest transform; no re-formatting needed |
| omitempty on all recap int fields (zero = absent) | Red/SC/VSC counts of 0 should not appear on cards (FR-015) |

---

## Open Items / Risks

| Item | Severity | Mitigation |
|------|----------|-----------|
| OpenF1 `/v1/race_control` may return empty for some sessions (pre-2023 historical data) | Low — 2026 only | Log WARN and skip; backfill continues (FR-005 Scenario 2) |
| Fastest lap time nil for pre-Feature-005 sessions | Low | Spec Edge Cases: omit field; card renders without time |
| Q1/Q2 cutoff derivation assumes standard 20-car grid | Low | Sprint qualifying and irregular formats fall back to nil cutoff (field omitted) |
| `RaceControlHydrator.Hydrate` adds latency to first read of a session | Low | Lazy fill is a one-time cost per session; result is persisted; subsequent reads are fast |
| Concurrent requests triggering multiple lazy fills for the same session | Low | Cosmos upsert is idempotent; duplicate fetches are wasteful but not harmful; acceptable for current scale |
