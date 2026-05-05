# Quickstart: Stable Identity Migration

## Prerequisites

- Go 1.25+
- Azure Cosmos DB connection (via Managed Identity or emulator)
- `COSMOS_ACCOUNT_ENDPOINT` environment variable set

## Build

```bash
cd backend && go build ./...
```

## Run Tests

```bash
cd backend && go test ./...
```

## Run Backfill (stamp meeting keys)

```bash
# Dry run — shows what would be updated without making changes
COSMOS_ACCOUNT_ENDPOINT=https://your-cosmos.documents.azure.com:443/ \
  go run ./cmd/backfill --season=2026 --stamp-meeting-keys --dry-run

# Actual run — updates documents in Cosmos DB
COSMOS_ACCOUNT_ENDPOINT=https://your-cosmos.documents.azure.com:443/ \
  go run ./cmd/backfill --season=2026 --stamp-meeting-keys
```

## Verify

After backfill, verify via the rounds API:

```bash
# A round that shifted due to cancellation should still show correct data
curl http://localhost:8080/api/v1/seasons/2026/rounds/5
```

## Key Files

| File | Purpose |
|------|---------|
| `backend/cmd/backfill/main.go` | CLI entry point with `--stamp-meeting-keys` flag |
| `backend/internal/domain/meeting_index.go` | MeetingIndex: round ↔ meeting_key resolution |
| `backend/internal/api/rounds/service.go` | Rounds API with meeting_key-first query pattern |
| `backend/internal/api/analysis/service.go` | Analysis API with meeting_key-first query pattern |
| `backend/internal/storage/repository.go` | Storage types with MeetingKey/SessionKey fields |
| `backend/internal/storage/cosmos/sessions.go` | Cosmos queries by meeting_key |
| `backend/internal/storage/cosmos/analysis.go` | Analysis queries by meeting_key |

## Implementation Phases

| Phase | Status | Description |
|-------|--------|-------------|
| 1 | ✅ Complete | Domain: MeetingIndex type + BuildMeetingIndex function |
| 2 | ✅ Complete | Storage: MeetingKey/SessionKey fields on types + ByMeetingKey query methods |
| 3 | ✅ Complete | API: Rounds service meeting_key-first query + round fallback |
| 4 | ✅ Complete | API: Analysis service meeting_key-first query + round fallback |
| 5 | 🔲 Pending | CLI: `--stamp-meeting-keys` backfill mode |
| 6 | 🔲 Pending | Validation: End-to-end testing, regression verification |
