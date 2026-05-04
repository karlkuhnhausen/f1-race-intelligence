# API Contracts: Standings Overhaul

Base URL: `/api/v1`

## Existing Endpoints (Enhanced)

### GET /standings/drivers

Returns current driver championship standings with expanded statistics.

**Query Parameters**:
| Param | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `year` | int | no | current year | Season year (2023–current) |

**Response** `200 OK`:
```json
{
  "year": 2026,
  "data_as_of_utc": "2026-05-02T19:30:00Z",
  "rows": [
    {
      "position": 1,
      "driver_number": 12,
      "driver_name": "Andrea Kimi Antonelli",
      "team_name": "Mercedes",
      "team_color": "27F4D2",
      "points": 75.0,
      "wins": 2,
      "podiums": 4,
      "dnfs": 0,
      "poles": 1
    }
  ]
}
```

**Error Responses**:
- `400 Bad Request`: Invalid year parameter (outside 2023–current range)
- `404 Not Found`: No data available for requested year
- `500 Internal Server Error`: Storage or computation failure

---

### GET /standings/constructors

Returns current constructor championship standings with expanded statistics.

**Query Parameters**:
| Param | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `year` | int | no | current year | Season year (2023–current) |

**Response** `200 OK`:
```json
{
  "year": 2026,
  "data_as_of_utc": "2026-05-02T19:30:00Z",
  "rows": [
    {
      "position": 1,
      "team_name": "Mercedes",
      "team_color": "27F4D2",
      "points": 130.0,
      "wins": 3,
      "podiums": 7,
      "dnfs": 0
    }
  ]
}
```

**Error Responses**: Same as drivers endpoint.

---

## New Endpoints

### GET /standings/drivers/progression

Returns per-race cumulative points for all drivers in a season, for rendering progression line charts.

**Query Parameters**:
| Param | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `year` | int | no | current year | Season year (2023–current) |

**Response** `200 OK`:
```json
{
  "year": 2026,
  "rounds": [
    { "session_key": 11234, "round_name": "Australia" },
    { "session_key": 11240, "round_name": "China Sprint" },
    { "session_key": 11241, "round_name": "China" },
    { "session_key": 11249, "round_name": "Japan" },
    { "session_key": 11275, "round_name": "Miami Sprint" }
  ],
  "drivers": [
    {
      "driver_number": 12,
      "driver_name": "Andrea Kimi Antonelli",
      "team_name": "Mercedes",
      "team_color": "27F4D2",
      "points_by_round": [18.0, 18.0, 43.0, 72.0, 75.0]
    }
  ]
}
```

**Notes**:
- `points_by_round` array is parallel to `rounds` array (same length, same order).
- Points are cumulative (`points_current` from each championship snapshot).
- Includes Sprint sessions as separate rounds when they affect standings.

---

### GET /standings/constructors/progression

Returns per-race cumulative points for all constructors in a season.

**Query Parameters**:
| Param | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `year` | int | no | current year | Season year (2023–current) |

**Response** `200 OK`:
```json
{
  "year": 2026,
  "rounds": [
    { "session_key": 11234, "round_name": "Australia" },
    { "session_key": 11240, "round_name": "China Sprint" }
  ],
  "teams": [
    {
      "team_name": "Mercedes",
      "team_color": "27F4D2",
      "points_by_round": [43.0, 43.0]
    }
  ]
}
```

---

### GET /standings/drivers/compare

Returns head-to-head comparison data for two drivers in a season.

**Query Parameters**:
| Param | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `year` | int | no | current year | Season year |
| `driver1` | int | yes | — | First driver number |
| `driver2` | int | yes | — | Second driver number |

**Response** `200 OK`:
```json
{
  "year": 2026,
  "data_as_of_utc": "2026-05-02T19:30:00Z",
  "driver1": {
    "driver_number": 12,
    "driver_name": "Andrea Kimi Antonelli",
    "team_name": "Mercedes",
    "team_color": "27F4D2",
    "position": 1,
    "points": 75.0,
    "wins": 2,
    "podiums": 4,
    "dnfs": 0,
    "poles": 1
  },
  "driver2": {
    "driver_number": 63,
    "driver_name": "George Russell",
    "team_name": "Mercedes",
    "team_color": "27F4D2",
    "position": 2,
    "points": 68.0,
    "wins": 1,
    "podiums": 3,
    "dnfs": 0,
    "poles": 1
  },
  "deltas": {
    "position": -1,
    "points": 7.0,
    "wins": 1,
    "podiums": 1,
    "dnfs": 0,
    "poles": 0
  },
  "progression": {
    "rounds": [
      { "session_key": 11234, "round_name": "Australia" },
      { "session_key": 11275, "round_name": "Miami Sprint" }
    ],
    "driver1_points": [18.0, 75.0],
    "driver2_points": [25.0, 68.0]
  }
}
```

**Notes**:
- `deltas` are driver1 - driver2 (positive means driver1 is ahead).
- `deltas.position` is inverted: negative means driver1 has a better (lower) position number.

**Error Responses**:
- `400 Bad Request`: Missing driver1 or driver2, invalid year, same driver specified twice
- `404 Not Found`: One or both drivers not found in standings for that year

---

### GET /standings/constructors/compare

Returns head-to-head comparison data for two constructors in a season.

**Query Parameters**:
| Param | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `year` | int | no | current year | Season year |
| `team1` | string | yes | — | First team name (URL-encoded) |
| `team2` | string | yes | — | Second team name (URL-encoded) |

**Response** `200 OK`:
```json
{
  "year": 2026,
  "data_as_of_utc": "2026-05-02T19:30:00Z",
  "team1": {
    "team_name": "Mercedes",
    "team_color": "27F4D2",
    "position": 1,
    "points": 130.0,
    "wins": 3,
    "podiums": 7,
    "dnfs": 0
  },
  "team2": {
    "team_name": "McLaren",
    "team_color": "FF8000",
    "position": 3,
    "points": 55.0,
    "wins": 0,
    "podiums": 2,
    "dnfs": 1
  },
  "deltas": {
    "position": -2,
    "points": 75.0,
    "wins": 3,
    "podiums": 5,
    "dnfs": -1
  },
  "progression": {
    "rounds": [
      { "session_key": 11234, "round_name": "Australia" }
    ],
    "team1_points": [43.0],
    "team2_points": [10.0]
  }
}
```

**Error Responses**:
- `400 Bad Request`: Missing team1 or team2, invalid year, same team specified twice
- `404 Not Found`: One or both teams not found in standings for that year

---

### GET /standings/constructors/{team}/drivers

Returns individual driver breakdown for a specific constructor.

**Path Parameters**:
| Param | Type | Description |
|-------|------|-------------|
| `team` | string | Team name slug (URL-encoded, e.g., "Mercedes", "Red Bull Racing") |

**Query Parameters**:
| Param | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `year` | int | no | current year | Season year |

**Response** `200 OK`:
```json
{
  "year": 2026,
  "team_name": "Mercedes",
  "team_color": "27F4D2",
  "team_points": 130.0,
  "team_position": 1,
  "drivers": [
    {
      "driver_number": 12,
      "driver_name": "Andrea Kimi Antonelli",
      "position": 1,
      "points": 75.0,
      "wins": 2,
      "podiums": 4,
      "dnfs": 0,
      "poles": 1,
      "points_percentage": 57.7
    },
    {
      "driver_number": 63,
      "driver_name": "George Russell",
      "position": 2,
      "points": 68.0,
      "wins": 1,
      "podiums": 3,
      "dnfs": 0,
      "poles": 1,
      "points_percentage": 52.3
    }
  ]
}
```

**Notes**:
- `points_percentage` = (driver points / team points) × 100, rounded to 1 decimal.
- Drivers sorted by points descending.
- Sum of driver points may exceed team points if sprint/race scoring differs (sprint points accrue to both driver and constructor championships separately). However, OpenF1 reports team points independently, so we trust `team_points` from the team championship snapshot.

**Error Responses**:
- `400 Bad Request`: Invalid year
- `404 Not Found`: Team not found in standings for that year
