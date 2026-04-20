# Implementation Plan: Race Calendar and Championship Standings

**Branch**: `002-race-calendar-standings` | **Date**: 2026-04-19 | **Spec**: `/specs/002-race-calendar-standings/spec.md`
**Input**: Feature specification from `/specs/002-race-calendar-standings/spec.md`

## Summary

Deliver the foundation dashboard capability in two slices: (1) 2026 race calendar with upcoming-race countdown and hard-coded cancellation handling for Bahrain R4 and Saudi Arabia R5, and (2) driver/constructor standings surfaced through backend-owned APIs. The Go backend (Chi) polls OpenF1 every 5 minutes, persists and serves cache-first data from Cosmos DB serverless, and merges standings from Hyprace/OpenF1 identity data. The React frontend consumes backend APIs only and renders calendar, cancellation indicators, countdown, and standings tables.

## Technical Context

**Language/Version**: Go 1.22+ (backend), TypeScript with React 18+ (frontend)  
**Primary Dependencies**: Chi router, Azure Cosmos DB SDK for Go, minimal HTTP client libs, React + Vite, table/time formatting utilities only as justified  
**Storage**: Azure Cosmos DB (serverless) for normalized race meetings, source metadata, and refresh timestamps  
**Testing**: Go test (unit/integration), frontend unit tests (Vitest), API contract tests, smoke tests in AKS deployment pipeline  
**Target Platform**: Linux containers on Azure Kubernetes Service (two workloads: backend and frontend)
**Project Type**: Web application with separate API and UI services  
**Performance Goals**: Calendar P95 < 300ms, standings P95 < 350ms, poll success >= 95% daily  
**Constraints**: UI never calls external APIs; backend polls every 5 minutes; HTTPS at NGINX ingress; Azure Firewall egress allow-list; Key Vault + Managed Identity for secrets  
**Scale/Scope**: Single-season (2026) read-heavy dashboard pages; 24 rounds; two standings tables

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
- Stack gate: PASS (Go, React, Cosmos DB serverless, AKS)
- Architecture gate: PASS (backend-only upstream integration)
- Data gate: PASS (poll + cache-first + freshness metadata)
- Security gate: PASS (Key Vault + Managed Identity requirement)
- Network gate: PASS (NGINX HTTPS + Azure Firewall egress)
- Delivery gate: PASS (Helm + GitHub Actions stage order preserved)
- Observability gate: PASS (structured JSON + metrics)
- Dependency gate: PASS (explicit minimal-dependency policy)
- Spec authority gate: PASS (FR/SC mapping in artifacts below)

## Project Structure

### Documentation (this feature)

```text
specs/002-race-calendar-standings/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)
<!--
  ACTION REQUIRED: Replace the placeholder tree below with the concrete layout
  for this feature. Delete unused options and expand the chosen structure with
  real paths (e.g., apps/admin, packages/something). The delivered plan must
  not include Option labels.
-->

```text
backend/
├── cmd/api/
├── internal/
│   ├── api/
│   ├── domain/
│   ├── ingest/
│   ├── storage/
│   ├── standings/
│   └── observability/
└── tests/
  ├── unit/
  ├── integration/
  └── contract/

frontend/
├── src/
│   ├── app/
│   ├── pages/
│   ├── features/
│   │   ├── calendar/
│   │   └── standings/
│   └── services/
└── tests/

deploy/
└── helm/
  ├── backend/
  └── frontend/

.github/
└── workflows/
  └── ci-cd.yml
```

**Structure Decision**: Use a web application split with dedicated backend and frontend services plus Helm deployment artifacts. This matches the constitution's three-tier rule, isolates upstream integrations in backend code, and supports separate AKS container workloads.

## Phase 0: Research and Clarifications

- Polling and ingestion behavior documented in `research.md` with failure fallback semantics.
- Hyprace/OpenF1 blending assumptions and fallback response behavior documented.
- Cosmos DB serverless partitioning and freshness metadata strategy documented.

## Phase 1: Design and Contracts

- Domain entities defined in `data-model.md` with lifecycle/status rules.
- Backend REST contracts captured in `contracts/openapi.yaml`.
- Local/CI execution flow documented in `quickstart.md`.

## Traceability Matrix (Spec -> Plan)

- FR-001/FR-008/FR-009 -> backend ingest scheduler, Cosmos cache schema, poll metrics/logging.
- FR-002/FR-003/FR-004/FR-014 -> calendar query API, cancellation flags, next-race computation.
- FR-005 -> frontend data layer limited to backend API base URL.
- FR-006/FR-007 -> standings aggregation services + endpoints.
- FR-010/FR-011/FR-012 -> Helm values, network policy, secret provider integration.
- FR-013 -> dependency review checklist in implementation tasks.
- SC-001..SC-007 -> contract tests, integration tests, and runtime SLO dashboards.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |
