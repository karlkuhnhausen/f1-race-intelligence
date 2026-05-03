# Data Model: Session Recap Strip (Feature 005)

## Storage Layer Changes (`backend/internal/storage/repository.go`)

### New Types

```go
// RaceControlSummary is the aggregated race-control state for a single session.
// Stored as a nested object within the session document in Cosmos DB.
type RaceControlSummary struct {
    RedFlagCount   int            `json:"red_flag_count"`
    SafetyCarCount int            `json:"safety_car_count"`
    VSCCount       int            `json:"vsc_count"`
    NotableEvents  []NotableEvent `json:"notable_events"`
    FetchedAtUTC   time.Time      `json:"fetched_at_utc"`
}

// NotableEvent is a single race-control activation within a session.
type NotableEvent struct {
    // EventType is one of: "red_flag", "safety_car", "vsc", "investigation"
    EventType string `json:"event_type"`
    // LapNumber is the lap on which the first (or only) activation occurred.
    // May be 0 for pre-race events.
    LapNumber int `json:"lap_number"`
    // Count is the number of distinct activations of this event type.
    Count int `json:"count"`
}
```

### `Session` Struct Extensions

Two new optional fields appended to the existing `Session` struct (no migration needed — old documents deserialize with nil values):

```go
// RaceControlSummary is populated at session finalization time and by the
// backfill tool. Nil for sessions finalized before Feature 005 shipped unless
// the backfill has run.
RaceControlSummary *RaceControlSummary `json:"race_control_summary,omitempty"`

// FastestLapTimeSeconds is the duration (in seconds) of the fastest lap set
// in this session. Populated at finalization by the session poller (from the
// /v1/laps data already fetched for DeriveFastestLap). Nil if the session was
// finalized before Feature 005 shipped or if no lap data was available.
FastestLapTimeSeconds *float64 `json:"fastest_lap_time_seconds,omitempty"`
```

### `SessionRepository` Interface Extension

One new method added to the existing interface:

```go
// GetFinalizedSessions returns all session documents for the season where
// finalized=true. Used by the backfill CLI to identify sessions that need
// race-control summary population.
GetFinalizedSessions(ctx context.Context, season int) ([]Session, error)
```

---

## API Layer Types (`backend/internal/api/rounds/dto.go`)

### New Types

```go
// NotableEventDTO is the API representation of a single notable race-control event.
type NotableEventDTO struct {
    // EventType is one of: "red_flag", "safety_car", "vsc", "investigation"
    EventType string `json:"event_type"`
    // LapNumber is the lap on which the first activation occurred (0 for pre-race).
    LapNumber int `json:"lap_number"`
    // Count is the number of distinct activations of this event type in the session.
    Count int `json:"count"`
}

// SessionRecapDTO is the pre-computed summary payload for one session.
// Fields present depend on session type (race/sprint, qualifying/sprint-qualifying,
// practice). Absent fields are omitted from the JSON response.
//
// Race / Sprint fields: WinnerName, WinnerTeam, GapToP2, FastestLapHolder,
//   FastestLapTeam, FastestLapTimeSeconds, TotalLaps, plus event fields.
//
// Qualifying / Sprint Qualifying fields: PoleSitterName, PoleSitterTeam,
//   PoleTime, GapToP2, Q1CutoffTime, Q2CutoffTime, plus event fields.
//
// Practice fields: BestDriverName, BestDriverTeam, BestLapTime,
//   TotalLaps, plus event fields.
//
// Event fields (all session types): RedFlagCount, SafetyCarCount, VSCCount,
//   TopEvent.
type SessionRecapDTO struct {
    // --- Race / Sprint ---

    // WinnerName is the P1 classified finisher's full name.
    WinnerName string `json:"winner_name,omitempty"`
    // WinnerTeam is the winner's team name (used for team color lookup on frontend).
    WinnerTeam string `json:"winner_team,omitempty"`
    // GapToP2 is the gap from P1 to P2, as a formatted string (e.g., "+5.132" or "+1 LAP").
    // Empty string if fewer than two classified finishers.
    GapToP2 string `json:"gap_to_p2,omitempty"`
    // FastestLapHolder is the driver name who set the fastest lap.
    FastestLapHolder string `json:"fastest_lap_holder,omitempty"`
    // FastestLapTeam is the team name of the fastest lap holder.
    FastestLapTeam string `json:"fastest_lap_team,omitempty"`
    // FastestLapTimeSeconds is the fastest lap duration in seconds.
    // Omitted if not available (e.g., sessions finalized before Feature 005).
    FastestLapTimeSeconds *float64 `json:"fastest_lap_time_seconds,omitempty"`
    // TotalLaps is the number of laps completed in the session.
    // For race: winner's NumberOfLaps. For practice: sum of all drivers' laps.
    TotalLaps int `json:"total_laps,omitempty"`

    // --- Qualifying / Sprint Qualifying ---

    // PoleSitterName is the P1 qualifier's full name.
    PoleSitterName string `json:"pole_sitter_name,omitempty"`
    // PoleSitterTeam is the P1 qualifier's team name.
    PoleSitterTeam string `json:"pole_sitter_team,omitempty"`
    // PoleTime is the pole lap time in seconds (Q3 time for standard qualifying).
    PoleTime *float64 `json:"pole_time,omitempty"`
    // GapToP2 is shared with race: gap from pole to P2 qualifying time (seconds, as formatted string).
    // Q1CutoffTime is the Q1 elimination threshold time (seconds).
    // Omitted for session formats without Q1 elimination (e.g., Sprint Qualifying).
    Q1CutoffTime *float64 `json:"q1_cutoff_time,omitempty"`
    // Q2CutoffTime is the Q2 elimination threshold time (seconds).
    // Omitted for session formats without Q2 elimination.
    Q2CutoffTime *float64 `json:"q2_cutoff_time,omitempty"`

    // --- Practice ---

    // BestDriverName is the driver with the best lap time in the practice session.
    BestDriverName string `json:"best_driver_name,omitempty"`
    // BestDriverTeam is the best driver's team name.
    BestDriverTeam string `json:"best_driver_team,omitempty"`
    // BestLapTime is the session best lap time in seconds (same as P1 result's BestLapTime).
    BestLapTime *float64 `json:"best_lap_time,omitempty"`

    // --- All sessions with race-control data ---

    // RedFlagCount is the number of distinct red flag activations.
    RedFlagCount int `json:"red_flag_count,omitempty"`
    // SafetyCarCount is the number of distinct safety car deployments.
    SafetyCarCount int `json:"safety_car_count,omitempty"`
    // VSCCount is the number of distinct virtual safety car deployments.
    VSCCount int `json:"vsc_count,omitempty"`
    // TopEvent is the single highest-priority notable event, if any occurred.
    // Priority: red_flag > safety_car > vsc > investigation.
    TopEvent *NotableEventDTO `json:"top_event,omitempty"`
}
```

### `SessionDetailDTO` Extension

Add one field to the existing struct:

```go
// RecapSummary is populated for completed sessions only. Nil for sessions
// that are upcoming or in-progress, or completed sessions missing race-control
// data when graceful degradation applies.
RecapSummary *SessionRecapDTO `json:"recap_summary,omitempty"`
```

---

## Ingest Layer Types (`backend/internal/ingest/race_control.go`)

### Internal OpenF1 Shape

```go
// openF1RaceControlMsg is the raw upstream shape from OpenF1 /v1/race_control.
type openF1RaceControlMsg struct {
    Category  string  `json:"category"`
    Flag      string  `json:"flag"`
    LapNumber int     `json:"lap_number"`
    Message   string  `json:"message"`
    SessionKey int    `json:"session_key"`
    Date      string  `json:"date"` // ISO-8601; used for time-proximity dedup of lap_number=0 events
}
```

---

## Cosmos DB Document Impact

No schema migration or new container required. The `sessions` container (partition key `season`) accommodates the new fields as append-only nested objects. Old finalized documents (without `race_control_summary`) deserialize with `RaceControlSummary == nil`, which the rounds service handles gracefully (recap rendered without event line).

**Document ID format**: Unchanged — `{year}-{round:02d}-{session_type_slug}` (e.g., `2026-01-race`).

---

## Entity Relationships

```
storage.Session (1)
  └── storage.RaceControlSummary (0..1)  — embedded
        └── storage.NotableEvent (0..N)  — ordered list

storage.Session (1)  ←→  storage.SessionResult (0..N)  — linked by season+round+session_type

rounds.SessionDetailDTO (1)
  ├── rounds.SessionResultDTO (0..N)
  └── rounds.SessionRecapDTO (0..1)      — derived at read time from Session + SessionResults
        └── rounds.NotableEventDTO (0..1) — top event only (highest priority)
```

---

## Event Type Enum Values

| Domain value | JSON value | Priority |
|---|---|---|
| Red Flag | `"red_flag"` | 1 (highest) |
| Safety Car | `"safety_car"` | 2 |
| Virtual Safety Car | `"vsc"` | 3 |
| Investigation | `"investigation"` | 4 (lowest) |
