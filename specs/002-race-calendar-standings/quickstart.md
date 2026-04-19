# Quickstart: Race Calendar and Championship Standings

## Prerequisites

- Go 1.22+
- Node.js 20+
- Docker
- Azure CLI authenticated to target subscription
- Access to Cosmos DB (serverless), AKS, and Key Vault resources

## 1. Backend setup

1. Create backend service skeleton in `backend/` with Chi router.
2. Configure environment variables through Key Vault references (no plaintext secrets).
3. Implement scheduled workers:
   - OpenF1 meetings poll every 5 minutes.
   - Hyprace standings refresh every 5 minutes.
4. Persist normalized records in Cosmos DB with season partitioning.

## 2. Frontend setup

1. Create React app in `frontend/`.
2. Configure API base URL to backend service only.
3. Implement pages/components:
   - Race calendar list/table with cancelled visual state.
   - Upcoming race card with countdown timer.
   - Driver and constructor standings tables.

## 3. API contract validation

1. Implement endpoints from `contracts/openapi.yaml`.
2. Run contract tests against local backend.
3. Verify browser network tab has no OpenF1 or Hyprace direct calls.

## 4. Local run flow

1. Start backend with local Cosmos emulator or dev Cosmos account.
2. Start frontend and point to local backend URL.
3. Verify:
   - 24 rounds present.
   - Bahrain R4 and Saudi Arabia R5 rendered cancelled.
   - Next-race selection excludes cancelled rounds.
   - Standings tables contain required columns.

## 5. Deployment flow (AKS)

1. Build backend and frontend images.
2. Deploy with Helm charts:
   - `deploy/helm/backend`
   - `deploy/helm/frontend`
3. Enforce ingress TLS and outbound egress controls.
4. Confirm logs/metrics appear in Azure Monitor.