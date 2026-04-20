# Data Model: Race Results & Session Data

## Entity: Session

- Purpose: Represents one session within a race weekend (FP1, FP2, FP3, Sprint Qualifying, Sprint, Qualifying, Race), storing metadata and availability status.
- Primary key: `id` (format: `{season}-{round:02d}-{session_type_slug}`, e.g., `2026-05-race`)
- Partition key: `season`
- Document type discriminator: `type: "session"`

Fields:
- `id` (string, required) — Cosmos DB document ID
- `season` (int, required, example `2026`)
- `round` (int, required, 1..24)
- `meeting_key` (int, required) — OpenF1 meeting identifier
- `session_key` (int, required) — OpenF1 session identifier
- `session_name` (string, required) — Human-readable name, e.g., "Practice 1", "Qualifying", "Race"
- `session_type` (enum, required) — One of: `practice1`, `practice2`, `practice3`, `sprint_qualifying`, `sprint`, `qualifying`, `race`
- `status` (enum, required) — One of: `completed`, `in_progress`, `upcoming`, `not_available`
- `date_start_utc` (string datetime, required) — Session scheduled start
- `date_end_utc` (string datetime, required) — Session scheduled end
- `data_as_of_utc` (string datetime, required) — Last successful ingestion timestamp
- `source` (string, required, default `openf1`)

Validation rules:
- `session_type` must be one of the defined enum values.
- `session_key` must be unique within a season.
- `status` transitions: `upcoming` → `in_progress` → `completed`. `not_available` is terminal for sessions that were expected but have no data.

Session type slug mapping (for document IDs):

| OpenF1 session_name | session_type slug | session_type enum |
|---|---|---|
| Practice 1 | `fp1` | `practice1` |
| Practice 2 | `fp2` | `practice2` |
| Practice 3 | `fp3` | `practice3` |
| Sprint Qualifying | `sprint-qualifying` | `sprint_qualifying` |
| Sprint | `sprint` | `sprint` |
| Qualifying | `qualifying` | `qualifying` |
| Race | `race` | `race` |

## Entity: SessionResult

- Purpose: Represents one driver's finalized result within a session.
- Primary key: `id` (format: `{season}-{round:02d}-{session_type_slug}-{driver_number}`, e.g., `2026-05-race-44`)
- Partition key: `season`
- Document type discriminator: `type: "session_result"`

Fields (common to all session types):
- `id` (string, required) — Cosmos DB document ID
- `season` (int, required)
- `round` (int, required)
- `session_key` (int, required) — OpenF1 session identifier
- `session_type` (enum, required) — Same enum as Session
- `position` (int, required, >=1) — Final position in session
- `driver_number` (int, required) — FIA car number
- `driver_name` (string, required) — Full name, e.g., "Max VERSTAPPEN"
- `driver_acronym` (string, required) — Three-letter, e.g., "VER"
- `team_name` (string, required)
- `number_of_laps` (int, required, >=0) — Laps completed in session
- `data_as_of_utc` (string datetime, required)
- `source` (string, required, default `openf1`)

Fields (race-specific):
- `finishing_status` (enum, optional) — One of: `Finished`, `DNF`, `DNS`, `DSQ`. Present only for race and sprint sessions.
- `race_time` (number, optional) — Total race time in seconds for P1; null for others.
- `gap_to_leader` (string, optional) — Time gap as string (e.g., "+5.123s" or "+1 LAP"). Null for P1.
- `points` (number, optional, >=0) — Points scored in this session.
- `fastest_lap` (boolean, optional) — True if this driver set the fastest race lap.

Fields (qualifying-specific):
- `q1_time` (number, optional) — Best Q1 lap time in seconds. Null if did not participate.
- `q2_time` (number, optional) — Best Q2 lap time in seconds. Null if eliminated in Q1.
- `q3_time` (number, optional) — Best Q3 lap time in seconds. Null if eliminated in Q2 or earlier.

Fields (practice-specific):
- `best_lap_time` (number, optional) — Best lap time in seconds.
- `gap_to_fastest` (number, optional) — Gap to session fastest in seconds. 0 for P1.

Validation rules:
- `position` must be unique per session (per season + round + session_type).
- For race sessions: `finishing_status` must be non-null. If `finishing_status` is `DNF` or `DNS`, `gap_to_leader` should reflect the status.
- For qualifying: at least `q1_time` should be non-null for all classified drivers.
- For practice: `best_lap_time` should be non-null for drivers who set a time.

## Entity: Session (extended from existing RaceMeeting)

No changes to the existing `RaceMeeting` entity. The `meeting_key` stored in `RaceMeeting.SourceHash` links meetings to their sessions.

## Relationship Notes

- `RaceMeeting` (1) → `Session` (many): A race meeting has multiple sessions. Linked by `season` + `round`.
- `Session` (1) → `SessionResult` (many): A session has results for each driver. Linked by `season` + `round` + `session_type`.
- `Session.session_key` links to OpenF1 upstream data for refresh.
- `SessionResult` denormalizes `driver_name` and `team_name` from the OpenF1 `/v1/drivers` endpoint to avoid join queries.
- All entities share the `season` partition key enabling efficient single-partition queries for a round detail page.

## Query Patterns

1. **Round detail page load**: Query all `Session` documents where `season={year}` AND `round={round}` → returns session metadata. Then query all `SessionResult` documents where `season={year}` AND `round={round}` → returns all results. Both queries execute within the same `season` partition.

2. **Session ingestion upsert**: For each polled session, upsert `Session` document by `id`. For each driver result, upsert `SessionResult` document by `id`. Upsert semantics ensure later ingestion overwrites earlier data (spec edge case: corrections).

## Points Lookup Table (2026 FIA standard)

| Position | Points |
|---|---|
| 1 | 25 |
| 2 | 18 |
| 3 | 15 |
| 4 | 12 |
| 5 | 10 |
| 6 | 8 |
| 7 | 6 |
| 8 | 4 |
| 9 | 2 |
| 10 | 1 |
| Fastest lap (P1–P10) | +1 |

Sprint race points:
| Position | Points |
|---|---|
| 1 | 8 |
| 2 | 7 |
| 3 | 6 |
| 4 | 5 |
| 5 | 4 |
| 6 | 3 |
| 7 | 2 |
| 8 | 1 |
