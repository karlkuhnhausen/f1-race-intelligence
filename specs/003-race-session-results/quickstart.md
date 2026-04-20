# Quickstart: Race Results & Session Data

## Prerequisites

- Go 1.25+
- Node.js 22+
- Docker
- Helm 3.x
- Azure CLI authenticated to target subscription
- Access to Cosmos DB (serverless), AKS, Key Vault, and ACR resources
- Feature 002 (race-calendar-standings) fully deployed and operational

## 1. Backend changes

### New files to create

```text
backend/internal/domain/session.go           # Session type enums and domain types
backend/internal/api/rounds/dto.go           # RoundDetailResponse, SessionDetailDTO, SessionResultDTO
backend/internal/api/rounds/service.go       # Round detail service (fetches from SessionRepository)
backend/internal/api/rounds/handler.go       # HTTP handler for GET /api/v1/rounds/{round}
backend/internal/ingest/session_transform.go # Transform OpenF1 session/result data to storage types
backend/internal/storage/cosmos/session_repository.go  # Cosmos DB CRUD for sessions and results
```

### Files to modify

```text
backend/internal/storage/repository.go       # Add SessionRepository interface
backend/internal/api/router.go               # Add /api/v1/rounds/{round} route
backend/internal/ingest/openf1_poller.go     # Extend poll() to fetch sessions + results
backend/cmd/api/main.go                      # Wire SessionRepository and inject into router
```

### New backend endpoint

```
GET /api/v1/rounds/{round}?year=2026
```

Returns `RoundDetailResponse` (see contracts/openapi.yaml) with meeting metadata and all session results.

### Running locally

```bash
cd backend
go mod download
go build -o bin/api cmd/api/main.go
COSMOS_ACCOUNT_ENDPOINT=<endpoint> COSMOS_DATABASE_NAME=f1raceintelligence ./bin/api
```

### Running tests

```bash
cd backend
go test ./tests/unit/...
go test ./tests/integration/...
go test ./tests/contract/...
```

## 2. Frontend changes

### New dependency

```bash
cd frontend
npm install react-router-dom
```

Justification: Required for URL-based client-side routing to support bookmarkable round detail pages and browser back/forward navigation (FR-009, FR-013, FR-014).

### New files to create

```text
frontend/src/features/rounds/RoundDetailPage.tsx    # Round detail page component
frontend/src/features/rounds/RaceResults.tsx         # Race results table component
frontend/src/features/rounds/QualifyingResults.tsx   # Qualifying results table component
frontend/src/features/rounds/PracticeResults.tsx     # Practice results table component
frontend/src/features/rounds/roundApi.ts             # API client for round detail endpoint
```

### Files to modify

```text
frontend/src/App.tsx                                 # Replace state-based nav with react-router
frontend/src/features/calendar/CalendarPage.tsx      # Make round rows clickable (FR-013)
```

### Running locally

```bash
cd frontend
npm ci
npm run dev      # Development server on :5173
```

### Running tests

```bash
cd frontend
npm test
```

## 3. API contract reference

See `specs/003-race-session-results/contracts/openapi.yaml` for the complete OpenAPI specification.

Key response shape for `GET /api/v1/rounds/{round}?year=2026`:

```json
{
  "year": 2026,
  "round": 3,
  "race_name": "Australian Grand Prix",
  "circuit_name": "Albert Park",
  "country_name": "Australia",
  "start_datetime_utc": "2026-03-20T05:00:00Z",
  "end_datetime_utc": "2026-03-22T07:00:00Z",
  "status": "completed",
  "is_cancelled": false,
  "sessions": [
    {
      "session_type": "practice1",
      "session_name": "Practice 1",
      "status": "completed",
      "date_start_utc": "...",
      "date_end_utc": "...",
      "data_as_of_utc": "...",
      "results": [
        {
          "position": 1,
          "driver_number": 1,
          "driver_name": "Max VERSTAPPEN",
          "driver_acronym": "VER",
          "team_name": "Red Bull Racing",
          "number_of_laps": 25,
          "best_lap_time": 78.234,
          "gap_to_fastest": 0
        }
      ]
    },
    {
      "session_type": "race",
      "session_name": "Race",
      "status": "completed",
      "date_start_utc": "...",
      "date_end_utc": "...",
      "data_as_of_utc": "...",
      "results": [
        {
          "position": 1,
          "driver_number": 1,
          "driver_name": "Max VERSTAPPEN",
          "driver_acronym": "VER",
          "team_name": "Red Bull Racing",
          "number_of_laps": 58,
          "finishing_status": "Finished",
          "race_time": 5234.567,
          "gap_to_leader": null,
          "points": 26,
          "fastest_lap": true
        }
      ]
    }
  ]
}
```

## 4. Deployment

No changes to Helm charts or Kubernetes resources are expected. The new endpoint is served by the existing backend deployment. Frontend changes are served by the existing nginx container.

Standard deployment pipeline: `lint → test → build → push → deploy` via GitHub Actions.

## 5. End-to-end validation notes

### Phases 1–4 (completed)
- Backend compiles, all domain types and session repository wired
- Round detail API returns sessions with results grouped by type
- Frontend renders race results with classified/non-classified separation
- Calendar rows link to round detail pages, back navigation works

### Phases 5–7 (completed)
- **QualifyingResults** component renders Q1/Q2/Q3 times, shows "—" for segments not reached
- **PracticeResults** component renders best lap, gap to fastest, and lap count
- **RoundDetailPage** routes qualifying/sprint_qualifying → QualifyingResults, practice1/2/3 → PracticeResults
- Upcoming sessions show "Not yet available" instead of an empty results table
- Backend unit tests: session type mapping, slug generation, type predicates, transform functions (18 tests)
- Backend integration tests: session ingestion round-trip, upsert idempotency, empty round handling (4 tests)
- Backend integration tests: structured log schema for session ingestion, round detail API, and upsert errors (3 tests)
- Frontend network boundary test extended to explicitly scan rounds feature files
- Helm charts verified — no changes needed (same backend service, same Cosmos DB connection)

### Test counts
- Backend unit: 18 tests (next_race_selector + session_transform)
- Backend integration: 19 tests (cache flow, cancellations, countdown, log schema, session ingestion)
- Backend contract: 11 tests (calendar, rounds, standings)
- Frontend: 30 tests (calendar, countdown, standings, race results, round detail, network boundary)
