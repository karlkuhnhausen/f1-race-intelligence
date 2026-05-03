# Data Model: Session Deep Dive Page (Feature 006)

## Storage Layer (`backend/internal/storage/repository.go`)

### New Interface: `AnalysisRepository`

```go
// AnalysisRepository defines read/write operations for session analysis data.
// Analysis data is stored as per-driver batch documents in the sessions container.
type AnalysisRepository interface {
    // UpsertSessionPositions persists aggregated position data for all drivers in a session.
    // Each driver gets one document containing all their lap positions.
    // Idempotent: re-upserting overwrites existing data (same document IDs).
    UpsertSessionPositions(ctx context.Context, season, round int, sessionType string, positions []SessionPosition) error

    // UpsertSessionIntervals persists gap-to-leader data for all drivers in a session.
    UpsertSessionIntervals(ctx context.Context, season, round int, sessionType string, intervals []SessionInterval) error

    // UpsertSessionStints persists tire stint data for all drivers in a session.
    UpsertSessionStints(ctx context.Context, season, round int, sessionType string, stints []SessionStint) error

    // UpsertSessionPits persists pit stop data for all drivers in a session.
    UpsertSessionPits(ctx context.Context, season, round int, sessionType string, pits []SessionPit) error

    // UpsertSessionOvertakes persists overtake events for a session.
    // Stored as a single document per session (not per-driver).
    UpsertSessionOvertakes(ctx context.Context, season, round int, sessionType string, overtakes []SessionOvertake) error

    // GetSessionAnalysis retrieves all analysis data for a given session.
    // Returns nil (not error) if no analysis data exists for the session.
    GetSessionAnalysis(ctx context.Context, season, round int, sessionType string) (*SessionAnalysisData, error)

    // HasAnalysisData checks whether analysis data has been ingested for a session.
    // Used by the backfill CLI to skip already-populated sessions (idempotency).
    HasAnalysisData(ctx context.Context, season, round int, sessionType string) (bool, error)
}
```

### New Types

```go
// SessionPosition stores aggregated position data for one driver in one session.
// One Cosmos document per driver per session.
type SessionPosition struct {
    ID           string         `json:"id"`            // analysis_position_{round}_{sessiontype}_{drivernum}
    Type         string         `json:"type"`          // "analysis_position"
    Season       int            `json:"season"`        // partition key
    Round        int            `json:"round"`
    SessionType  string         `json:"session_type"`  // "race" or "sprint"
    DriverNumber int            `json:"driver_number"`
    DriverName   string         `json:"driver_name"`
    DriverAcronym string        `json:"driver_acronym"`
    TeamName     string         `json:"team_name"`
    Laps         []PositionLap  `json:"laps"`          // one entry per completed lap
}

// PositionLap is a single lap's position for a driver.
type PositionLap struct {
    LapNumber int `json:"lap_number"`
    Position  int `json:"position"`
}

// SessionInterval stores gap-to-leader data for one driver in one session.
type SessionInterval struct {
    ID           string          `json:"id"`           // analysis_interval_{round}_{sessiontype}_{drivernum}
    Type         string          `json:"type"`         // "analysis_interval"
    Season       int             `json:"season"`
    Round        int             `json:"round"`
    SessionType  string          `json:"session_type"`
    DriverNumber int             `json:"driver_number"`
    DriverAcronym string         `json:"driver_acronym"`
    TeamName     string          `json:"team_name"`
    Laps         []IntervalLap   `json:"laps"`
}

// IntervalLap is a single lap's gap data for a driver.
type IntervalLap struct {
    LapNumber   int     `json:"lap_number"`
    GapToLeader float64 `json:"gap_to_leader"` // seconds; 0 for leader
    Interval    float64 `json:"interval"`      // gap to car ahead; 0 for leader
}

// SessionStint stores one tire stint for one driver.
// Multiple documents per driver per session (one per stint).
type SessionStint struct {
    ID             string `json:"id"`           // analysis_stint_{round}_{sessiontype}_{drivernum}_{stintnumber}
    Type           string `json:"type"`         // "analysis_stint"
    Season         int    `json:"season"`
    Round          int    `json:"round"`
    SessionType    string `json:"session_type"`
    DriverNumber   int    `json:"driver_number"`
    DriverAcronym  string `json:"driver_acronym"`
    TeamName       string `json:"team_name"`
    StintNumber    int    `json:"stint_number"` // 1, 2, 3...
    Compound       string `json:"compound"`     // SOFT, MEDIUM, HARD, INTERMEDIATE, WET
    LapStart       int    `json:"lap_start"`
    LapEnd         int    `json:"lap_end"`
    TyreAgeAtStart int    `json:"tyre_age_at_start"`
}

// SessionPit stores one pit stop event for one driver.
type SessionPit struct {
    ID            string  `json:"id"`           // analysis_pit_{round}_{sessiontype}_{drivernum}_{lapnumber}
    Type          string  `json:"type"`         // "analysis_pit"
    Season        int     `json:"season"`
    Round         int     `json:"round"`
    SessionType   string  `json:"session_type"`
    DriverNumber  int     `json:"driver_number"`
    DriverAcronym string  `json:"driver_acronym"`
    TeamName      string  `json:"team_name"`
    LapNumber     int     `json:"lap_number"`
    PitDuration   float64 `json:"pit_duration"`  // total pit lane time in seconds
    StopDuration  float64 `json:"stop_duration"` // stationary time in seconds (0 if unavailable)
}

// SessionOvertake stores one overtake event.
type SessionOvertake struct {
    ID                      string `json:"id"`   // analysis_overtake_{round}_{sessiontype}_{index}
    Type                    string `json:"type"` // "analysis_overtake"
    Season                  int    `json:"season"`
    Round                   int    `json:"round"`
    SessionType             string `json:"session_type"`
    OvertakingDriverNumber  int    `json:"overtaking_driver_number"`
    OvertakingDriverName    string `json:"overtaking_driver_name"`
    OvertakenDriverNumber   int    `json:"overtaken_driver_number"`
    OvertakenDriverName     string `json:"overtaken_driver_name"`
    LapNumber               int    `json:"lap_number"`
    Position                int    `json:"position"` // resulting position of overtaking driver
}

// SessionAnalysisData is the combined query result for all analysis data in a session.
// Used by the API service to build the response DTO.
type SessionAnalysisData struct {
    Positions  []SessionPosition  `json:"positions"`
    Intervals  []SessionInterval  `json:"intervals"`
    Stints     []SessionStint     `json:"stints"`
    Pits       []SessionPit       `json:"pits"`
    Overtakes  []SessionOvertake  `json:"overtakes"`
}
```

### Document ID Examples

| Data Type | Document ID Pattern | Example |
|-----------|-------------------|---------|
| Position | `analysis_position_{round}_{type}_{driver}` | `analysis_position_4_race_1` |
| Interval | `analysis_interval_{round}_{type}_{driver}` | `analysis_interval_4_race_44` |
| Stint | `analysis_stint_{round}_{type}_{driver}_{stint}` | `analysis_stint_4_race_44_2` |
| Pit | `analysis_pit_{round}_{type}_{driver}_{lap}` | `analysis_pit_4_race_44_18` |
| Overtake | `analysis_overtake_{round}_{type}_{index}` | `analysis_overtake_4_race_7` |

### Cosmos Query Patterns

**Get all analysis data for a session** (single partition query):
```sql
SELECT * FROM c
WHERE c.season = @season
  AND c.round = @round
  AND c.session_type = @sessionType
  AND STARTSWITH(c.type, 'analysis_')
```

**Check if analysis data exists** (cheap existence check):
```sql
SELECT VALUE COUNT(1) FROM c
WHERE c.season = @season
  AND c.round = @round
  AND c.session_type = @sessionType
  AND c.type = 'analysis_position'
```

---

## Domain Layer (`backend/internal/domain/analysis.go`)

Domain types used across ingest, storage, and API layers. These are the pure domain representations without Cosmos-specific fields (ID, Type, Season, etc.).

```go
package domain

// AnalysisPosition is the domain representation of a driver's position data.
type AnalysisPosition struct {
    DriverNumber  int
    DriverName    string
    DriverAcronym string
    TeamName      string
    Laps          []PositionLap
}

type PositionLap struct {
    LapNumber int
    Position  int
}

// AnalysisInterval is the domain representation of a driver's gap data.
type AnalysisInterval struct {
    DriverNumber  int
    DriverAcronym string
    TeamName      string
    Laps          []IntervalLap
}

type IntervalLap struct {
    LapNumber   int
    GapToLeader float64
    Interval    float64
}

// AnalysisStint is the domain representation of one tire stint.
type AnalysisStint struct {
    DriverNumber   int
    DriverAcronym  string
    TeamName       string
    StintNumber    int
    Compound       string // SOFT, MEDIUM, HARD, INTERMEDIATE, WET
    LapStart       int
    LapEnd         int
    TyreAgeAtStart int
}

// AnalysisPit is the domain representation of one pit stop.
type AnalysisPit struct {
    DriverNumber  int
    DriverAcronym string
    TeamName      string
    LapNumber     int
    PitDuration   float64 // total pit lane time (seconds)
    StopDuration  float64 // stationary time (seconds); 0 if unavailable
}

// AnalysisOvertake is the domain representation of one overtake event.
type AnalysisOvertake struct {
    OvertakingDriverNumber int
    OvertakingDriverName   string
    OvertakenDriverNumber  int
    OvertakenDriverName    string
    LapNumber              int
    Position               int
}

// AnalysisData is the combined result of all analysis data for a session.
type AnalysisData struct {
    Positions  []AnalysisPosition
    Intervals  []AnalysisInterval
    Stints     []AnalysisStint
    Pits       []AnalysisPit
    Overtakes  []AnalysisOvertake
}
```

---

## API Layer (`backend/internal/api/analysis/dto.go`)

### Response Types

```go
package analysis

// SessionAnalysisDTO is the top-level response for GET /api/v1/rounds/{round}/sessions/{type}/analysis.
type SessionAnalysisDTO struct {
    Year        int                  `json:"year"`
    Round       int                  `json:"round"`
    SessionType string               `json:"session_type"`
    TotalLaps   int                  `json:"total_laps"`
    Positions   []PositionDriverDTO  `json:"positions"`
    Intervals   []IntervalDriverDTO  `json:"intervals,omitempty"`
    Stints      []StintDTO           `json:"stints,omitempty"`
    Pits        []PitDTO             `json:"pits,omitempty"`
    Overtakes   []OvertakeDTO        `json:"overtakes,omitempty"`
}

// PositionDriverDTO contains position data for one driver.
type PositionDriverDTO struct {
    DriverNumber  int              `json:"driver_number"`
    DriverName    string           `json:"driver_name"`
    DriverAcronym string           `json:"driver_acronym"`
    TeamName      string           `json:"team_name"`
    Laps          []PositionLapDTO `json:"laps"`
}

// PositionLapDTO is one lap's position for a driver.
type PositionLapDTO struct {
    Lap      int `json:"lap"`
    Position int `json:"position"`
}

// IntervalDriverDTO contains gap data for one driver.
type IntervalDriverDTO struct {
    DriverNumber  int              `json:"driver_number"`
    DriverAcronym string           `json:"driver_acronym"`
    TeamName      string           `json:"team_name"`
    Laps          []IntervalLapDTO `json:"laps"`
}

// IntervalLapDTO is one lap's gap data for a driver.
type IntervalLapDTO struct {
    Lap         int     `json:"lap"`
    GapToLeader float64 `json:"gap_to_leader"`
    Interval    float64 `json:"interval"`
}

// StintDTO is one tire stint for one driver.
type StintDTO struct {
    DriverNumber   int    `json:"driver_number"`
    DriverAcronym  string `json:"driver_acronym"`
    TeamName       string `json:"team_name"`
    StintNumber    int    `json:"stint_number"`
    Compound       string `json:"compound"`
    LapStart       int    `json:"lap_start"`
    LapEnd         int    `json:"lap_end"`
    TyreAgeAtStart int    `json:"tyre_age_at_start"`
}

// PitDTO is one pit stop for one driver.
type PitDTO struct {
    DriverNumber  int     `json:"driver_number"`
    DriverAcronym string  `json:"driver_acronym"`
    TeamName      string  `json:"team_name"`
    Lap           int     `json:"lap"`
    PitDuration   float64 `json:"pit_duration"`
    StopDuration  float64 `json:"stop_duration,omitempty"`
}

// OvertakeDTO is one overtake event.
type OvertakeDTO struct {
    OvertakingDriverNumber int    `json:"overtaking_driver_number"`
    OvertakingDriverName   string `json:"overtaking_driver_name"`
    OvertakenDriverNumber  int    `json:"overtaken_driver_number"`
    OvertakenDriverName    string `json:"overtaken_driver_name"`
    Lap                    int    `json:"lap"`
    Position               int    `json:"position"`
}
```

---

## Frontend Types (`frontend/src/features/analysis/analysisTypes.ts`)

```typescript
export interface PositionLap {
  lap: number;
  position: number;
}

export interface PositionDriver {
  driver_number: number;
  driver_name: string;
  driver_acronym: string;
  team_name: string;
  laps: PositionLap[];
}

export interface IntervalLap {
  lap: number;
  gap_to_leader: number;
  interval: number;
}

export interface IntervalDriver {
  driver_number: number;
  driver_acronym: string;
  team_name: string;
  laps: IntervalLap[];
}

export interface Stint {
  driver_number: number;
  driver_acronym: string;
  team_name: string;
  stint_number: number;
  compound: 'SOFT' | 'MEDIUM' | 'HARD' | 'INTERMEDIATE' | 'WET';
  lap_start: number;
  lap_end: number;
  tyre_age_at_start: number;
}

export interface PitStop {
  driver_number: number;
  driver_acronym: string;
  team_name: string;
  lap: number;
  pit_duration: number;
  stop_duration?: number;
}

export interface Overtake {
  overtaking_driver_number: number;
  overtaking_driver_name: string;
  overtaken_driver_number: number;
  overtaken_driver_name: string;
  lap: number;
  position: number;
}

export interface SessionAnalysisResponse {
  year: number;
  round: number;
  session_type: string;
  total_laps: number;
  positions: PositionDriver[];
  intervals?: IntervalDriver[];
  stints?: Stint[];
  pits?: PitStop[];
  overtakes?: Overtake[];
}
```

---

## Sizing Estimates

| Data Type | Docs per Session | Avg Doc Size | Total per Session |
|-----------|-----------------|--------------|-------------------|
| Position | 20 (one per driver) | ~3KB (60 laps × 2 fields) | ~60KB |
| Interval | 20 | ~4KB (60 laps × 3 fields) | ~80KB |
| Stint | ~60 (20 drivers × 3 avg stints) | ~0.3KB | ~18KB |
| Pit | ~40 (20 drivers × 2 avg stops) | ~0.3KB | ~12KB |
| Overtake | ~30-50 events | ~0.3KB | ~15KB |
| **Total** | **~170 documents** | | **~185KB** |

API response size (JSON): ~120KB uncompressed, ~15KB gzipped. Well within performance goals.
