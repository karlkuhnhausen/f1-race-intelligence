# Data Model: Race Calendar and Championship Standings

## Entity: RaceMeeting

- Purpose: Normalized representation of one F1 season round.
- Primary key: `id` (format: `season-round`, e.g., `2026-04`)
- Partition key: `season`

Fields:
- `id` (string, required)
- `season` (int, required, example `2026`)
- `round` (int, required, 1..24)
- `race_name` (string, required)
- `circuit_name` (string, required)
- `country_name` (string, required)
- `start_datetime_utc` (string datetime, required)
- `end_datetime_utc` (string datetime, required)
- `status` (enum: `scheduled|cancelled|completed|unknown`, required)
- `is_cancelled` (boolean, required)
- `cancelled_label` (string, optional)
- `cancelled_reason` (string, optional)
- `source` (string, required, default `openf1`)
- `data_as_of_utc` (string datetime, required)
- `source_hash` (string, required)

Validation rules:
- `round` must be unique within season.
- For Bahrain R4 and Saudi Arabia R5, `status` must be `cancelled`.
- If `is_cancelled=true`, `cancelled_label` must be non-empty.

## Entity: UpcomingRaceSnapshot

- Purpose: Computed next-race context used by frontend countdown.
- Primary key: `id` (format: `season-next`)
- Partition key: `season`

Fields:
- `id` (string, required)
- `season` (int, required)
- `next_round` (int, required)
- `countdown_target_utc` (string datetime, required)
- `generated_at_utc` (string datetime, required)
- `excluded_rounds` (array of object, optional)

Validation rules:
- `next_round` must reference a non-cancelled round with start >= now.

## Entity: DriverStandingRow

- Purpose: Drivers championship row exposed to frontend.
- Primary key: `id` (format: `season-driver-position`)
- Partition key: `season`

Fields:
- `id` (string, required)
- `season` (int, required)
- `position` (int, required, >=1)
- `driver_name` (string, required)
- `team_name` (string, required)
- `points` (number, required, >=0)
- `wins` (int, required, >=0)
- `data_as_of_utc` (string datetime, required)
- `source` (string, required, default `hyprace`)

Validation rules:
- Positions should be unique per season.

## Entity: ConstructorStandingRow

- Purpose: Constructors championship row exposed to frontend.
- Primary key: `id` (format: `season-constructor-position`)
- Partition key: `season`

Fields:
- `id` (string, required)
- `season` (int, required)
- `position` (int, required, >=1)
- `team_name` (string, required)
- `points` (number, required, >=0)
- `data_as_of_utc` (string datetime, required)
- `source` (string, required, default `hyprace`)

Validation rules:
- Positions should be unique per season.

## Relationship Notes

- `RaceMeeting` drives `UpcomingRaceSnapshot` computation.
- Standings rows are season-scoped and independent of race-level documents.
- API responses must include `data_as_of_utc` for freshness transparency.