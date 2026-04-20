# Quickstart: Race Calendar and Championship Standings

## Prerequisites

- Go 1.25+
- Node.js 22+
- Docker
- Helm 3.x
- Azure CLI authenticated to target subscription
- Access to Cosmos DB (serverless), AKS, Key Vault, and ACR resources

## 1. Backend setup

```bash
cd backend
go mod download
go build -o bin/api cmd/api/main.go
```

Environment variables (set via Key Vault in production):

| Variable | Description |
|---|---|
| `BACKEND_LISTEN_ADDR` | Listen address (default `:8080`) |
| `COSMOS_ACCOUNT_ENDPOINT` | Cosmos DB account endpoint |
| `COSMOS_DATABASE_NAME` | Database name (`f1raceintelligence`) |
| `KEYVAULT_URI` | Azure Key Vault URI |

The backend starts two background pollers when `COSMOS_ACCOUNT_ENDPOINT` is set:
- **OpenF1 poller** — fetches race meetings every 5 minutes.
- **Hyprace poller** — fetches championship standings every 5 minutes (fictional API; returns empty data).

## 2. Frontend setup

```bash
cd frontend
npm ci
npm run dev      # Development server on :5173
npm run build    # Production build
```

All data flows through `src/services/apiClient.ts` → `/api/v1/*`. No direct calls to OpenF1 or Hyprace.

## 3. Run tests

```bash
# Backend (19 tests: 6 contract, 10 integration, 6 unit)
cd backend && go test -v -race ./...

# Frontend (16 tests: 4 calendar table, 5 countdown, 4 standings, 3 network boundary)
cd frontend && npx vitest run
```

### Validated results (2026-04-20)

| Check | Result |
|---|---|
| Backend tests | 19 passed |
| Frontend tests | 16 passed |
| Calendar rounds | 26 |
| Cancelled races | 2 (Bahrain GP R6, Saudi Arabian GP R7) |
| Next race | R8 (Miami GP) |
| Driver standings rows | 0 (Hyprace API fictional) |
| Constructor standings rows | 0 (Hyprace API fictional) |

## 4. Local run flow

1. Start backend: `COSMOS_ACCOUNT_ENDPOINT=<endpoint> go run cmd/api/main.go`
2. Start frontend: `VITE_API_BASE_URL=http://localhost:8080/api/v1 npm run dev`
3. Verify:
   - 26 rounds present in calendar.
   - Bahrain GP (R6) and Saudi Arabian GP (R7) rendered as cancelled.
   - Next-race selection skips cancelled rounds.
   - Countdown timer targets the next non-cancelled race.
   - Standings tables render (empty if Hyprace unavailable).

## 5. Deployment flow (AKS)

```bash
# Helm deploy (CI/CD does this automatically on master push)
helm upgrade --install f1-backend deploy/helm/backend \
  --namespace f1-race-intelligence --create-namespace \
  --set image.repository=<acr>/f1-backend \
  --set image.tag=<sha> \
  --set workloadIdentity.clientId=<client-id> \
  --set env.KEYVAULT_URI=<vault-uri> \
  --set env.COSMOS_ACCOUNT_ENDPOINT=<cosmos-endpoint>

helm upgrade --install f1-frontend deploy/helm/frontend \
  --namespace f1-race-intelligence --create-namespace \
  --set image.repository=<acr>/f1-frontend \
  --set image.tag=<sha>
```

### Live endpoints

| Endpoint | URL |
|---|---|
| Frontend | http://f1.20.171.233.61.nip.io/ |
| Backend API | http://api-f1.20.171.233.61.nip.io/api/v1/ |
| Healthz | http://api-f1.20.171.233.61.nip.io/healthz |

### CI/CD pipeline (`.github/workflows/ci-cd.yml`)

Pipeline stages: **lint → test → build → push → deploy**

- Push and deploy are gated to `main`/`master` branches only.
- Images are SHA-pinned (`github.sha` tag) for exact traceability.
- OIDC federation — no stored credentials.
- Helm `--wait` ensures rollback on failed health checks.

## 6. Monitoring

Deploy the Azure Monitor dashboard:

```bash
az deployment group create \
  --resource-group rg-f1raceintel \
  --template-file deploy/monitoring/azure-monitor-dashboard.json \
  --parameters logAnalyticsWorkspaceId=<workspace-id>
```

Dashboard panels: API latency (p50/p95/p99), poll success rate, HTTP error rate, pod health.
Alerts: p95 > 500ms, poll failures, pod restarts > 2.