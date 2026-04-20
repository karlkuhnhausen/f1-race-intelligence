# Implementation Plan: Race Results & Session Data

**Branch**: `003-race-session-results` | **Date**: 2026-04-19 | **Spec**: `/specs/003-race-session-results/spec.md`
**Input**: Feature specification from `/specs/003-race-session-results/spec.md`

## Summary

Add race results and session data to the F1 Race Intelligence Dashboard. Extend the existing OpenF1 poller to ingest session metadata and per-session results (race, qualifying, practice, sprint) via the `/v1/sessions`, `/v1/session_result`, and `/v1/drivers` endpoints. Persist session and result data in Cosmos DB alongside existing meeting documents. Expose a round detail API endpoint returning all session results for a given round. Add a frontend round detail page with client-side routing from the calendar table, displaying race results, qualifying results, and practice session times in separate sections with appropriate status indicators for in-progress or unavailable sessions.

## Technical Context

**Language/Version**: Go 1.25+ (backend), TypeScript with React 18+ (frontend)
**Primary Dependencies**: Chi router, Azure Cosmos DB SDK for Go, React + Vite; no new dependencies anticipated
**Storage**: Azure Cosmos DB (serverless) — new `sessions` and `session_results` document types partitioned by `season`
**Testing**: Go test (unit/integration/contract), Vitest (frontend unit), API contract tests
**Target Platform**: Linux containers on Azure Kubernetes Service (existing backend and frontend workloads)
**Project Type**: Web application extending existing API and UI services
**Performance Goals**: Round detail page P95 < 400ms; session ingestion completes within 5-minute poll window
**Constraints**: UI never calls external APIs; backend polls every 5 minutes; HTTPS at NGINX ingress; 3 req/s OpenF1 rate limit
**Scale/Scope**: Single-season (2026); up to 24 rounds × 7 session types × 20 drivers = ~3,360 result documents

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- Stack gate: Uses Go backend, React frontend, Cosmos DB, and AKS for production workloads.
- Architecture gate: UI communicates only with backend APIs; external API calls are backend-only.
- Data gate: OpenF1 integration includes Cosmos DB caching strategy, data freshness policy, and backfill behavior.
- Security gate: Secrets flow through Azure Key Vault with Managed Identity; no plaintext secret handling.
- Network gate: HTTPS termination at NGINX ingress and Azure Firewall egress policy are defined.
- Delivery gate: Kubernetes delivery uses Helm charts; GitHub Actions stages are lint -> test -> build -> push -> deploy.
- Observability gate: Structured JSON logging and Azure Monitor ingestion are specified.
- Dependency gate: Every added dependency includes explicit justification and maintenance owner.
- Spec authority gate: Implementation plan traces all major work items back to specification requirements.

Constitution gates status (initial):
- Stack gate: **PASS** — Go, React, Cosmos DB serverless, AKS; no new runtimes or databases
- Architecture gate: **PASS** — backend-only upstream integration; frontend calls `/api/v1/rounds/{round}` only
- Data gate: **PASS** — session data cached in Cosmos DB with `data_as_of_utc` freshness; piggybacks on existing 5-min poll
- Security gate: **PASS** — no new secrets needed; OpenF1 is free-tier without API keys; existing Key Vault + Managed Identity preserved
- Network gate: **PASS** — NGINX HTTPS ingress unchanged; OpenF1 already in Azure Firewall egress allow-list
- Delivery gate: **PASS** — extends existing Helm charts; no new Kubernetes resource types; same CI/CD pipeline
- Observability gate: **PASS** — session ingestion and result API operations include structured JSON logging
- Dependency gate: **PASS** — no new Go or npm dependencies anticipated; react-router may be added with justification
- Spec authority gate: **PASS** — all work items trace to FR-001–FR-016 and SC-001–SC-007

## Project Structure

### Documentation (this feature)

```text
specs/003-race-session-results/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   └── openapi.yaml
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```text
backend/
├── cmd/api/
│   └── main.go                    # Updated: wire SessionRepository + session poller
├── internal/
│   ├── api/
│   │   ├── router.go              # Updated: add /api/v1/rounds/{round} route
│   │   ├── calendar/              # Existing (unchanged)
│   │   ├── standings/             # Existing (unchanged)
│   │   └── rounds/                # NEW: round detail handler/service/dto
│   │       ├── dto.go
│   │       ├── service.go
│   │       └── handler.go
│   ├── domain/
│   │   ├── race_meeting.go        # Existing (unchanged)
│   │   └── session.go             # NEW: session domain types + session type enum
│   ├── ingest/
│   │   ├── openf1_poller.go       # Updated: add session + result polling
│   │   ├── meeting_transform.go   # Existing (unchanged)
│   │   └── session_transform.go   # NEW: transform OpenF1 sessions/results to storage types
│   ├── storage/
│   │   ├── repository.go          # Updated: add SessionRepository interface
│   │   └── cosmos/
│   │       ├── client.go          # Existing
│   │       ├── calendar_repository.go  # Existing (unchanged)
│   │       └── session_repository.go   # NEW: Cosmos DB session CRUD
│   ├── standings/                 # Existing (unchanged)
│   └── observability/             # Existing (unchanged)
└── tests/
    ├── unit/
    │   └── session_transform_test.go   # NEW
    ├── integration/
    │   └── session_ingestion_test.go   # NEW
    └── contract/
        └── rounds_contract_test.go     # NEW

frontend/
├── src/
│   ├── App.tsx                    # Updated: add routing for round detail page
│   ├── features/
│   │   ├── calendar/
│   │   │   ├── CalendarPage.tsx   # Updated: make rows clickable
│   │   │   └── calendarApi.ts     # Existing (unchanged)
│   │   ├── standings/             # Existing (unchanged)
│   │   └── rounds/                # NEW: round detail feature
│   │       ├── RoundDetailPage.tsx
│   │       ├── RaceResults.tsx
│   │       ├── QualifyingResults.tsx
│   │       ├── PracticeResults.tsx
│   │       └── roundApi.ts
│   └── services/
│       └── apiClient.ts           # Existing (unchanged)
└── tests/
    └── rounds/
        ├── RoundDetailPage.test.tsx    # NEW
        └── RaceResults.test.tsx        # NEW

deploy/
└── helm/
    ├── backend/                   # Existing (no changes expected)
    └── frontend/                  # Existing (no changes expected)
```

**Structure Decision**: Extends the existing web application structure established in feature 002. New backend code follows the existing `dto.go → service.go → handler.go` pattern in a new `rounds/` API domain package. New frontend code follows the existing `features/{domain}/` pattern. No new top-level directories.

## Complexity Tracking

No constitution violations. No complexity justifications needed.
