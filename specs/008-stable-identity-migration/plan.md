# Implementation Plan: Stable Identity Migration

**Branch**: `008-stable-identity-migration` | **Date**: 2026-05-05 | **Spec**: [spec.md](specs/008-stable-identity-migration/spec.md)
**Input**: Feature specification from `/specs/008-stable-identity-migration/spec.md`

## Summary

Introduce immutable `meeting_key` and `session_key` fields as stable identity anchors for session-related Cosmos DB documents. This eliminates data integrity issues caused by round-number shifts after mid-season cancellations. The `MeetingIndex` (built from calendar data) provides the authoritative round→meeting_key mapping. API layers prefer meeting_key-based queries with fallback to round-based queries for pre-migration data. A backfill CLI mode (`--stamp-meeting-keys`) retroactively populates these fields on existing documents.

**Status**: Phases 1-4 are fully implemented and committed. Remaining work is Phase 5 (backfill CLI) and Phase 6 (end-to-end validation).

## Technical Context

**Language/Version**: Go 1.25+  
**Primary Dependencies**: Chi v5 router, Azure Cosmos DB SDK for Go (`azcosmos`), Azure Identity SDK  
**Storage**: Azure Cosmos DB (serverless) — `calendar`, `sessions` containers  
**Testing**: `go test ./...` (backend), `npx vitest run` (frontend, unchanged)  
**Target Platform**: Linux containers on AKS 1.33  
**Project Type**: Backend-only CLI extension + API query path changes  
**Performance Goals**: Backfill processes ~500 documents without rate-limit violations against Cosmos DB  
**Constraints**: Idempotent backfill (re-runnable), no external API calls needed (uses existing cached calendar data)  
**Scale/Scope**: ~200-500 existing documents across sessions, session_results, and analysis containers for 2025-2026 seasons

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- ✅ Stack gate: Pure Go backend changes + Cosmos DB queries on AKS. No frontend changes.
- ✅ Architecture gate: No UI changes. All logic is backend-only (CLI tool + API service layer).
- ✅ Data gate: meeting_key/session_key are derived from cached OpenF1 data already in Cosmos DB calendar container. No new external API calls.
- ✅ Security gate: Backfill CLI uses same Managed Identity → Key Vault → Cosmos DB path as the main service.
- ✅ Network gate: No new ingress/egress paths. CLI runs as a Job in AKS using existing network policies.
- ✅ Delivery gate: Backfill packaged via existing Dockerfile. Deployable as Kubernetes Job via Helm.
- ✅ Observability gate: All backfill operations emit structured JSON logs via `slog`.
- ✅ Dependency gate: No new dependencies introduced. Uses existing `azcosmos` and `azidentity` packages.
- ✅ Spec authority gate: All work traces to FR-001 through FR-010 in the specification.

## Project Structure

### Documentation (this feature)

```text
specs/008-stable-identity-migration/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
backend/
├── cmd/backfill/main.go              # CLI entry point — add --stamp-meeting-keys mode
├── internal/
│   ├── domain/
│   │   └── meeting_index.go          # MeetingIndex + BuildMeetingIndex (already complete)
│   ├── api/
│   │   ├── rounds/service.go         # meeting_key-first query + round fallback (already complete)
│   │   └── analysis/service.go       # meeting_key-first query + round fallback (already complete)
│   └── storage/
│       ├── repository.go             # Session/SessionResult/Analysis types with MeetingKey fields (already complete)
│       └── cosmos/
│           ├── sessions.go           # GetSessionsByMeetingKey, GetSessionResultsByMeetingKey (already complete)
│           └── analysis.go           # GetSessionAnalysisByMeetingKey (already complete)
└── tests/
    └── unit/                         # Unit tests for backfill stamp logic
```

**Structure Decision**: Backend-only feature. All changes are in `backend/cmd/backfill/` (CLI) and `backend/internal/` (domain, storage, API layers). Phases 1-4 already committed on this branch added `domain/meeting_index.go`, `GetSessionsByMeetingKey`/`GetSessionResultsByMeetingKey`/`GetSessionAnalysisByMeetingKey` query methods, and meeting_key-first query logic in rounds and analysis services. Phase 5 adds the `--stamp-meeting-keys` CLI mode. Phase 6 validates everything end-to-end.

## Complexity Tracking

No constitution violations. All work uses existing stack (Go, Cosmos DB, AKS, Helm, slog).
