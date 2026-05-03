# Quickstart: Session Deep Dive Page (Feature 006)

## Prerequisites

- Go 1.25+ (`go version`)
- Node.js 20+ with npm (`node --version`)
- golangci-lint in PATH (`export PATH="$HOME/go/bin:$PATH"`)
- Access to the `006-session-deep-dive` branch

## Branch Setup

```bash
git checkout 006-session-deep-dive
```

## Backend Development

### Build and Test

```bash
cd backend
go build ./...
go test ./...
```

### Lint

```bash
export PATH="$HOME/go/bin:$PATH"
cd backend
golangci-lint run ./...
```

### Key Files in This Feature

| File | Change |
|------|--------|
| `backend/internal/domain/analysis.go` | **New** — domain types for 5 analysis data categories |
| `backend/internal/storage/repository.go` | Add `AnalysisRepository` interface |
| `backend/internal/storage/cosmos/analysis.go` | **New** — Cosmos implementation of `AnalysisRepository` |
| `backend/internal/ingest/analysis.go` | **New** — OpenF1 fetchers for position/intervals/stints/pit/overtakes + aggregation |
| `backend/internal/ingest/session_poller.go` | Trigger analysis ingestion at session finalization (Race/Sprint only) |
| `backend/internal/api/analysis/handler.go` | **New** — HTTP handler for analysis endpoint |
| `backend/internal/api/analysis/service.go` | **New** — service layer for analysis data |
| `backend/internal/api/analysis/dto.go` | **New** — response DTOs |
| `backend/internal/api/router.go` | Register analysis route |
| `backend/cmd/backfill/main.go` | Add `--analysis` flag |

### Running the Backfill CLI (Post-Deployment)

```bash
# Build the backfill binary
cd backend
go build -o bin/backfill ./cmd/backfill/

# Dry-run — see what sessions would be processed:
./bin/backfill --analysis --dry-run --season=2026

# Run for real (production post-deploy):
./bin/backfill --analysis --season=2026

# Run both race-control AND analysis backfill:
./bin/backfill --season=2026 --analysis
```

Environment variables:
- `COSMOS_ACCOUNT_ENDPOINT` — Cosmos DB endpoint URL
- `COSMOS_KEY` or Managed Identity — authentication

### Testing the Analysis Ingest

```bash
cd backend
go test ./internal/ingest/... -run Analysis -v
```

### Testing the Analysis API Handler

```bash
cd backend
go test ./internal/api/analysis/... -v
```

### Testing the Analysis Repository

```bash
cd backend
go test ./internal/storage/cosmos/... -run Analysis -v
```

## Frontend Development

### Install Dependencies

```bash
cd frontend
npm install
```

**Note**: This feature adds `recharts` as a new dependency. Run `npm install` after checking out the branch.

### Run Dev Server

```bash
cd frontend
npm run dev
```

### Run Tests

```bash
cd frontend
npx vitest run                    # all tests
npx vitest run --reporter=verbose # verbose output
npx vitest run tests/analysis/    # analysis tests only
```

### Type Check

```bash
cd frontend
npx tsc --noEmit
```

### Key Frontend Files

| File | Description |
|------|-------------|
| `frontend/src/features/analysis/AnalysisPage.tsx` | Main analysis page layout |
| `frontend/src/features/analysis/PositionChart.tsx` | Position chart (recharts LineChart) |
| `frontend/src/features/analysis/GapToLeaderChart.tsx` | Gap progression chart |
| `frontend/src/features/analysis/TireStrategyChart.tsx` | Tire compound swimlane |
| `frontend/src/features/analysis/PitStopTimeline.tsx` | Pit stop timing scatter chart |
| `frontend/src/features/analysis/analysisApi.ts` | API client |
| `frontend/src/features/analysis/analysisTypes.ts` | TypeScript types |
| `frontend/src/features/rounds/RoundDetailPage.tsx` | "View Analysis" button addition |
| `frontend/src/app/routes.tsx` | New route registration |

### Navigating to the Analysis Page (Dev)

1. Start the frontend dev server (`npm run dev`)
2. Navigate to a round detail page: `http://localhost:5173/rounds/4?year=2026`
3. Click "View Analysis" on a completed Race or Sprint session
4. Or navigate directly: `http://localhost:5173/rounds/4/sessions/race/analysis?year=2026`

## API Endpoint

```
GET /api/v1/rounds/{round}/sessions/{type}/analysis?year={year}
```

- `{round}` — round number (1-24)
- `{type}` — `"race"` or `"sprint"` only
- `year` — optional, defaults to 2026

Returns 200 with analysis data or 404 if not yet available.

## Design System

Charts use the existing design system colors:
- Team colors for driver lines (from existing team color mapping)
- Compound colors: Soft=red, Medium=yellow, Hard=white, Intermediate=green, Wet=blue
- Dark background theme consistent with existing UI
