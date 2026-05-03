# Implementation Plan: Session Deep Dive Page

**Branch**: `006-session-deep-dive` | **Date**: 2026-05-02 | **Spec**: [spec.md](spec.md)  
**Input**: Feature specification from `/specs/006-session-deep-dive/spec.md`

---

## Summary

Add a dedicated post-session analysis page accessible from round detail, providing rich visualizations (position chart, gap-to-leader, tire strategy swimlane, pit stop timeline, overtake annotations) for completed Race and Sprint sessions. The backend ingests five OpenF1 endpoints (`/position`, `/intervals`, `/stints`, `/pit`, `/overtakes`) using the same 2h post-session buffer pattern from Feature 005, aggregates position data to 1 point per driver per lap server-side, and exposes a single combined `GET /api/v1/rounds/{round}/sessions/{type}/analysis` endpoint. The frontend adds a new `analysis` feature with `recharts`-based chart components. A backfill CLI extension populates data for all pre-existing 2026 Race/Sprint sessions.

---

## Technical Context

**Language/Version**: Go 1.25+ (backend), TypeScript 5.6 / React 18 (frontend)  
**Primary Dependencies**: Chi v5 router, Azure Cosmos DB SDK for Go, recharts (NEW — React charting library), Vitest 4.1  
**Storage**: Azure Cosmos DB serverless — `sessions` container, `season` partition key; new document types for position/interval/stint/pit/overtake data  
**Testing**: `go test ./...` (backend), `npx vitest run` (frontend, pool: "threads")  
**Target Platform**: AKS 1.33 (existing deployment)  
**Project Type**: Web application (Go API + React SPA)  
**Performance Goals**: Analysis page renders all charts within 3 seconds of navigation; API response <1s for largest expected payload (~1200 aggregated position points + interval/stint/pit/overtake data)  
**Constraints**: OpenF1 rate limit ≤1 req/s respected by poller and backfill; position data aggregated server-side to avoid sending 5000+ raw rows to frontend; mobile-responsive charts  
**Scale/Scope**: ~5-7 Race/Sprint sessions in 2026 at launch needing backfill; ~20 drivers × ~60 laps = ~1200 position points per session (post-aggregation)

---

## Constitution Check

| Gate | Status | Notes |
|------|--------|-------|
| **Stack gate** | ✅ PASS | Go backend, React frontend, Cosmos DB serverless, AKS — no new platform components |
| **Architecture gate** | ✅ PASS | Frontend calls only the backend `/api/v1/rounds/{round}/sessions/{type}/analysis` endpoint; OpenF1 calls are backend-only (ingest poller, backfill CLI) |
| **Data gate** | ✅ PASS | All 5 OpenF1 data types fetched at session finalization + 2h buffer, persisted to Cosmos before serving. Backfill covers pre-existing sessions. No pass-through at request time. Data is immutable post-session — cached indefinitely. |
| **Security gate** | ✅ PASS | No new secrets. OpenF1 is free-tier, no API key. Backfill CLI uses same Managed Identity / Key Vault path as main service. |
| **Network gate** | ✅ PASS | No changes to ingress TLS or egress firewall rules. Backend already has egress to `api.openf1.org`. |
| **Delivery gate** | ✅ PASS | No new Helm/Bicep resources. CI/CD pipeline order unchanged: lint → test → build → push → deploy. Backfill is a manual post-deploy step (extends existing CLI). |
| **Observability gate** | ✅ PASS | Ingestion logs structured JSON (session key, data type, row count, duration). API handler logs request timing. |
| **Dependency gate** | ✅ PASS | One new dependency: `recharts`. Justification documented in [dependency-justification.md](dependency-justification.md). Owner: @karlkuhnhausen. |
| **Spec authority gate** | ✅ PASS | All work items trace to FR-001–FR-020 and CA-001–CA-010 in spec.md. |

---

## Project Structure

### Documentation (this feature)

```
specs/006-session-deep-dive/
├── plan.md              ← This file
├── research.md          ← Phase 0 output
├── data-model.md        ← Phase 1 output
├── quickstart.md        ← Phase 1 output
├── dependency-justification.md ← Constitution CA-009 requirement
├── contracts/
│   └── analysis-api.md  ← Phase 1 output
└── tasks.md             ← Phase 2 output (/speckit.tasks — NOT created here)
```

### Source Code Changes

```
backend/
├── internal/
│   ├── domain/
│   │   └── analysis.go              ← NEW: SessionPosition, SessionInterval,
│   │                                         SessionStint, SessionPit, SessionOvertake types
│   ├── storage/
│   │   ├── repository.go            ← EXTEND: add AnalysisRepository interface
│   │   └── cosmos/
│   │       └── analysis.go          ← NEW: Cosmos implementation of AnalysisRepository
│   ├── ingest/
│   │   ├── analysis.go              ← NEW: OpenF1 fetchers for 5 endpoints + aggregation
│   │   └── session_poller.go        ← EXTEND: trigger analysis ingestion at finalization
│   └── api/
│       ├── analysis/
│       │   ├── handler.go           ← NEW: GET handler for session analysis
│       │   ├── dto.go               ← NEW: SessionAnalysisDTO response types
│       │   └── service.go           ← NEW: orchestrate data fetch from repository
│       └── router.go                ← EXTEND: register analysis route
└── cmd/
    └── backfill/
        └── main.go                  ← EXTEND: add --analysis flag for analysis data backfill

frontend/
├── src/
│   ├── features/
│   │   ├── analysis/
│   │   │   ├── AnalysisPage.tsx         ← NEW: main analysis page layout
│   │   │   ├── PositionChart.tsx        ← NEW: lap-by-lap position chart (recharts)
│   │   │   ├── GapToLeaderChart.tsx     ← NEW: gap progression chart (recharts)
│   │   │   ├── TireStrategyChart.tsx    ← NEW: swimlane compound chart (recharts)
│   │   │   ├── PitStopTimeline.tsx      ← NEW: pit stop timing visualization (recharts)
│   │   │   ├── analysisApi.ts           ← NEW: API client for analysis endpoint
│   │   │   └── analysisTypes.ts         ← NEW: TypeScript types for analysis data
│   │   └── rounds/
│   │       └── RoundDetailPage.tsx      ← EXTEND: add "View Analysis" buttons
│   └── app/
│       └── routes.tsx                   ← EXTEND: add analysis route
├── tests/
│   └── analysis/
│       ├── AnalysisPage.test.tsx        ← NEW
│       ├── PositionChart.test.tsx       ← NEW
│       ├── GapToLeaderChart.test.tsx    ← NEW
│       ├── TireStrategyChart.test.tsx   ← NEW
│       └── PitStopTimeline.test.tsx     ← NEW
```

---

## Phase 0: Research

**Status**: Complete. See [research.md](research.md).

All technical unknowns resolved:
1. OpenF1 `/position` response shape and aggregation strategy
2. OpenF1 `/intervals` response shape and gap-to-leader derivation
3. OpenF1 `/stints` response shape and compound mapping
4. OpenF1 `/pit` response shape and duration fields
5. OpenF1 `/overtakes` data availability and completeness guarantees
6. Recharts suitability for multi-line charts with 20 series × 60 points
7. Cosmos DB document model: batch documents vs. per-row documents
8. Position data aggregation algorithm (server-side deduplication)
9. Backfill CLI extension strategy (new flag vs. new subcommand)
10. Frontend route structure and navigation flow

---

## Phase 1: Design & Contracts

### 1.1 Data Model

See [data-model.md](data-model.md) for full type definitions.

#### Storage Layer Additions

**New interface** — `storage.AnalysisRepository`:
```go
type AnalysisRepository interface {
    UpsertSessionPositions(ctx context.Context, season, round int, sessionType string, positions []SessionPosition) error
    UpsertSessionIntervals(ctx context.Context, season, round int, sessionType string, intervals []SessionInterval) error
    UpsertSessionStints(ctx context.Context, season, round int, sessionType string, stints []SessionStint) error
    UpsertSessionPits(ctx context.Context, season, round int, sessionType string, pits []SessionPit) error
    UpsertSessionOvertakes(ctx context.Context, season, round int, sessionType string, overtakes []SessionOvertake) error
    GetSessionAnalysis(ctx context.Context, season, round int, sessionType string) (*SessionAnalysisData, error)
    HasAnalysisData(ctx context.Context, season, round int, sessionType string) (bool, error)
}
```

**New document types** (stored in `sessions` container with `season` partition key):
- `SessionPosition` — type discriminator: `"analysis_position"`
- `SessionInterval` — type discriminator: `"analysis_interval"`
- `SessionStint` — type discriminator: `"analysis_stint"`
- `SessionPit` — type discriminator: `"analysis_pit"`
- `SessionOvertake` — type discriminator: `"analysis_overtake"`

Each uses composite document IDs: `analysis_{datatype}_{round}_{sessiontype}_{drivernum}` to enable idempotent upserts and efficient querying by session.

#### API Layer

**New endpoint**: `GET /api/v1/rounds/{round}/sessions/{type}/analysis?year={year}`

Returns `SessionAnalysisDTO` containing all five data arrays. See [contracts/analysis-api.md](contracts/analysis-api.md).

### 1.2 Backend Implementation Details

#### `backend/internal/domain/analysis.go` (new file)

Domain types for the five analysis data categories:

```go
type SessionPosition struct {
    DriverNumber  int
    DriverName    string
    DriverAcronym string
    TeamName      string
    Laps          []PositionLap  // one entry per lap
}

type PositionLap struct {
    LapNumber int
    Position  int
}

type SessionInterval struct {
    DriverNumber  int
    DriverAcronym string
    TeamName      string
    Laps          []IntervalLap
}

type IntervalLap struct {
    LapNumber   int
    GapToLeader float64  // seconds; 0 for leader
    Interval    float64  // gap to car ahead; 0 for leader
}

type SessionStint struct {
    DriverNumber  int
    DriverAcronym string
    TeamName      string
    StintNumber   int
    Compound      string  // SOFT, MEDIUM, HARD, INTERMEDIATE, WET
    LapStart      int
    LapEnd        int
    TyreAgeAtStart int
}

type SessionPit struct {
    DriverNumber  int
    DriverAcronym string
    TeamName      string
    LapNumber     int
    PitDuration   float64  // seconds (pit lane time)
    StopDuration  float64  // seconds (stationary time)
}

type SessionOvertake struct {
    OvertakingDriverNumber int
    OvertakingDriverName   string
    OvertakenDriverNumber  int
    OvertakenDriverName    string
    LapNumber              int
    Position               int  // resulting position of overtaking driver
}
```

#### `backend/internal/ingest/analysis.go` (new file)

**Purpose**: Encapsulate all OpenF1 analysis data fetching and server-side aggregation.

```
Functions:
  FetchPositionData(ctx, client, sessionKey) ([]RawPosition, error)
    → GET https://api.openf1.org/v1/position?session_key={key}
    → Returns raw position entries (multiple per driver per lap — position updates in real-time)

  AggregatePositions(raw []RawPosition, drivers map[int]DriverInfo) []domain.SessionPosition
    → Deduplicate: keep LAST position entry per (driver_number, lap_number)
    → This gives the final position each driver held at end of each lap
    → Attach driver name, acronym, team from the drivers map
    → Result: 1 point per driver per lap (FR-010)

  FetchIntervalData(ctx, client, sessionKey) ([]RawInterval, error)
    → GET https://api.openf1.org/v1/intervals?session_key={key}
    → Returns gap_to_leader and interval for each driver at sample points

  AggregateIntervals(raw []RawInterval, drivers map[int]DriverInfo) []domain.SessionInterval
    → Deduplicate: keep LAST interval entry per (driver_number, lap_number)
    → Lap number derived from the entry's lap_number field (present in API since 2024)

  FetchStintData(ctx, client, sessionKey) ([]domain.SessionStint, error)
    → GET https://api.openf1.org/v1/stints?session_key={key}
    → Maps directly to domain type; compound normalization (uppercase)

  FetchPitData(ctx, client, sessionKey) ([]domain.SessionPit, error)
    → GET https://api.openf1.org/v1/pit?session_key={key}
    → Maps pit_duration and duration fields

  FetchOvertakeData(ctx, client, sessionKey) ([]domain.SessionOvertake, error)
    → GET https://api.openf1.org/v1/overtaking?session_key={key}
    → May return empty array — not an error condition (FR-015)

  FetchAllAnalysisData(ctx, client, sessionKey, drivers) (*AnalysisFetchResult, error)
    → Orchestrates all 5 fetches with 500ms delays between each
    → Logs structured JSON: data type, row count, fetch duration
    → Returns combined result; partial failures for non-position data logged but non-fatal
    → Position data is REQUIRED; if it fails, returns error
```

#### `backend/internal/ingest/session_poller.go` extensions

At session finalization (after `fetchAndUpsertResults` succeeds and 2h buffer elapsed), for Race and Sprint sessions only:

1. **Call `FetchAllAnalysisData`** with the session key and driver info already available from the results fetch.
2. **Persist via `AnalysisRepository`**: Upsert all five data types.
3. **Rate limiting**: 500ms between each of the 5 OpenF1 fetches (total ~2.5s additional per session).
4. **Failure handling**: Log error and continue — analysis data is non-blocking for session finalization. The backfill CLI can recover any missed sessions.

#### `backend/internal/api/analysis/` (new package)

- **handler.go**: Chi-compatible HTTP handler; extracts `round` and `type` URL params, validates session type, delegates to service.
- **service.go**: Calls `AnalysisRepository.GetSessionAnalysis`; maps domain types to DTOs.
- **dto.go**: Response types matching the API contract.

#### `backend/internal/api/router.go` extension

```go
// Analysis API
analysisSvc := analysis.NewService(analysisRepo, logger)
analysisHandler := analysis.NewHandler(analysisSvc, logger)

r.Route("/api/v1", func(r chi.Router) {
    // ... existing routes ...
    r.Get("/rounds/{round}/sessions/{type}/analysis", analysisHandler.GetSessionAnalysis)
})
```

#### `backend/cmd/backfill/main.go` extension

Add `--analysis` flag (default: false). When set:
1. Query all finalized Race and Sprint sessions for the season via `GetFinalizedSessions`.
2. For each, check `analysisRepo.HasAnalysisData` — skip if already populated (idempotent per FR-017).
3. Fetch all 5 endpoints from OpenF1, aggregate, persist.
4. Rate limit: 1000ms between sessions, 500ms between endpoints within a session.
5. Log progress: session_key, status (updated/skipped/failed), counts per data type.
6. Continue on individual session failure (FR-016 Scenario 3).

### 1.3 Frontend Implementation Details

#### New dependency: `recharts`

**Version**: ^2.12 (latest stable)  
**Justification**: See [dependency-justification.md](dependency-justification.md).

#### `frontend/src/features/analysis/AnalysisPage.tsx`

- Route: `/rounds/:round/sessions/:sessionType/analysis`
- Fetches analysis data on mount via `analysisApi.ts`
- Renders back-navigation link ("← Back to Round {N}"), session header, then vertically-stacked charts
- Shows "Analysis not yet available" state when API returns 404 (FR-014)
- Gracefully omits individual chart sections when their data arrays are empty (FR-015)
- Mobile layout: full-width charts stacked vertically (FR-018)

#### Chart Components (all use `ResponsiveContainer` for mobile)

- **PositionChart**: `LineChart`, inverted Y-axis (1 top, 20 bottom), one `Line` per driver colored by team, X = lap. Supports overtake annotations as `ReferenceDot` markers when overtake data is present. (FR-004, FR-008, FR-009)
- **GapToLeaderChart**: `LineChart`, gap (seconds) Y-axis, one `Line` per driver. (FR-005)
- **TireStrategyChart**: Custom swimlane using recharts `BarChart` with stacked horizontal bars; compound-colored (Soft=red, Medium=yellow, Hard=white, Inter=green, Wet=blue). One row per driver. (FR-006)
- **PitStopTimeline**: `ScatterChart` with lap on X-axis, drivers on Y-axis (categorical), dot size proportional to stop duration. Distinguishes slow stops (>5s) with different styling. (FR-007)

#### Route and Navigation

```tsx
// routes.tsx addition
<Route path="/rounds/:round/sessions/:sessionType/analysis" element={<AnalysisPage />} />
```

"View Analysis" button on `RoundDetailPage.tsx` for completed Race/Sprint sessions (FR-002, FR-003):
```tsx
{session.status === 'completed' && isRaceType(session.session_type) && (
  <Link to={`/rounds/${roundNum}/sessions/${session.session_type}/analysis?year=${year}`}>
    View Analysis →
  </Link>
)}
```

---

## Complexity Tracking

> No constitution violations. No complexity justifications needed.

| Item | Notes |
|------|-------|
| New dependency (`recharts`) | Justified: 5 chart types required, no existing charting in the project, recharts is React-native and tree-shakeable. See dependency-justification.md. |
| New Cosmos document types (5) | Required by spec: each data category has different structure and query patterns |
| Separate AnalysisRepository interface | Clean separation from SessionRepository; analysis data is structurally different (batch per-driver documents vs. single session documents) |
