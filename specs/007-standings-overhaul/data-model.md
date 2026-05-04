# Data Model: Standings Overhaul

## Entity Relationship Overview

```
┌─────────────────────────────┐       ┌─────────────────────────────┐
│  DriverChampionshipSnapshot │       │  TeamChampionshipSnapshot   │
│  (per driver per race)      │       │  (per team per race)        │
├─────────────────────────────┤       ├─────────────────────────────┤
│  id (PK)                    │       │  id (PK)                    │
│  season (partition key)     │       │  season (partition key)     │
│  session_key                │       │  session_key                │
│  meeting_key                │       │  meeting_key                │
│  driver_number              │       │  team_name                  │
│  position_start             │       │  position_start             │
│  position_current           │       │  position_current           │
│  points_start               │       │  points_start               │
│  points_current             │       │  points_current             │
│  data_as_of_utc             │       │  data_as_of_utc             │
│  source                     │       │  source                     │
└──────────────┬──────────────┘       └─────────────────────────────┘
               │
               │ joins via driver_number + meeting_key
               ▼
┌─────────────────────────────┐
│  Driver (already exists)    │
├─────────────────────────────┤
│  driver_number              │
│  full_name                  │
│  team_name                  │
│  team_colour (hex)          │
│  name_acronym               │
│  session_key / meeting_key  │
└─────────────────────────────┘

┌─────────────────────────────┐       ┌─────────────────────────────┐
│  SessionResult              │       │  StartingGrid               │
│  (per driver per session)   │       │  (per driver per meeting)   │
├─────────────────────────────┤       ├─────────────────────────────┤
│  id (PK)                    │       │  id (PK)                    │
│  season (partition key)     │       │  season (partition key)     │
│  session_key                │       │  meeting_key                │
│  meeting_key                │       │  driver_number              │
│  driver_number              │       │  position                   │
│  position                   │       │  lap_duration               │
│  points                     │       │  data_as_of_utc             │
│  dnf                        │       │  source                     │
│  dns                        │       └─────────────────────────────┘
│  dsq                        │
│  number_of_laps             │
│  gap_to_leader              │
│  duration                   │
│  data_as_of_utc             │
│  source                     │
└─────────────────────────────┘
```

## Entity Definitions

### DriverChampionshipSnapshot

Represents a driver's championship standing at a specific point in the season (after a Race or Sprint session).

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | yes | Document ID: `{season}-champ-driver-{session_key}-{driver_number}` |
| `season` | int | yes | Championship year (partition key) |
| `session_key` | int | yes | OpenF1 session identifier |
| `meeting_key` | int | yes | OpenF1 meeting identifier |
| `driver_number` | int | yes | Driver's car number |
| `position_start` | int (nullable) | yes | Championship position before the session (null for first race of season) |
| `position_current` | int | yes | Championship position after the session |
| `points_start` | float64 (nullable) | yes | Championship points before the session (null for first race) |
| `points_current` | float64 | yes | Championship points after the session |
| `data_as_of_utc` | datetime | yes | Timestamp when this data was ingested |
| `source` | string | yes | Always `"openf1"` |
| `type` | string | yes | Discriminator: `"championship_driver"` |

**Validation rules**:
- `position_current` ≥ 1
- `points_current` ≥ 0
- `driver_number` > 0
- `session_key` > 0
- `season` in range [2023, current_year]

**State transitions**: Immutable once written (represents a historical snapshot). May be overwritten if OpenF1 data is corrected post-race.

---

### TeamChampionshipSnapshot

Represents a constructor team's championship standing at a specific point in the season.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | yes | Document ID: `{season}-champ-team-{session_key}-{team_slug}` |
| `season` | int | yes | Championship year (partition key) |
| `session_key` | int | yes | OpenF1 session identifier |
| `meeting_key` | int | yes | OpenF1 meeting identifier |
| `team_name` | string | yes | Team name (e.g., "McLaren", "Red Bull Racing") |
| `team_slug` | string | yes | URL-safe slug (e.g., "mclaren", "red-bull-racing") |
| `position_start` | int (nullable) | yes | Championship position before the session |
| `position_current` | int | yes | Championship position after the session |
| `points_start` | float64 (nullable) | yes | Championship points before the session |
| `points_current` | float64 | yes | Championship points after the session |
| `data_as_of_utc` | datetime | yes | Timestamp when this data was ingested |
| `source` | string | yes | Always `"openf1"` |
| `type` | string | yes | Discriminator: `"championship_team"` |

**Validation rules**:
- `team_name` non-empty string
- `position_current` ≥ 1
- `points_current` ≥ 0

**State transitions**: Immutable once written. Overwritten on data correction.

---

### SessionResult

Represents a single driver's result in a completed Race or Sprint session. Used to derive wins, podiums, and DNFs.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | yes | Document ID: `{season}-result-{session_key}-{driver_number}` |
| `season` | int | yes | Championship year (partition key) |
| `session_key` | int | yes | OpenF1 session identifier |
| `meeting_key` | int | yes | OpenF1 meeting identifier |
| `driver_number` | int | yes | Driver's car number |
| `position` | int | yes | Finishing position (1-based) |
| `points` | float64 | yes | Points awarded in this session |
| `dnf` | bool | yes | Did Not Finish |
| `dns` | bool | yes | Did Not Start |
| `dsq` | bool | yes | Disqualified |
| `number_of_laps` | int | yes | Laps completed |
| `gap_to_leader` | float64/string | yes | Time gap to P1 in seconds, or "+N LAP(S)" |
| `duration` | float64 | yes | Total race time in seconds |
| `data_as_of_utc` | datetime | yes | Timestamp when ingested |
| `source` | string | yes | Always `"openf1"` |
| `type` | string | yes | Discriminator: `"session_result"` |

**Validation rules**:
- `position` ≥ 1
- `points` ≥ 0
- At most one of `dnf`, `dns`, `dsq` can be true

---

### StartingGrid

Represents a driver's starting position for a race meeting. Used to derive pole positions.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | yes | Document ID: `{season}-grid-{meeting_key}-{driver_number}` |
| `season` | int | yes | Championship year (partition key) |
| `meeting_key` | int | yes | OpenF1 meeting identifier |
| `driver_number` | int | yes | Driver's car number |
| `position` | int | yes | Grid position (1 = pole) |
| `lap_duration` | float64 | yes | Qualifying lap time in seconds |
| `data_as_of_utc` | datetime | yes | Timestamp when ingested |
| `source` | string | yes | Always `"openf1"` |
| `type` | string | yes | Discriminator: `"starting_grid"` |

**Validation rules**:
- `position` ≥ 1
- `lap_duration` > 0

---

## Aggregated Views (computed at query time, not stored)

### DriverStandingsRow (API response)

Computed by joining `DriverChampionshipSnapshot` (latest session in season) + session results + starting grids:

| Field | Type | Description |
|-------|------|-------------|
| `position` | int | Current championship position |
| `driver_name` | string | Full name (from driver identity) |
| `driver_number` | int | Car number |
| `team_name` | string | Current team |
| `team_color` | string | Team hex color |
| `points` | float64 | Current championship points |
| `wins` | int | Count of sessions where position = 1 |
| `podiums` | int | Count of sessions where position ≤ 3 |
| `dnfs` | int | Count of sessions where dnf = true |
| `poles` | int | Count of meetings where starting grid position = 1 |

### ConstructorStandingsRow (API response)

Computed by joining `TeamChampionshipSnapshot` (latest session in season) + aggregated driver results:

| Field | Type | Description |
|-------|------|-------------|
| `position` | int | Current championship position |
| `team_name` | string | Team name |
| `team_color` | string | Team hex color |
| `points` | float64 | Current championship points |
| `wins` | int | Combined driver wins |
| `podiums` | int | Combined driver podiums |
| `dnfs` | int | Combined driver DNFs |

### ProgressionPoint (API response, array per competitor)

One entry per race session in chronological order:

| Field | Type | Description |
|-------|------|-------------|
| `session_key` | int | Race session identifier |
| `round_name` | string | Circuit/race name for x-axis label |
| `points` | float64 | Cumulative points after this race |
| `position` | int | Championship position after this race |

---

## Cosmos DB Container Layout

| Container | Partition Key | Document Types |
|-----------|---------------|----------------|
| `standings` | `/season` | `championship_driver`, `championship_team` |
| `sessions` | `/season` | `session_result`, `starting_grid` (alongside existing session documents) |

**Note**: Session results and starting grids are stored in the `sessions` container because they are per-session data and the container already uses `/season` as partition key. Championship snapshots stay in `standings` container to keep backward compatibility with existing API queries.

---

## Migration from Hyprace Schema

| Old Format | New Format | Action |
|------------|------------|--------|
| `{season}-driver-{position}` | `{season}-champ-driver-{session_key}-{driver_number}` | Old documents are dead (zero rows exist in production). No migration needed. |
| `{season}-constructor-{position}` | `{season}-champ-team-{session_key}-{team_slug}` | Same — zero rows exist. |
| `source = "hyprace"` filter in queries | `type = "championship_driver"` discriminator | Update queries to use type discriminator instead of source filter. |
