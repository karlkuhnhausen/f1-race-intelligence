# Day 2: The First Thing Anyone Sees — Phase 3 and the Race Calendar MVP

*Posted April 19, 2026 · Karl Kuhnhausen*

---

In Day 0, I ratified a constitution. In Day 1, I built the entire foundation — data layer, pollers, infrastructure, CI/CD — without typing a line of code. Today, for the first time, there is something a human being can look at and understand.

Twenty-four rounds of the 2026 FIA Formula 1 World Championship. A table. Cancelled races marked in red. A card that tells you the next race is Miami, and how long until lights out.

This is Phase 3: User Story 1 — the calendar MVP. Ten tasks. Fourteen tests. The first vertical slice from database to browser.

---

## Why the Calendar Comes First

In a spec-driven project, you don't start with what's hardest. You start with what proves the architecture works end-to-end. The race calendar is the simplest possible feature that touches every layer: Cosmos DB query, Go service logic, HTTP handler, JSON response, React rendering. If this works — if data flows from the database through the API contract into a browser component — then every future feature is a variation on the same path.

The constitution calls this *Dispositio* — arrangement. The calendar isn't the most impressive feature. It's the one that validates the arrangement.

---

## The Domain Model

Phase 2 had repository interfaces and raw storage types. Phase 3 introduces the first proper domain type: `RaceMeeting`.

```go
type MeetingStatus string

const (
    StatusScheduled MeetingStatus = "scheduled"
    StatusCancelled MeetingStatus = "cancelled"
    StatusCompleted MeetingStatus = "completed"
    StatusUnknown   MeetingStatus = "unknown"
)

type RaceMeeting struct {
    Round            int
    RaceName         string
    CircuitName      string
    CountryName      string
    StartDatetimeUTC time.Time
    EndDatetimeUTC   time.Time
    Status           MeetingStatus
    IsCancelled      bool
    CancelledLabel   string
    CancelledReason  string
}
```

Four status values. An explicit `IsCancelled` flag alongside the status enum. A `CancelledLabel` and `CancelledReason` for the two rounds — Bahrain and Saudi Arabia — that aren't happening in 2026.

This is a deliberate design decision. The status enum could carry the cancellation semantics alone. But the UI needs a human-readable label ("Cancelled") and the data layer benefits from a separate boolean for query filtering. Two slightly different representations of the same fact, each serving its own consumer. The constitution's *Decor* principle: fitness for purpose.

---

## The Normalization Layer

Raw OpenF1 data doesn't look like a `RaceMeeting`. It arrives as flat JSON with meeting keys, loosely-typed dates, and no round numbers. The `NormalizeMeetings` function in `internal/ingest/meeting_transform.go` bridges the gap:

```go
func NormalizeMeetings(raw []openF1Meeting, season int) []storage.RaceMeeting {
    now := time.Now().UTC()
    meetings := make([]storage.RaceMeeting, 0, len(raw))

    for i, r := range raw {
        round := i + 1
        startUTC, _ := time.Parse(time.RFC3339, r.DateStart)

        m := storage.RaceMeeting{
            ID:               fmt.Sprintf("%d-%02d", season, round),
            Season:           season,
            Round:            round,
            // ...
            DataAsOfUTC:      now,
            SourceHash:       fmt.Sprintf("%d", r.MeetingKey),
        }
        meetings = append(meetings, m)
    }
    return meetings
}
```

Deterministic IDs (`2026-01`, `2026-02`, ...) derived from season and round. Deterministic round numbering from array position. A `DataAsOfUTC` timestamp stamped at normalization time — so downstream consumers always know when this data was last refreshed. A `SourceHash` from the meeting key for upsert idempotency.

This function is pure. Same input, same output (modulo the clock). No side effects. No database calls. It takes raw upstream data and produces clean domain objects. The poller calls it; the repository persists the result. Separation of concerns at the function level.

---

## The Calendar Service

The service layer is where business logic lives — and for the calendar, the most interesting logic is *next-race computation*.

The API contract (defined in the OpenAPI spec back in the planning phase) requires three things beyond the round list: `next_round`, `countdown_target_utc`, and `data_as_of_utc`. The first two don't exist in the database. They're computed at request time.

```go
func (s *Service) GetCalendar(ctx context.Context, season int) (*CalendarResponse, error) {
    meetings, err := s.repo.GetMeetingsBySeason(ctx, season)
    // ...
    for _, m := range meetings {
        // Build round DTOs...

        // Compute next round: first non-cancelled round with start >= now.
        if nextRound == 0 && !m.IsCancelled && m.StartDatetimeUTC.After(now) {
            nextRound = m.Round
            t := m.StartDatetimeUTC
            countdownTarget = &t
        }
    }

    return &CalendarResponse{
        Year:               season,
        DataAsOfUTC:        latestDataAsOf,
        NextRound:          nextRound,
        CountdownTargetUTC: countdownTarget,
        Rounds:             rounds,
    }, nil
}
```

The next-race algorithm is a single pass through the meetings. It finds the first future, non-cancelled round — skipping Bahrain (R4) and Saudi Arabia (R5) automatically — and reports its start time as the countdown target. On April 19, 2026, that's the Miami Grand Prix on May 4th. In a week's time, the same code will return the Spanish Grand Prix without any configuration change.

The `CountdownTargetUTC` field is a pointer — `*time.Time` — because after the season ends, there is no next race. The JSON serializes to `null`. The frontend handles this gracefully. No special-casing needed.

---

## The HTTP Handler

Thin. That's the design word for this handler.

```go
func (h *Handler) GetCalendar(w http.ResponseWriter, r *http.Request) {
    yearStr := r.URL.Query().Get("year")
    if yearStr == "" {
        http.Error(w, `{"error":"year query parameter is required"}`, http.StatusBadRequest)
        return
    }

    year, err := strconv.Atoi(yearStr)
    if err != nil || year < 1950 || year > 2100 {
        http.Error(w, `{"error":"invalid year parameter"}`, http.StatusBadRequest)
        return
    }

    resp, err := h.service.GetCalendar(r.Context(), year)
    // ...
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
}
```

Year validation — integers only, 1950 to 2100 — and delegation to the service. No business logic. No data transformation. No database awareness. The handler knows HTTP. The service knows calendars. The repository knows Cosmos DB. Each layer knows exactly one thing. *Ordo* — clear boundaries.

---

## The Frontend

The `CalendarPage` component is the first real React code in this project. Three sub-components in one file:

**`CalendarPage`** — The container. Calls `fetchCalendar(2026)` on mount, manages loading/error/data state, renders the table and the next-race card.

**`RaceRow`** — A table row. Applies CSS classes for cancelled and next-race rows. Renders a red "Cancelled" badge for Bahrain and Saudi Arabia.

**`NextRaceHighlight`** — A card above the table. Shows the next race name, circuit, country, and a countdown computed client-side from the `countdown_target_utc` timestamp. "15d 8h until lights out."

The typed API service — `calendarApi.ts` — defines `RaceMeetingDTO` and `CalendarResponse` interfaces that mirror the backend's JSON contract exactly. Field names match. Types match. The status field is a union type: `'scheduled' | 'cancelled' | 'completed' | 'unknown'`. TypeScript enforces the contract at compile time.

```typescript
export async function fetchCalendar(year: number): Promise<CalendarResponse> {
  return apiClient.get<CalendarResponse>(`/calendar?year=${year}`);
}
```

One line. Through the single API gateway established in Phase 2. No direct HTTP calls. No URL construction outside the service file. The *Decor* principle holds.

---

## The Tests

Fourteen tests across three layers. Every one passes.

**Four contract tests** (Go, `tests/contract/`):
- The calendar endpoint returns exactly 24 rounds for the 2026 season.
- Every round has all required fields: round number, race name, circuit, country, dates, status.
- Cancelled rounds (R4, R5) have `is_cancelled: true` and a `cancelled_label`.
- A request with a missing year parameter returns 400.

These tests run against a mock repository seeded with 24 rounds — no Cosmos DB, no network. They verify the API contract, not the infrastructure.

**Two integration tests** (Go, `tests/integration/`):
- The poll-to-cache-to-API flow: meetings ingested via the normalization layer are retrievable through the calendar service.
- Upsert idempotency: the same meeting upserted twice produces one record, not two.

These tests use an in-memory repository. They verify the data flow from ingest through storage to the API response.

**Four component tests** (Vitest + React Testing Library):
- The calendar table renders all rounds.
- Cancelled rounds display the "Cancelled" badge with the correct CSS class.
- The next-race card renders with the race name appearing in both the card and the table.
- The data freshness timestamp is displayed.

The component tests mock the API layer and verify rendering. They found a real bug during development: `getByText('Miami Grand Prix')` threw because Miami appears twice — once in the table row, once in the next-race card. The fix was `getAllByText`. A genuine edge case that only surfaces with a component that highlights one of its own table rows.

---

## What I Noticed in Phase 3

**The first vertical slice is the hardest.** Not because the code is complex — it isn't. Because it's the first time every layer has to agree. The domain model feeds the service, which shapes the DTO, which the handler serializes, which the frontend deserializes, which the component renders. If any layer's assumptions are wrong, the whole slice fails. Once this path works, every subsequent feature is a variation.

**Next-race computation is deceptively simple.** "Find the first future, non-cancelled round" sounds trivial. But the implementation has to handle: cancelled rounds that should be skipped, the end-of-season case where no future rounds exist (pointer field serializing to `null`), and the fact that "future" is relative to UTC time, not local time. Three edge cases in one line of iteration logic.

**The frontend tests caught a real problem.** The Miami Grand Prix duplication — table row plus highlight card — is exactly the kind of thing you don't notice in manual testing until a user reports it. The test framework flagged it immediately. Testing Library's `getByText` enforces uniqueness; `getAllByText` is the explicit acknowledgment that a value appears multiple times. That distinction matters.

**Fourteen tests for ten tasks is the right ratio.** Not every task needs its own test. The domain model and normalization layer are tested indirectly through the contract and integration tests. The handler is tested through the contract tests. The frontend API service is tested through the component tests. Testing at boundaries, not at every function, keeps the test suite focused and the feedback fast.

---

## The Scorecard

Phase 3 was ten tasks across the full vertical:

| Task | Layer | What |
|---|---|---|
| T015 | Test | Contract tests for GET /api/v1/calendar |
| T016 | Test | Integration tests for poll→cache→API flow |
| T017 | Test | Frontend component tests for CalendarPage |
| T018 | Backend | RaceMeeting domain model with status enum |
| T019 | Backend | OpenF1 meetings normalization layer |
| T020 | Backend | Calendar repository query + freshness tracking |
| T021 | Backend | Calendar service with next-race computation |
| T022 | Backend | GET /api/v1/calendar HTTP handler |
| T023 | Frontend | CalendarPage with table, badges, countdown |
| T024 | Frontend | Typed API service via apiClient gateway |

Nineteen files changed. Fourteen tests passing. One commit. One push.

---

## What's Next

Phase 4 is **User Story 2 — Countdown Widget and Race Details**, and the first real deployment to Azure. Read about it in [Day 3: From Localhost to the Cloud — Phase 4 and the First Real Deployment](day-3-phase-4-deployment.md).

Phase 5 is **User Story 3 — Cancelled Round Handling and Championship Standings**. Phase 6 is polish.

But the important thing about Phase 3 isn't what it built. It's what it proved. Data flows from Cosmos DB through Go through JSON through TypeScript into a React component, and every step is tested, typed, and traceable. The architecture carries.

Vitruvius would look at this and nod. Not because it's beautiful — it's a table and a card. But because the *arrangement* is sound, the *proportions* are right, and the *economy* is strict. Twenty-four rounds. One endpoint. One component. One clean vertical slice.

The calendar is up. Now we make it count down.

*Next: [Day 3: From Localhost to the Cloud — Phase 4 and the First Real Deployment](day-3-phase-4-deployment.md)*

---

*Previous: [Day 1: Laying the Foundation — Phase 2 and the Architecture That Carries Everything](day-1-phase-2-foundation.md)*

*The repo is public and every commit tells the story:*
[github.com/karlkuhnhausen/f1-race-intelligence](https://github.com/karlkuhnhausen/f1-race-intelligence)
