# Implementation Plan: Standings Overhaul

**Branch**: `007-standings-overhaul` | **Date**: 2026-05-03 | **Spec**: [spec.md](spec.md)  
**Input**: Feature specification from `/specs/007-standings-overhaul/spec.md`

---

## Summary

Replace the fictional Hyprace standings integration with real OpenF1 championship data and enhance the standings page with rich analytics. The Go backend removes all Hyprace dead code and ingests championship standings from OpenF1 beta endpoints (`/v1/championship_drivers`, `/v1/championship_teams`) triggered by the session poller's post-race finalization hook. Session results and starting grids provide expanded statistics (wins, podiums, DNFs, poles). New API endpoints serve progression data, head-to-head comparisons, and constructor driver breakdowns. The React frontend adds year picker, recharts-based progression charts, comparison panel, and expandable constructor rows. A backfill CLI extension populates data for all completed 2023–2026 Race and Sprint sessions.

---

## Technical Context

**Language/Version**: Go 1.25+ (backend), TypeScript 5.6 / React 18 (frontend)  
**Primary Dependencies**: Chi v5 router, Azure Cosmos DB SDK for Go, recharts 3.8 (existing — React charting library), Vitest 4.1  
**Storage**: Azure Cosmos DB serverless — `standings` container (championship snapshots, partition key `/season`), `sessions` container (session results, starting grids, partition key `/season`)  
**Testing**: `go test ./...` (backend), `npx vitest run` (frontend, pool: "threads")  
**Target Platform**: AKS 1.33 (existing deployment)  
**Project Type**: Web application (Go API + React SPA)  
**Performance Goals**: Standings page renders within 3 seconds; API responses <1s for current season standings; progression endpoint <2s for full season (~480 data points)  
**Constraints**: OpenF1 rate limit ≤1 req/s respected by poller and backfill; championship endpoints are beta (graceful fallback to cached data); no new dependencies  
**Scale/Scope**: ~20 drivers × ~24 races per season × 4 years (2023–2026) = ~2000 championship documents; ~20 drivers × ~24 races = ~480 session result documents per season; ~20 × ~24 = ~480 starting grid documents per season

---

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Gate | Status | Notes |
|------|--------|-------|
| **Stack gate** | ✅ PASS | Go backend, React frontend, Cosmos DB serverless, AKS — no new platform components |
| **Architecture gate** | ✅ PASS | Frontend calls only backend `/api/v1/standings/*` endpoints; OpenF1 calls are backend-only (session poller, backfill CLI) |
| **Data gate** | ✅ PASS | All OpenF1 championship, session result, and starting grid data cached in Cosmos DB before serving. Post-race event-driven ingestion with 2h finalization buffer. Backfill for historical seasons. No pass-through. |
| **Security gate** | ✅ PASS | No new secrets. OpenF1 is free-tier, no API key. Hyprace Key Vault references removed (net reduction). Managed Identity continues for Cosmos DB access. |
| **Network gate** | ✅ PASS | No changes to ingress TLS. Egress already allows `api.openf1.org`. Hyprace egress rule removed. |
| **Delivery gate** | ✅ PASS | No new Helm/Bicep resources. CI/CD unchanged: lint → test → build → push → deploy. Backfill is a manual post-deploy step (extends existing CLI). |
| **Observability gate** | ✅ PASS | Championship ingestion logs structured JSON: session key, data type, row count, duration, errors. API handlers log request timing. |
| **Dependency gate** | ✅ PASS | No new dependencies. Frontend reuses `recharts` (already justified in Feature 006). Backend uses only stdlib + existing packages. |
| **Spec authority gate** | ✅ PASS | All work items trace to FR-001–FR-020 in spec.md. |

---

## Project Structure

### Documentation (this feature)

```text
specs/007-standings-overhaul/
├── plan.md              # This file
├── research.md          # Phase 0 output — API exploration and design decisions
├── data-model.md        # Phase 1 output — entity definitions and Cosmos schema
├── quickstart.md        # Phase 1 output — setup and run guide
├── contracts/
│   └── api.md           # Phase 1 output — HTTP endpoint contracts
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
backend/
├── cmd/
│   ├── api/main.go                              # MODIFY: remove Hyprace poller, wire championship ingestion
│   └── backfill/main.go                         # MODIFY: add --championship flag
├── internal/
│   ├── standings/
│   │   ├── hyprace_client.go                    # DELETE: fictional poller
│   │   ├── championship_ingester.go             # NEW: OpenF1 championship data fetcher
│   │   └── stats_aggregator.go                  # NEW: compute wins/podiums/DNFs/poles from session data
│   ├── storage/
│   │   ├── repository.go                        # MODIFY: update interfaces, add new types
│   │   └── cosmos/client.go                     # MODIFY: update queries, add new methods
│   ├── api/
│   │   ├── router.go                            # MODIFY: add new route registrations
│   │   └── standings/
│   │       ├── handler.go                       # MODIFY: add new endpoint handlers
│   │       ├── service.go                       # MODIFY: add progression, comparison, breakdown logic
│   │       └── dto.go                           # MODIFY: expand DTOs for new fields and endpoints
│   └── ingest/
│       └── session_poller.go                    # MODIFY: trigger championship ingestion on finalization
└── tests/
    ├── contract/
    │   └── standings_contract_test.go           # MODIFY: update for new endpoints and response shapes
    └── unit/
        ├── championship_ingester_test.go        # NEW: test OpenF1 data fetching and transformation
        └── stats_aggregator_test.go             # NEW: test stats computation logic

frontend/
├── src/
│   ├── features/
│   │   └── standings/
│   │       ├── StandingsPage.tsx                # MODIFY: add year picker, chart view toggle, comparison
│   │       ├── standingsApi.ts                  # MODIFY: add new API client functions
│   │       ├── ProgressionChart.tsx             # NEW: recharts line chart component
│   │       ├── ComparisonPanel.tsx              # NEW: head-to-head comparison UI
│   │       ├── ConstructorBreakdown.tsx         # NEW: expandable driver breakdown
│   │       └── YearPicker.tsx                   # NEW: season selector component
│   └── features/
│       └── design-system/
│           └── StandingsTable.tsx               # MODIFY: add columns for podiums, DNFs, poles
└── tests/
    └── standings/
        ├── StandingsPage.test.tsx               # MODIFY: update for new UI elements
        ├── ProgressionChart.test.tsx            # NEW: chart rendering tests
        ├── ComparisonPanel.test.tsx             # NEW: comparison UI tests
        └── ConstructorBreakdown.test.tsx        # NEW: breakdown UI tests
```

**Structure Decision**: Web application structure. Extends existing backend/frontend layout with new files in `backend/internal/standings/` for ingestion logic, expanded `backend/internal/api/standings/` for API endpoints, and new frontend components in `frontend/src/features/standings/` for charts, comparisons, and breakdowns.

---

## Complexity Tracking

> No constitution violations. No entries needed.
