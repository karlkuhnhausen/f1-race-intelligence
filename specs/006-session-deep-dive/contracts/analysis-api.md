# API Contract: Session Analysis

## Endpoint

```
GET /api/v1/rounds/{round}/sessions/{type}/analysis?year={year}
```

**New endpoint** introduced by Feature 006. Returns all analysis data for a completed Race or Sprint session.

---

## Path Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `round` | int | yes | Round number (1-24) |
| `type` | string | yes | Session type: `"race"` or `"sprint"` only |

## Query Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `year` | int | no | 2026 | Season year |

---

## Response: 200 OK

Returned when analysis data has been ingested for the requested session.

```jsonc
{
  "year": 2026,
  "round": 4,
  "session_type": "race",
  "total_laps": 53,

  // Position data: one entry per driver with per-lap positions.
  // Always present when analysis data exists (required data type).
  "positions": [
    {
      "driver_number": 1,
      "driver_name": "Max Verstappen",
      "driver_acronym": "VER",
      "team_name": "Red Bull Racing",
      "laps": [
        { "lap": 1, "position": 1 },
        { "lap": 2, "position": 1 },
        // ... one entry per completed lap
        { "lap": 53, "position": 1 }
      ]
    },
    {
      "driver_number": 4,
      "driver_name": "Lando Norris",
      "driver_acronym": "NOR",
      "team_name": "McLaren",
      "laps": [
        { "lap": 1, "position": 4 },
        { "lap": 2, "position": 3 },
        // DNF drivers have fewer laps (line ends at retirement lap)
        { "lap": 48, "position": 2 }
      ]
    }
    // ... all 20 drivers
  ],

  // Interval data: gap-to-leader per driver per lap.
  // Omitted (null) if interval data was unavailable from the source.
  "intervals": [
    {
      "driver_number": 1,
      "driver_acronym": "VER",
      "team_name": "Red Bull Racing",
      "laps": [
        { "lap": 1, "gap_to_leader": 0, "interval": 0 },
        { "lap": 2, "gap_to_leader": 0, "interval": 0 }
        // Leader always has 0 gap
      ]
    },
    {
      "driver_number": 4,
      "driver_acronym": "NOR",
      "team_name": "McLaren",
      "laps": [
        { "lap": 1, "gap_to_leader": 1.234, "interval": 0.567 },
        { "lap": 2, "gap_to_leader": 1.891, "interval": 0.891 }
      ]
    }
  ],

  // Stint data: tire compound usage per driver.
  // Omitted (null) if stint data was unavailable.
  "stints": [
    {
      "driver_number": 1,
      "driver_acronym": "VER",
      "team_name": "Red Bull Racing",
      "stint_number": 1,
      "compound": "SOFT",
      "lap_start": 1,
      "lap_end": 18,
      "tyre_age_at_start": 0
    },
    {
      "driver_number": 1,
      "driver_acronym": "VER",
      "team_name": "Red Bull Racing",
      "stint_number": 2,
      "compound": "HARD",
      "lap_start": 19,
      "lap_end": 53,
      "tyre_age_at_start": 0
    }
    // ... all stints for all drivers
  ],

  // Pit stop data: timing for each pit stop.
  // Omitted (null) if pit data was unavailable.
  "pits": [
    {
      "driver_number": 1,
      "driver_acronym": "VER",
      "team_name": "Red Bull Racing",
      "lap": 18,
      "pit_duration": 22.456,    // total pit lane time (seconds)
      "stop_duration": 2.3       // stationary time (seconds); 0 if unavailable
    },
    {
      "driver_number": 4,
      "driver_acronym": "NOR",
      "team_name": "McLaren",
      "lap": 20,
      "pit_duration": 24.891,
      "stop_duration": 2.5
    }
  ],

  // Overtake data: on-track overtake events.
  // Omitted (null) if overtake data was unavailable or empty.
  // This is the P3 priority data — graceful degradation expected.
  "overtakes": [
    {
      "overtaking_driver_number": 4,
      "overtaking_driver_name": "Lando Norris",
      "overtaken_driver_number": 16,
      "overtaken_driver_name": "Charles Leclerc",
      "lap": 12,
      "position": 3
    }
  ]
}
```

---

## Response: 404 Not Found

Returned when:
- The session type is not `"race"` or `"sprint"`
- The session exists but analysis data has not been ingested yet
- The round does not exist for the given year

```jsonc
{
  "error": "analysis_not_available",
  "message": "Analysis data is not yet available for this session. Data typically appears approximately 2 hours after session end."
}
```

---

## Response: 400 Bad Request

Returned when path parameters are malformed.

```jsonc
{
  "error": "invalid_request",
  "message": "Round must be a positive integer"
}
```

---

## Data Guarantees

| Field | Guarantee |
|-------|-----------|
| `positions` | Always present when response is 200. Never null/empty. |
| `intervals` | May be null if interval data unavailable from source. Frontend shows placeholder. |
| `stints` | May be null if stint data unavailable. Frontend shows placeholder. |
| `pits` | May be null if no pit stops occurred (sprint edge case) or data unavailable. |
| `overtakes` | May be null if overtake data unavailable or empty. Frontend degrades gracefully. |

---

## Performance Characteristics

| Metric | Expected Value |
|--------|---------------|
| Response size (uncompressed) | ~100-150KB |
| Response size (gzipped) | ~12-18KB |
| Cosmos query (single partition) | <100ms |
| Total API response time (p95) | <500ms |
| Number of Cosmos documents read | ~170 per session |

---

## Caching Behavior

- Analysis data is **immutable** once ingested (post-session data does not change)
- The API MAY set `Cache-Control: public, max-age=86400` (24h) on 200 responses
- 404 responses MUST NOT be cached (data may become available after ingestion)

---

## CORS

Same as existing API — handled by the nginx ingress controller. No additional CORS configuration needed.
