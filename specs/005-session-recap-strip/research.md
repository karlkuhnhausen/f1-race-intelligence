# Research: Session Recap Strip (Feature 005)

## Resolved Questions

### 1. OpenF1 `/v1/race_control` Response Shape
**Decision**: Parse `category`, `flag`, `message`, and `lap_number` fields from each message.

OpenF1 race control message fields used by this feature:
| Field | Type | Notes |
|-------|------|-------|
| `category` | string | "Flag", "SafetyCar", "Drs", "Other" |
| `flag` | string | "RED", "YELLOW", "GREEN", "CHEQUERED", "SC", "VSC" (null for SafetyCar category) |
| `message` | string | Human-readable message text; distinguishes SC vs VSC deployments |
| `lap_number` | int | Lap on which the event occurred (may be 0 for pre-race events) |
| `session_key` | int | Must match the requested session |

**Rationale**: These are the only fields needed for deduplication and for deriving the notable event list.

**Alternatives considered**: Using `scope` or `driver_number` fields for sector-specific flags — rejected because our summary is session-level, not sector-specific.

---

### 2. Race-Control Deduplication Logic
**Decision**: Deduplicate by grouping messages of the same activation type by `lap_number`. Messages of the same type on the same lap are counted as one event. If lap is 0, use time-proximity (60-second window).

**Activation type classification**:
- **Red flag**: `flag == "RED"` (ignore "GREEN"/"CLEAR" flag events that end the red flag period)
- **Safety Car**: `message` starts with `"SAFETY CAR DEPLOYED"` (not `"SAFETY CAR IN THIS LAP"`, not `"SAFETY CAR ENDING"`)
- **Virtual Safety Car**: `message` starts with `"VIRTUAL SAFETY CAR DEPLOYED"` (not `"ENDING"`)
- **Investigation**: `category == "Other"` and `message` contains `"UNDER INVESTIGATION"` — only counted for the priority ranking if red flag / SC / VSC counts are all zero

**Rationale**: OpenF1 occasionally emits multiple messages for the same activation event (e.g., deployment confirmation messages). Grouping by lap_number is the simplest reliable deduplication strategy. The lap_number is always present for events that occur during a lap.

**Alternatives considered**: Time-proximity deduplication (±60s) — retained as fallback for lap_number == 0 edge cases.

---

### 3. Fastest Lap Time Storage
**Decision**: Add `FastestLapTimeSeconds *float64` to `storage.Session`. The session poller already fetches `/v1/laps` to derive the fastest lap driver (`DeriveFastestLap`); extract the winning driver's best lap duration from the same slice and store it in the Session document at finalization time. The existing `DeriveFastestLap` function is not modified.

**Rationale**: FR-010 requires showing the fastest lap time on Race recap cards. The lap time data is already fetched by the Feature 003 path but not persisted. Storing it in the Session document avoids re-fetching laps at read time.

**Alternatives considered**:
- Fetch laps at API read time (read-through) — rejected: violates constitution principle III (no pass-through at request time)
- Derive from existing `SessionResult.RaceTime` — rejected: `RaceTime` is total race duration, not fastest lap time
- Omit fastest lap time entirely — rejected: FR-010 explicit requirement

**Historical sessions**: The backfill tool (FR-005) handles race_control only. For sessions finalized before this feature ships, `FastestLapTimeSeconds` will be `nil`. The recap card omits the time field if nil (per spec Edge Cases: "fields with no data are omitted").

---

### 4. Where Recap Summary is Computed
**Decision**: Derive the `SessionRecapDTO` server-side in `backend/internal/api/rounds/service.go`, populated into `SessionDetailDTO.RecapSummary`. The frontend receives a fully-computed payload per session.

**Rationale**: Keeps the frontend thin; all derivation logic (winner, gap, cutoff times, event priority) lives in the service layer where it can be unit-tested without DOM dependencies.

**Alternatives considered**:
- Derive on the frontend from raw results — rejected: duplicates business logic, harder to test, and exposes unnecessary data
- New dedicated `/recap` endpoint — rejected: the round detail endpoint already returns all needed data; a new endpoint would add RTT and over-complicate the API surface

---

### 5. Lazy Fill Architecture
**Decision**: Inject a `RaceControlHydrator` interface into the rounds `Service`. Implementations live in `backend/internal/ingest/`. For contract tests, inject a nil hydrator (lazy fill simply skips, recap renders without race-control event line).

**Hydrator contract**:
```go
type RaceControlHydrator interface {
    Hydrate(ctx context.Context, sess storage.Session) (*storage.RaceControlSummary, error)
}
```

The hydrator fetches from OpenF1, persists to Cosmos via `UpsertSession`, and returns the summary. On error, the service logs a warning and returns the response without the event line (graceful degradation per FR-006 Scenario 2).

**Rationale**: Avoids putting HTTP client code directly in the API service layer; preserves testability; keeps OpenF1 integration isolated in the ingest package.

---

### 6. Backfill Repository Access
**Decision**: Add `GetFinalizedSessions(ctx context.Context, season int) ([]storage.Session, error)` to `storage.SessionRepository`. The backfill CLI uses this to fetch all finalized session documents and filter to those missing `RaceControlSummary`.

**Rationale**: `GetFinalizedSessionKeys` (existing) only returns a `map[int]int` — insufficient for the backfill which needs full `Session` structs. A separate query returning full documents is needed.

**Alternatives considered**: Using `GetSessionsByRound` per round — rejected: requires knowing round numbers, which requires a calendar query; simpler to query directly by season + finalized flag.

---

### 7. Q1/Q2 Cutoff Time Derivation
**Decision**: Q1 cutoff = `Q1Time` of the last driver who has a `Q1Time` but `nil Q2Time` (eliminated in Q1). Q2 cutoff = `Q2Time` of the last driver who has a `Q2Time` but `nil Q3Time` (eliminated in Q2). If no such driver exists (format with no elimination, e.g., sprint qualifying), omit the field.

**Rationale**: Results are already sorted by position ascending. Iterating to find the last result with only a Q1 time gives the cutoff for that elimination segment.

---

### 8. Practice "Total Laps"
**Decision**: Sum of `NumberOfLaps` across all driver results for the session.

**Rationale**: In F1, "total laps in practice" is typically reported as the aggregate across all cars (e.g., "FP1: 145 laps completed by all drivers"). This is more informative than any single driver's lap count.

---

### 9. Session Order for the Recap Strip
**Decision**: Chronological ascending by session start time: FP1 → FP2 → FP3 → Sprint Qualifying → Sprint → Qualifying → Race.

**Rationale**: FR-009 explicitly specifies this order. Uses `DateStartUTC` for ordering, same as Cosmos queries (already sorted in `cosmos/sessions.go`).

---

### 10. SchemaVersion Handling
**Decision**: Do NOT bump `SessionSchemaVersion`. Add `RaceControlSummary` and `FastestLapTimeSeconds` as optional fields populated forward (nil for historical docs). The backfill handles historical data for race_control. This avoids triggering a full re-fetch of all finalized sessions' results/drivers/laps.

**Rationale**: Bumping the schema version would cause the poller to re-fetch results, drivers, and laps for all ~70 finalized sessions of the 2026 season — an unnecessary load on OpenF1. The new fields are append-only; old documents simply deserialize with nil values.
