# Quickstart: Standings Overhaul

## 1. Backend setup

```bash
cd backend
go build ./...
go test ./...
```

### Environment variables

| Variable | Description |
|----------|-------------|
| `COSMOS_ACCOUNT_ENDPOINT` | Cosmos DB account endpoint |
| `COSMOS_DATABASE_NAME` | Database name (`f1raceintelligence`) |
| `KEYVAULT_URI` | Azure Key Vault URI |

The backend starts the session poller which now also triggers championship data ingestion after Race and Sprint sessions are finalized (2-hour buffer post-session-end).

### Backfill championship data

```bash
# Backfill all completed 2026 race sessions
go run cmd/backfill/main.go --season=2026 --championship

# Backfill a historical season
go run cmd/backfill/main.go --season=2025 --championship

# Dry run (log without writing)
go run cmd/backfill/main.go --season=2026 --championship --dry-run
```

## 2. Frontend setup

```bash
cd frontend
npm ci
npm run dev      # Development server on :5173
npm run build    # Production build
```

All data flows through `src/services/apiClient.ts` → `/api/v1/*`. No direct calls to OpenF1.

## 3. Run tests

```bash
# Backend
cd backend && go test -v -race ./...

# Frontend
cd frontend && npx vitest run
```

## 4. API endpoints

| Endpoint | Purpose |
|----------|---------|
| `GET /api/v1/standings/drivers?year=2026` | Driver standings with stats |
| `GET /api/v1/standings/constructors?year=2026` | Constructor standings with stats |
| `GET /api/v1/standings/drivers/progression?year=2026` | Driver points progression |
| `GET /api/v1/standings/constructors/progression?year=2026` | Constructor points progression |
| `GET /api/v1/standings/drivers/compare?year=2026&driver1=12&driver2=63` | Driver head-to-head |
| `GET /api/v1/standings/constructors/compare?year=2026&team1=Mercedes&team2=McLaren` | Constructor head-to-head |
| `GET /api/v1/standings/constructors/{team}/drivers?year=2026` | Constructor driver breakdown |

## 5. Key changes from previous implementation

| Before (Feature 002) | After (Feature 007) |
|-----------------------|---------------------|
| Hyprace poller (fictional, returned empty) | OpenF1 championship ingestion (real data) |
| 5-minute standing poll interval | Event-driven: ingest after session finalization |
| No progression data | Per-race snapshots enabling progression charts |
| Only position, name, points, wins | Position, name, points, wins, podiums, DNFs, poles |
| Current year only | Historical seasons (2023–current) |
| No comparisons | Head-to-head comparison endpoint |
| No constructor breakdown | Constructor → driver drill-down |

## 6. Data flow

```
OpenF1 API                    Backend                         Cosmos DB
───────────                   ───────                         ─────────
/championship_drivers ──────► Session poller (on finalize) ──► standings container
/championship_teams ────────► (2h after session ends)       ──► standings container
/session_result ────────────►                               ──► sessions container
/starting_grid ─────────────►                               ──► sessions container
/drivers ───────────────────► (already ingested by poller)  ──► sessions container

                              API handlers ◄──────────────────── Query + aggregate
                              /standings/* 
                                   │
                                   ▼
                              Frontend (React)
                              StandingsPage, ProgressionChart,
                              ComparisonPanel, ConstructorBreakdown
```

## 7. Cosmos DB containers

| Container | Partition Key | New Document Types |
|-----------|---------------|--------------------|
| `standings` | `/season` | `championship_driver`, `championship_team` |
| `sessions` | `/season` | `session_result`, `starting_grid` |
