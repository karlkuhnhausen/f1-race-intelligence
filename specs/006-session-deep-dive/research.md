# Research: Session Deep Dive Page (Feature 006)

## Resolved Questions

### 1. OpenF1 `/position` Response Shape and Aggregation Strategy

**Decision**: Fetch all position entries for a session, then aggregate to 1 entry per (driver_number, lap_number) by keeping the LAST entry per pair (highest timestamp wins).

OpenF1 `/v1/position` response fields:
| Field | Type | Notes |
|-------|------|-------|
| `session_key` | int | Session identifier |
| `driver_number` | int | Car number |
| `position` | int | Track position (1-20) |
| `date` | string (ISO 8601) | Timestamp of the position update |

**Key insight**: OpenF1 emits position updates in real-time during the session — a single lap may have 2-5 position entries per driver as positions change mid-lap. For post-session analysis, we only care about the **final** position each driver held at the **end** of each lap.

**Aggregation algorithm**:
1. Group raw entries by (driver_number, lap_number) — derive lap_number from the entry's chronological position relative to lap boundaries (position data does not include a lap_number field directly; we must derive it from the `date` field relative to known lap start times from the `/laps` endpoint, OR use the simpler approach of counting sequential position changes).
2. **Simpler approach chosen**: Since we also fetch `/stints` which has `lap_start`/`lap_end`, and the position data is time-ordered, assign lap numbers by counting position entries per driver chronologically. The Nth distinct position value OR the Nth time-boundary crossing = lap N.
3. **Even simpler approach (final decision)**: OpenF1 position data DOES include a `meeting_key` and is ordered by `date`. We'll fetch position data and cross-reference with the laps endpoint to derive per-lap positions. OR: use the fact that position data includes many updates per lap — we group by driver, sort by date, and take one sample per lap boundary.

**Final practical approach**: The OpenF1 position endpoint returns data sampled at high frequency. We'll aggregate by bucketing timestamps into laps using the known race distance (total laps from session results, already available). Take the last position value before each lap boundary.

**Rationale**: Server-side aggregation per FR-010 reduces payload from ~5000 raw rows to ~1200 points (20 drivers × 60 laps). This is a manageable payload for the frontend charting library.

**Alternatives considered**:
- Send all raw position data to frontend and aggregate client-side — rejected: violates FR-010, 5000+ rows causes render jank
- Use only position at lap completion (from `/laps` endpoint) — rejected: `/laps` doesn't have a position field; we need the position endpoint

---

### 2. OpenF1 `/intervals` Response Shape and Gap-to-Leader Derivation

**Decision**: Fetch all interval entries, aggregate to 1 per (driver_number, lap_number) by keeping the last entry per pair.

OpenF1 `/v1/intervals` response fields:
| Field | Type | Notes |
|-------|------|-------|
| `session_key` | int | Session identifier |
| `driver_number` | int | Car number |
| `gap_to_leader` | float/null | Gap in seconds to race leader; null for leader |
| `interval` | float/null | Gap to car immediately ahead; null for leader |
| `date` | string (ISO 8601) | Timestamp |

**Key insight**: Interval data is sampled frequently (similar to position). The `gap_to_leader` field directly gives us what we need for the gap-to-leader chart. Null means the driver IS the leader (gap = 0).

**Aggregation**: Same strategy as position — bucket by lap, take last value per lap per driver. The `date` field is used for lap assignment.

**Rationale**: Direct mapping to the spec requirement (FR-005). Gap values naturally converge during safety car periods (SC-002 acceptance scenario).

**Alternatives considered**:
- Calculate gaps from lap times — rejected: cumulative errors, doesn't account for pit stops
- Use only interval (gap to car ahead) — rejected: spec explicitly requires gap-to-leader (FR-005 Scenario 1)

---

### 3. OpenF1 `/stints` Response Shape and Compound Mapping

**Decision**: Fetch stints directly — they map 1:1 to our domain type with minimal transformation.

OpenF1 `/v1/stints` response fields:
| Field | Type | Notes |
|-------|------|-------|
| `session_key` | int | Session identifier |
| `driver_number` | int | Car number |
| `stint_number` | int | Sequential stint (1, 2, 3...) |
| `compound` | string | "SOFT", "MEDIUM", "HARD", "INTERMEDIATE", "WET" |
| `lap_start` | int | First lap of this stint |
| `lap_end` | int | Last lap of this stint |
| `tyre_age_at_start` | int | Age of tyres when stint began (0 for new set) |

**No aggregation needed**: Stints are already per-driver per-stint granularity. Typically 2-4 stints per driver per race.

**Compound color mapping** (frontend):
- SOFT → `#FF3333` (red)
- MEDIUM → `#FFC300` (yellow)
- HARD → `#FFFFFF` (white / light gray on dark bg)
- INTERMEDIATE → `#43B02A` (green)
- WET → `#0072CE` (blue)

**Rationale**: Direct 1:1 mapping. Data volume is tiny (~60-80 stints per race, 20 drivers × 2-4 stops).

---

### 4. OpenF1 `/pit` Response Shape and Duration Fields

**Decision**: Fetch pit data directly — maps to domain type with field renaming.

OpenF1 `/v1/pit` response fields:
| Field | Type | Notes |
|-------|------|-------|
| `session_key` | int | Session identifier |
| `driver_number` | int | Car number |
| `lap_number` | int | Lap on which pit entry occurred |
| `pit_duration` | float | Total time in pit lane (seconds) — entry to exit |
| `date` | string (ISO 8601) | Timestamp of pit entry |

**Key note**: OpenF1 provides `pit_duration` (total pit lane time including entry/exit) but NOT a separate "stationary time" field. For the spec requirement of distinguishing slow stops (>5s), we'll use `pit_duration` directly — a normal stop is ~20-25s pit lane time; "slow" in our context means pit_duration > 30s (indicating a problem or long stop).

**Update**: Some OpenF1 responses also include a `duration` field representing the stationary time. If present, we use it; otherwise fall back to `pit_duration`. The threshold for "visually distinguishable" slow stops (Acceptance Scenario 2, User Story 4) is: stop_duration > 5s stationary OR pit_duration > 30s total.

**Rationale**: Simple mapping. Data volume is tiny (~40-60 pit stops per race).

---

### 5. OpenF1 `/overtakes` Data Availability

**Decision**: Fetch overtake data opportunistically. Empty response is expected and handled gracefully (FR-015).

**IMPORTANT FINDING**: The OpenF1 API endpoint for overtakes is `/v1/overtaking` (not `/overtakes`). Fields:
| Field | Type | Notes |
|-------|------|-------|
| `session_key` | int | Session identifier |
| `driver_number` | int | Overtaking driver |
| `overtaken_driver_number` | int | Driver who was passed |
| `lap_number` | int | Lap of the overtake |
| `date` | string (ISO 8601) | Timestamp |

**Availability concerns**: The overtaking endpoint is one of the newer OpenF1 endpoints and data may be incomplete or unavailable for some sessions. The spec explicitly handles this as an edge case: "Incomplete overtake data: Overtake annotations section gracefully degrades — chart renders without annotations, no error message."

**Rationale**: Best-effort enrichment. Position chart is the hero visualization and works without overtake data. Overtakes are P3 priority in the spec.

**Alternatives considered**:
- Derive overtakes from position changes — rejected: position changes don't distinguish on-track passes from pit-related position gains
- Skip overtake data entirely — rejected: spec includes it as a P3 user story; worth attempting

---

### 6. Recharts Suitability for Multi-Line Charts

**Decision**: Use `recharts` v2.12+ as the charting library.

**Performance evaluation**:
- 20 lines × 60 points = 1200 data points total for the position chart — well within recharts' comfort zone (tested up to 10,000 points without jank)
- SVG-based rendering is appropriate for this data density
- `ResponsiveContainer` handles mobile/desktop layout natively
- Supports inverted Y-axis via `domain={[20, 1]}` on YAxis (needed for position chart)

**Key recharts components needed**:
- `LineChart` + `Line` → position chart, gap-to-leader chart
- `BarChart` + `Bar` → tire strategy swimlane (horizontal stacked bars)
- `ScatterChart` + `Scatter` → pit stop timeline
- `ResponsiveContainer` → all charts (mobile responsiveness)
- `Tooltip` → interactive hover details
- `ReferenceDot` → overtake annotations on position chart

**Bundle size**: ~150KB gzipped (tree-shakeable — only imported components are bundled).

**Rationale**: React-native library (uses React components, not imperative DOM manipulation). Works with existing React 18 setup. Large community, well-maintained, good TypeScript support. No Canvas/WebGL complexity needed for this data density.

**Alternatives considered**:
- `chart.js` + `react-chartjs-2` — rejected: imperative API wrapped in React; less idiomatic; canvas-based (harder to style with CSS)
- `d3` directly — rejected: low-level, significant implementation effort for standard chart types; not React-native
- `visx` (Airbnb) — rejected: also low-level d3 primitives; more code for less result vs. recharts
- `nivo` — rejected: heavier bundle, opinion about themes that conflicts with our design system
- `victory` — viable alternative but less active maintenance than recharts; similar API surface

---

### 7. Cosmos DB Document Model

**Decision**: Store analysis data as **per-driver batch documents** — one document per (session, data_type, driver). This gives ~20 documents per data type per session (one per driver), totaling ~100 documents per session for all 5 types.

**Document ID pattern**: `analysis_{datatype}_{round}_{sessiontype}_{drivernum}`
- Example: `analysis_position_4_race_1` (Round 4, Race, driver #1)
- Example: `analysis_stint_4_race_44` (Round 4, Race, driver #44)
- Overtakes use: `analysis_overtake_{round}_{sessiontype}` (single document per session since overtakes reference pairs of drivers)

**Partition key**: `season` (same as existing session documents — keeps all data for a season co-located).

**Rationale**: Per-driver documents keep individual document sizes manageable (~2-5KB each for position data with 60 laps). Querying all analysis data for a session = `SELECT * FROM c WHERE c.type LIKE 'analysis_%' AND c.round = X AND c.session_type = Y` (single partition query, ~100 results).

**Alternatives considered**:
- Single mega-document per session with all data — rejected: would be 50-100KB, exceeds comfortable Cosmos document size for frequent reads
- Per-lap documents — rejected: too many small documents (60 laps × 20 drivers × 2 types = 2400 documents per session); query overhead
- Separate container — rejected: unnecessary operational complexity; same partition key works fine

---

### 8. Position Data Aggregation Algorithm (Detail)

**Decision**: Two-pass aggregation on the backend:

**Pass 1 — Lap Assignment**:
- Position data from OpenF1 has `date` timestamps but no explicit `lap_number`
- We know the total race laps from the session results (already fetched)
- Strategy: Sort position entries by (driver_number, date). For each driver, number their unique position-change events sequentially as approximate laps.
- **Better strategy**: Cross-reference with interval data which DOES have implicit lap timing, or simply count position entries per driver and divide by known total laps to get approximate lap boundaries.
- **Best strategy (chosen)**: Fetch `/laps` endpoint to get lap start/end times per driver, then assign each position entry to the lap whose time window contains it.

**Pass 2 — Deduplication**:
- After lap assignment, keep only the LAST position entry per (driver_number, lap_number)
- This captures the final position held at end of each lap

**Practical note**: If lap timing data is unavailable (edge case), fall back to evenly dividing the chronological position entries into `total_laps` buckets. This is an approximation but sufficient for visualization.

**Output**: Exactly 1 position value per driver per lap. Drivers who DNF have entries only up to their retirement lap.

---

### 9. Backfill CLI Extension Strategy

**Decision**: Add `--analysis` boolean flag to the existing `backfill` binary. When set, it runs the analysis backfill flow instead of (or in addition to) the race-control backfill.

**Flags**:
```
--season=2026         (existing, required)
--dry-run             (existing, applies to analysis too)
--rate-limit-ms=1000  (existing, applies between sessions)
--analysis            (NEW: triggers analysis data backfill)
```

**Behavior when `--analysis` is set**:
1. Fetch all finalized sessions for the season
2. Filter to Race and Sprint types only
3. For each, check `HasAnalysisData` — skip if true (idempotent)
4. Fetch 5 OpenF1 endpoints, aggregate, persist
5. 500ms between endpoints, 1000ms between sessions
6. Expected runtime for 7 sessions: ~7 × (5 × 0.5s fetch + 5 × 0.5s delay + 1s inter-session) = ~42 seconds. Well within the 30-minute SC-007 budget.

**Rationale**: Extends the existing binary rather than creating a new one — fewer deployment artifacts, shared Cosmos wiring and credential management.

**Alternatives considered**:
- Separate `backfill-analysis` binary — rejected: duplicates Cosmos client setup, credential loading, and flag parsing
- Subcommand pattern (`backfill analysis --season=2026`) — rejected: over-engineering for a CLI with only 2 modes; simple flag is sufficient
- Run both backfills when no specific flag given — rejected: analysis backfill is much heavier (5 endpoints × N sessions); should be opt-in

---

### 10. Frontend Route Structure and Navigation Flow

**Decision**: New route at `/rounds/:round/sessions/:sessionType/analysis` with back-navigation to the round detail page.

**Navigation flow**:
```
Calendar → Round Detail Page → [View Analysis button] → Analysis Page
                                                              ↓
                                                    [← Back to Round N]
```

**Route parameters**:
- `:round` — round number (int)
- `:sessionType` — `"race"` or `"sprint"` (string)
- `?year=` query param — defaults to 2026 (consistent with other pages)

**"View Analysis" button placement**: Inside the `SessionCard` component on `RoundDetailPage.tsx`, shown only when:
- `session.status === 'completed'`
- `session.session_type === 'race' || session.session_type === 'sprint'`

NOT shown for: qualifying, sprint_qualifying, practice (per FR-001, FR-003).

**404 handling**: If the API returns 404 (analysis data not yet ingested), the page shows an informational message: "Analysis not yet available. Data typically appears approximately 2 hours after session end." This matches FR-014.

**Rationale**: Clean URL structure consistent with existing round detail pattern. Back-navigation maintains context.

**Alternatives considered**:
- Tab on the round detail page — rejected: analysis page has significant content (5 charts), would make round detail page too heavy
- Modal/drawer — rejected: charts need full viewport width for readability
- Nested route under round detail — rejected: analysis page has its own data fetch lifecycle; separate route is cleaner
