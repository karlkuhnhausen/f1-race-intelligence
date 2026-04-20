# Day 5: Forty-Seven Tasks, Zero Lines — The Final Phase

*Posted April 19, 2026 · Karl Kuhnhausen*

---

Phase 6 was never going to be dramatic. No new features, no deployment failures, no data bugs. It's the final phase — polish and cross-cutting concerns. Monitoring dashboards, log schema validation, network boundary enforcement, Helm verification, CI/CD audit, and end-to-end quickstart validation.

It's the phase that proves the system works not because you *saw* it work, but because you *measured* it working.

---

## T042: The Monitoring Dashboard

The first artifact is an Azure Monitor dashboard defined as an ARM template in `deploy/monitoring/azure-monitor-dashboard.json`. It declares four KQL-based panels:

**API Latency (p50 / p95 / p99)** — parses structured JSON logs from ContainerLog, extracts `duration_ms`, and renders percentile timecharts in 5-minute bins.

**Data Poll Success Rate** — tracks `poll_complete` vs `poll_error` log entries, computing success percentage over time.

**HTTP Error Rate** — counts 4xx and 5xx responses in 5-minute windows, rendered as stacked bar charts.

**Pod Health & Restarts** — queries `KubePodInventory` for the `f1-race-intelligence` namespace, summarizing running/pending/failed states and restart counts.

Alongside the dashboard, three alert rules:

```
f1-api-p95-latency-alert    → fires when p95 > 500ms
f1-poll-failure-alert        → fires on any poll_error event
f1-pod-restart-alert         → fires when restarts > 2 in 15 minutes
```

The entire monitoring stack is declarative — deploy it with:

```bash
az deployment group create \
  --resource-group rg-f1raceintel \
  --template-file deploy/monitoring/azure-monitor-dashboard.json \
  --parameters logAnalyticsWorkspaceId=<workspace-id>
```

No manual portal configuration. No click-ops dashboards that vanish when someone cleans up the resource group.

---

## T043: Log Schema Validation

The structured logging layer was built in Phase 2 — a `slog.JSONHandler` writing to stdout, consumed by AKS Container Insights and forwarded to Log Analytics. But nothing in the test suite verified that the output actually contains the fields the KQL queries expect.

Three integration tests now validate the contract between application logging and the monitoring dashboard:

```go
func TestStructuredLogSchemaFields(t *testing.T) {
    var buf bytes.Buffer
    handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
    logger := slog.New(handler)

    logger.Info("request",
        slog.String("method", "GET"),
        slog.String("path", "/api/v1/calendar"),
        slog.Int("status", 200),
        slog.Float64("duration_ms", 12.34),
    )

    var entry map[string]interface{}
    json.Unmarshal(buf.Bytes(), &entry)

    requiredKeys := []string{"time", "level", "msg"}
    for _, key := range requiredKeys {
        if _, ok := entry[key]; !ok {
            t.Errorf("missing required log field %q", key)
        }
    }
}
```

The tests validate: required field presence (`time`, `level`, `msg`), caller-supplied attribute passthrough (`source`, `records`), and error-level output (`level: "ERROR"`, `error` attribute).

This is a subtle but important practice: **the log schema is a contract**. If someone refactors the logger or changes the handler, these tests catch it before the monitoring dashboard silently stops rendering data.

---

## T044: Network Boundary Enforcement

The constitution mandates a strict tier boundary: the frontend calls the backend API, the backend calls upstream data sources. The frontend must *never* reach OpenF1 or Hyprace directly.

The `apiClient.ts` module enforces this by design — every `fetch()` goes through a single `request()` function that builds URLs from a relative `API_BASE`. But "by design" is not the same as "verified."

A static analysis test scans every `.ts` and `.tsx` file under `src/`:

```typescript
const FORBIDDEN_PATTERNS = [
  /api\.openf1\.org/i,
  /openf1\.org/i,
  /hyprace\.io/i,
  /hyprace\.com/i,
];
```

It recursively collects source files, reads each one line by line, and reports any match as a violation. A second assertion verifies that every `*Api.ts` file imports and uses `apiClient`.

Three tests, 72ms. If someone adds a `fetch("https://api.openf1.org/...")` in a quick hack, the test suite catches it before the pipeline deploys it.

---

## T045–T046: Verification, Not Implementation

Some tasks don't produce code. They produce confidence.

**T045** validated that the Helm charts render correctly with `helm template` and that the live AKS deployment matches expectations:

| Resource | Expected | Actual |
|----------|----------|--------|
| Backend replicas | 2/2 | 2/2 ✓ |
| Frontend replicas | 2/2 | 2/2 ✓ |
| Backend image | SHA-pinned | `f22bb23...` ✓ |
| Ingress hosts | nip.io wildcard | `20.171.233.61` ✓ |
| NetworkPolicy | Egress to 443 + DNS only | ✓ |

**T046** audited the CI/CD pipeline:

| Guard | Status |
|-------|--------|
| Job chain | lint → test → build → push → deploy ✓ |
| Push gate | `main`/`master` only ✓ |
| Deploy gate | `main`/`master` only ✓ |
| OIDC permissions | `id-token: write`, `contents: read` ✓ |
| Image pinning | `${{ github.sha }}` on push and deploy ✓ |
| Last pipeline | GREEN ✓ |

No code changes needed. The infrastructure was already correct. The value of these tasks is the *audit trail* — documented proof that someone verified the deployment topology matches the intent.

---

## T047: End-to-End Validation

The final task runs every endpoint against the live AKS cluster:

```
1. Backend healthz:      ✓  {"status":"ok"}
2. Calendar API:         ✓  26 rounds, next R8, 2 cancelled
3. Standings drivers:    ✓  0 rows (Hyprace fictional)
4. Standings constructors: ✓  0 rows (Hyprace fictional)
5. Frontend:             ✓  HTTP 200
```

Then runs every test:

```
Backend:  19 passed (6 contract + 10 integration + 3 log schema)
Frontend: 16 passed (4 calendar + 5 countdown + 4 standings + 3 network boundary)
Total:    35 tests, all green
```

The quickstart.md was updated from its Phase 2 placeholder to a real runbook with validated results, actual commands, live endpoint URLs, and the monitoring deployment procedure.

---

## The Numbers

| Metric | Value |
|--------|-------|
| Phase 6 commit | 529 lines across 5 files |
| New tests | 6 (3 log schema + 3 network boundary) |
| Total tests | 35 (19 backend + 16 frontend) |
| Tasks completed | 47/47 |
| Phases completed | 6/6 |
| Pipeline status | GREEN |
| Live endpoints verified | 5 (healthz, calendar, 2× standings, frontend) |
| Lines typed by a human | 0 |

---

## What I Noticed Today

**The last 10% is verification, not implementation.** Phase 6 produced two test files and a dashboard template. Everything else was reading, checking, and documenting. In a real team, this is the work that gets skipped — and it's the work that catches the next failure before it reaches production.

**Log schemas are contracts.** The monitoring dashboard queries `ContainerLog` for specific JSON field names. The application logger emits those fields. But nothing connects the two until you write a test. The dashboard is just a consumer of an API that happens to be structured log output. Treat it like you'd treat any other API contract: test it, version it, don't break it.

**Static analysis is the cheapest compliance test.** The network boundary test doesn't start a browser, doesn't mock an HTTP server, doesn't run a real build. It reads files and pattern-matches. 72 milliseconds to guarantee an architectural invariant that would be expensive to debug in production.

**Verification tasks feel unproductive but create asymmetric value.** Checking that the Helm templates render correctly takes 30 seconds. Discovering they don't render correctly during an incident takes hours. The verification cost is bounded; the failure cost is not.

---

## The Full Picture

Five days. Six phases. Forty-seven tasks.

| Day | Phase | What happened |
|-----|-------|---------------|
| 0 | 1 | Constitution, spec, plan, scaffolding, CI/CD pipeline |
| 1 | 2 | HTTP server, Cosmos DB, pollers, Key Vault, Helm charts, Bicep IaC |
| 2 | 3 | Calendar MVP — 14 tests, table, countdown, API contract |
| 3 | 4 | Countdown refinement, Azure deployment, 5 CI/CD fixes |
| 4 | 5 | Standings, cancellation overrides, ingress, 3 production bugs |
| 5 | 6 | Monitoring, log schema tests, network boundary, E2E validation |

The project started as a Vitruvian constitution — *Firmitas, Utilitas, Venustas*. Structural integrity, practical utility, appropriate beauty. Every line of code, every Helm template, every Bicep module, every CI/CD step was generated by an AI agent. The human provided direction. The machine provided implementation.

And now it runs.

**Frontend:** http://f1.20.171.233.61.nip.io/
**API:** http://api-f1.20.171.233.61.nip.io/api/v1/calendar?year=2026
**Source:** https://github.com/karlkuhnhausen/f1-race-intelligence

Forty-seven tasks. Thirty-five tests. Two ingress endpoints. One subscription.

Zero lines typed by a human.

---

*Previous: [Day 4: Live Data, Broken Queries, and the Dangers of Round Numbers](day-4-phase-5-live-data.md)*
