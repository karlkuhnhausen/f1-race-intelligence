# API Contract: Round Detail with Session Recap Summary

## Endpoint

```
GET /api/v1/rounds/{round}?year={year}
```

This feature **extends** the existing round detail endpoint (introduced in Feature 003). No new endpoint is added. The response shape gains a `recap_summary` field per session.

---

## Response Schema

```jsonc
{
  "year": 2026,
  "round": 4,
  "race_name": "Japanese Grand Prix",
  "circuit_name": "Suzuka International Racing Course",
  "country_name": "Japan",
  "data_as_of_utc": "2026-04-06T08:00:00Z",
  "sessions": [
    {
      // --- Existing fields (unchanged) ---
      "session_name": "Practice 1",
      "session_type": "practice1",
      "status": "completed",               // derived from timestamps at read time
      "date_start_utc": "2026-04-04T03:30:00Z",
      "date_end_utc": "2026-04-04T04:30:00Z",
      "results": [ /* ...existing SessionResultDTO array... */ ],

      // --- NEW field (Feature 005) ---
      // recap_summary is populated for completed sessions only.
      // Omitted (null) for: upcoming sessions, in-progress sessions,
      // and completed sessions where race-control data is unavailable
      // (graceful degradation path).
      "recap_summary": {
        // Session-type-specific fields are described below.
        // All numeric times are in seconds (float64).
        // All string times use "m:ss.sss" format where applicable.
      }
    }
  ]
}
```

---

## `recap_summary` by Session Type

### Race / Sprint

```jsonc
{
  "recap_summary": {
    "winner_name": "Max Verstappen",
    "winner_team": "Red Bull Racing",
    "gap_to_p2": "+8.294",             // formatted string; "+1 LAP" for laps-behind
    "fastest_lap_holder": "Lando Norris",
    "fastest_lap_team": "McLaren",
    "fastest_lap_time_seconds": 87.097, // omitted if not available (historical session)
    "total_laps": 53,                   // winner's number_of_laps

    // Race-control events (omitted entirely if no events occurred):
    "red_flag_count": 0,               // omitted if 0 (zero-value omitempty)
    "safety_car_count": 2,
    "vsc_count": 1,
    // top_event: single highest-priority event type with at least 1 activation.
    // Priority: red_flag > safety_car > vsc > investigation.
    "top_event": {
      "event_type": "safety_car",      // "red_flag" | "safety_car" | "vsc" | "investigation"
      "lap_number": 14,                // first occurrence; 0 for pre-race events
      "count": 2                       // distinct activations of this event type
    }
  }
}
```

**When no race-control events occurred**, `red_flag_count`, `safety_car_count`, `vsc_count`, and `top_event` are all omitted from the response (Go `omitempty` on zero-value int and nil pointer).

**When fewer than 2 classified finishers**, `gap_to_p2` is omitted.

**When no classified finisher exists** (e.g., red-flagged race, no laps completed), `winner_name`, `winner_team`, `gap_to_p2`, `fastest_lap_holder`, `fastest_lap_team`, `total_laps` are omitted.

---

### Qualifying / Sprint Qualifying

```jsonc
{
  "recap_summary": {
    "pole_sitter_name": "Charles Leclerc",
    "pole_sitter_team": "Ferrari",
    "pole_time": 86.983,               // Q3 time in seconds for standard qualifying
    "gap_to_p2": "+0.132",             // formatted string (Q3 P1 - Q3 P2 delta)
    "q1_cutoff_time": 88.211,          // Q1 time of the last P15 driver (standard format)
    "q2_cutoff_time": 87.654,          // Q2 time of the last P10 driver (standard format)
    // q1_cutoff_time and q2_cutoff_time are omitted for sprint qualifying
    // or single-segment formats where no elimination data exists.

    // Race-control events (same as race — omitted if no events):
    "red_flag_count": 1,
    "top_event": {
      "event_type": "red_flag",
      "lap_number": 0,                 // qualifying events use 0 when lap_number is irrelevant
      "count": 1
    }
  }
}
```

---

### Practice 1 / Practice 2 / Practice 3

```jsonc
{
  "recap_summary": {
    "best_driver_name": "George Russell",
    "best_driver_team": "Mercedes",
    "best_lap_time": 89.102,           // session-best lap time in seconds (P1 result's BestLapTime)
    "total_laps": 143,                 // sum of NumberOfLaps across all drivers in this session

    // Race-control events (same as race — omitted if no events):
    "red_flag_count": 1,
    "top_event": {
      "event_type": "red_flag",
      "lap_number": 0,
      "count": 1
    }
  }
}
```

---

## `recap_summary` Omission Rules

| Condition | recap_summary field |
|-----------|---------------------|
| Session status is `upcoming` | `recap_summary` omitted entirely |
| Session status is `in_progress` | `recap_summary` omitted entirely |
| Session status is `completed`, race-control data unavailable, no degradation retry | `recap_summary` included but event fields omitted |
| Session status is `completed`, full data available | All applicable fields included |

---

## Unchanged Contract

All fields from the existing Feature 003 response (year, round, race_name, circuit_name, country_name, data_as_of_utc, sessions array with session_name, session_type, status, date_start_utc, date_end_utc, results) remain identical. This is a non-breaking additive extension.

---

## Frontend TypeScript Interface Extension

```typescript
// Extend existing SessionDetail interface (frontend/src/features/rounds/roundApi.ts)

export interface NotableEvent {
  event_type: 'red_flag' | 'safety_car' | 'vsc' | 'investigation';
  lap_number: number;
  count: number;
}

export interface SessionRecapSummary {
  // Race / Sprint
  winner_name?: string;
  winner_team?: string;
  gap_to_p2?: string;
  fastest_lap_holder?: string;
  fastest_lap_team?: string;
  fastest_lap_time_seconds?: number;
  total_laps?: number;

  // Qualifying
  pole_sitter_name?: string;
  pole_sitter_team?: string;
  pole_time?: number;
  q1_cutoff_time?: number;
  q2_cutoff_time?: number;

  // Practice
  best_driver_name?: string;
  best_driver_team?: string;
  best_lap_time?: number;

  // All sessions
  red_flag_count?: number;
  safety_car_count?: number;
  vsc_count?: number;
  top_event?: NotableEvent;
}

export interface SessionDetail {
  // ...existing fields...
  recap_summary?: SessionRecapSummary;  // NEW
}
```
