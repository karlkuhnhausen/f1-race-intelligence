# Day 6: Clicking Into the Race — Session Results & Round Detail

*Posted April 19, 2026 · Karl Kuhnhausen*

---

The calendar told you *when*. Now it tells you *what happened*.

Feature 003 — Race Session Results — adds the ability to click any round in the calendar and see every session that occurred during that race weekend: practice, qualifying, sprint, and the race itself. Finishing positions, driver names, teams, statuses. The data pipeline fetches it from OpenF1, stores it in Cosmos DB, and the frontend renders it on a per-round detail page.

This is the first three phases. The foundation, the API, and the navigation. Eighteen tasks completed. The remaining thirteen will add dedicated race, qualifying, and practice result components with richer formatting — but the core is live.

![F1 Race Intelligence Dashboard — Calendar view with clickable race name hyperlinks](images/Race%20Results%20Hyperlinks.png)

*Every race name in the calendar is now a hyperlink. Click "Australian Grand Prix" and you're on the round detail page. Cancelled races (Bahrain, Saudi Arabia) stay plain text — there's nothing to click into.*

---

## Phase 1: Setup

Two tasks. Create the directory scaffolding and add the one new dependency.

The backend gained `backend/internal/api/rounds/` for the new handler, service, and DTOs. The frontend gained `frontend/src/features/rounds/` for the round detail page, results table, and API client. And `react-router-dom` was added to `package.json` — the first new frontend dependency since the project started — because URL-based routing is required for deep-linking to `/rounds/3?year=2026`.

No code yet. Just the directories and the dependency. Phase 1 exists so Phase 2 has somewhere to put things.

---

## Phase 2: The Data Pipeline

This is where the complexity lives. Six tasks, all blocking — nothing in the frontend can work until the backend can ingest, store, and serve session data.

### Domain Types

`backend/internal/domain/session.go` defines the type system:

```go
type SessionType string

const (
    SessionPractice1        SessionType = "practice1"
    SessionPractice2        SessionType = "practice2"
    SessionPractice3        SessionType = "practice3"
    SessionSprintQualifying SessionType = "sprint_qualifying"
    SessionSprint           SessionType = "sprint"
    SessionQualifying       SessionType = "qualifying"
    SessionRace             SessionType = "race"
)
```

Seven session types covering both standard and sprint weekends. A `MapOpenF1SessionType` function converts upstream names like `"Practice 1"` and `"Sprint Qualifying"` to our internal enum. `SessionTypeSlug` produces the URL-safe string used in document IDs: `2026-03-race`, `2026-03-qualifying`.

Finishing statuses — `Finished`, `DNF`, `DNS`, `DSQ` — are their own enum. The domain layer doesn't import anything from storage or API. It's a leaf package.

### Storage Layer

`backend/internal/storage/repository.go` adds two new structs — `Session` and `SessionResult` — and a `SessionRepository` interface:

```go
type SessionRepository interface {
    UpsertSession(ctx context.Context, s Session) error
    UpsertSessionResult(ctx context.Context, r SessionResult) error
    GetSessionsByRound(ctx context.Context, season, round int) ([]Session, error)
    GetSessionResultsByRound(ctx context.Context, season, round int) ([]SessionResult, error)
}
```

Four methods. Upsert for idempotent writes from the poller. Query-by-round for the API. Both `Session` and `SessionResult` documents live in the same Cosmos container, partitioned by `/season`, distinguished by a `type` discriminator field (`"session"` vs `"session_result"`).

### Cosmos Implementation

`backend/internal/storage/cosmos/sessions.go` implements all four repository methods. The queries filter by season, round, and document type. Originally the queries used `ORDER BY c.session_type, c.position` — which Cosmos DB rejected at runtime because multi-field ORDER BY requires a composite index that wasn't defined on the container. The fix: remove ORDER BY from the Cosmos queries and sort in Go with `sort.Slice`. Cheaper than adding composite indexes for a dataset that maxes out at ~150 results per round.

### Session Poller

`backend/internal/ingest/session_poller.go` extends the existing 5-minute poll cycle. It fetches sessions from OpenF1's `/v1/sessions` endpoint, then for each session fetches positions from `/v1/positions` and drivers from `/v1/drivers`. The transform layer (`session_transform.go`) converts the raw OpenF1 shapes into our domain types and writes them to Cosmos.

The poller hit an immediate real-world problem on first deployment: OpenF1 rate-limits at ~1 request/second. Hammering every session for positions and drivers in a tight loop produced a wall of `429 Too Many Requests` errors. The session metadata was ingested successfully (fetched in bulk by meeting key), but per-session position data requires sequential fetching with back-off — a problem for the next iteration.

### Wiring

`backend/cmd/api/main.go` creates the session repository from the Cosmos client and launches the session poller as a background goroutine. `backend/internal/api/router.go` gained the new route registration and the session repository injection.

---

## Phase 3: Navigation & Round Detail

Ten tasks. Three contract tests on the backend, four component tests on the frontend, and the implementation connecting them.

### The API: `GET /api/v1/rounds/{round}?year=2026`

The rounds handler validates the path parameter (1–30) and optional year query parameter (1950–2100, defaults to current year). The service queries the calendar repository for meeting metadata (race name, circuit, country) and the session repository for sessions and results. It assembles a `RoundDetailResponse`:

```go
type RoundDetailResponse struct {
    Season      int             `json:"season"`
    Round       int             `json:"round"`
    RaceName    string          `json:"race_name"`
    CircuitName string          `json:"circuit_name"`
    CountryName string          `json:"country_name"`
    DataAsOfUTC *time.Time      `json:"data_as_of_utc,omitempty"`
    Sessions    []SessionDetail `json:"sessions"`
}
```

Each `SessionDetail` contains the session name, type, status, date, and an array of `SessionResultDTO` entries — position, driver number, driver name, acronym, team, lap count, finishing status, gap, points.

Three contract tests validate the happy path (round with results), 400 for invalid parameters, and the empty-session case for a nonexistent round.

### Frontend Routing

`App.tsx` replaced the old `useState<Page>` tab-switching with `react-router-dom` routes:

- `/calendar` — the calendar page
- `/standings` — the standings page
- `/rounds/:round` — the new round detail page
- `*` → redirect to `/calendar`

The `NavLink` component highlights the active tab. `CalendarPage.tsx` wraps each race name in a `<Link to={/rounds/${round}?year=2026}>` — except for cancelled races, which remain plain text.

### The Round Detail Page

`RoundDetailPage.tsx` reads `round` from the URL path and `year` from the query string. It calls `fetchRoundDetail(round, year)`, then sorts sessions by weekend order (practice → qualifying → race) and renders a `SessionCard` for each.

Each card shows the session name, status badge, date, and a `SessionResultsTable` — or "No results available" if the session hasn't completed yet.

![Race Results Detail — Australian Grand Prix showing the Race session with 22 classified finishers](images/Race%20Results%20Detail-Australian%20GP.png)

*The Australian Grand Prix round detail page. George Russell P1, Kimi Antonelli P2, Charles Leclerc P3. Twenty-two finishers, all classified. The data was ingested from OpenF1, stored in Cosmos DB's `sessions` container, and served through the rounds API. Lewis Hamilton P4 — not bad for the first race of the season.*

### The Results Table

`SessionResultsTable.tsx` renders conditional columns based on session type. Race sessions show Status, Gap, and Points columns. Qualifying sessions show Q1, Q2, Q3 times. Practice sessions show Best Lap and Gap. A `formatLapTime` helper converts seconds to `M:SS.mmm` format.

Four frontend tests validate the round detail page: loading state, successful render with race data, error state, and navigation back to calendar. All wrapped in `MemoryRouter` + `Routes` to provide the router context that `useParams` requires.

---

## The Cosmos Gotcha

The most instructive bug was the Cosmos composite index error. The query:

```sql
SELECT * FROM c
WHERE c.season = @season AND c.round = @round AND c.type = 'session_result'
ORDER BY c.session_type, c.position
```

Works perfectly in single-partition mode with a single ORDER BY field. Add a second field and Cosmos demands a composite index — which wasn't defined on the container. The error message is clear (`"The order by query does not have a corresponding composite index that it can be served from"`), but it only surfaces at runtime, not at query parse time.

The fix was pragmatic: sort in Go. The result set per round is small (at most ~150 documents — 22 drivers × 7 sessions). A composite index would save a few microseconds of Go sorting at the cost of infrastructure complexity and an Azure deployment. Not worth it for this scale.

---

## What's Next

Phases 4–6 add dedicated result components:

- **Phase 4 (US1)**: Race results with time gaps, points scored, fastest lap indicator, and DNF/DNS/DSQ statuses
- **Phase 5 (US2)**: Qualifying results with Q1/Q2/Q3 segment times and elimination round indicators
- **Phase 6 (US3)**: Practice session results with best lap times, gaps to fastest, and lap counts

These three phases are independent — different component files, no shared state — so they can run in parallel. Phase 7 adds unit tests for the transform logic, integration tests for the ingestion pipeline, and E2E validation.

The foundation is deployed. The data is flowing. Click any race on the calendar and you'll see who finished where.

**Frontend:** http://f1.20.171.233.61.nip.io/
**Round Detail Example:** http://f1.20.171.233.61.nip.io/rounds/3?year=2026
**API:** http://api-f1.20.171.233.61.nip.io/api/v1/rounds/3?year=2026
**Source:** https://github.com/karlkuhnhausen/f1-race-intelligence

Eighteen tasks. Forty-two tests. One new Cosmos container. Zero lines typed by a human.

---

*Previous: [Day 5: Forty-Seven Tasks, Zero Lines — The Final Phase](day-5-phase-6-final-polish.md)*
*Next: [Day 7: Race Results, Rate Limits, and the Branch You Forgot You Were On](day-7-phase-4-and-branch-confusion.md)*
