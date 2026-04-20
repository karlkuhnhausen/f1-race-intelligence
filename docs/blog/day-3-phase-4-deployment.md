# Day 3: From Localhost to the Cloud — Phase 4 and the First Real Deployment

*Posted April 19, 2026 · Karl Kuhnhausen*

---

For three days, everything ran in tests. Contract tests with mock repositories. Integration tests with in-memory stores. Component tests with fake timers. The architecture was validated, the data flow was proven, and all 14 tests passed — but nothing had ever run on a real server.

Today, the F1 Race Intelligence Dashboard went live on Azure Kubernetes Service. Real pods. Real containers. Real HTTP traffic. The CI/CD pipeline built both Docker images, pushed them to Azure Container Registry, deployed them via Helm, and the health endpoint responded with `{"status":"ok"}`.

This is the story of Phase 4, the countdown refinement — and the far more dramatic story of what it took to get the pipeline green and the infrastructure operational.

---

## Phase 4: The Pure Function That Changed Everything

Phase 4 is about User Story 2: making the countdown widget production-quality. But the real contribution isn't the widget — it's the architectural pattern that emerged.

In Phase 3, next-race computation was inline in the calendar service. It worked, but it was tangled with the HTTP response construction. You couldn't test the selection logic without wiring up a full service and making an HTTP request. Phase 4 extracts it into a pure function:

```go
type NextRaceResult struct {
    Round           int
    CountdownTarget time.Time
    Found           bool
}

func SelectNextRace(meetings []RaceMeeting, now time.Time) NextRaceResult {
    for _, m := range meetings {
        if m.IsCancelled {
            continue
        }
        if m.Status == StatusCancelled {
            continue
        }
        if !m.StartDatetimeUTC.After(now) {
            continue
        }
        return NextRaceResult{
            Round:           m.Round,
            CountdownTarget: m.StartDatetimeUTC,
            Found:           true,
        }
    }
    return NextRaceResult{}
}
```

No dependencies. No database. No clock. No HTTP context. Just a slice of meetings and a point in time. The function returns the first future, non-cancelled round — skipping both the `IsCancelled` boolean flag and the `StatusCancelled` enum value, because the data model allows either to be set independently.

This is the constitution's *Firmitas* principle in code. The function is structurally sound because it depends on nothing that can break. Same input, same output, every time. The `now` parameter is injected, not read from the system clock, which makes every test deterministic.

---

## Six Unit Tests for One Function

The pure function earned six dedicated unit tests — more than any other function in the codebase:

1. **SkipsCancelled** — Bahrain and Saudi Arabia (both `IsCancelled: true`) are skipped; Miami is returned.
2. **SkipsPastRounds** — Completed rounds with dates before `now` are skipped; the first future round is returned.
3. **TieBreakLowerRoundWins** — If two rounds share the same start time, the one appearing first in the slice (lower round) wins.
4. **NoFutureRounds** — After the season ends, `Found` is `false` and `Round` is 0.
5. **EmptySlice** — A nil input returns the zero value. No panic.
6. **StatusCancelledWithoutFlag** — A round with `StatusCancelled` but `IsCancelled: false` is still skipped. Both cancellation signals are respected independently.

That last test is the interesting one. It covers a data inconsistency that shouldn't happen in production — but *could* happen if the upstream API sends conflicting signals. The function defends against it rather than assuming consistency. The constitution calls this *Firmitas* — structural integrity — and it extends to handling data you didn't expect.

---

## The Injectable Clock

The calendar service gained a new constructor:

```go
func NewServiceWithClock(repo CalendarRepository, now func() time.Time) *Service {
    return &Service{repo: repo, now: now}
}
```

The `now` field is a `func() time.Time` — a clock function that can be replaced in tests. The integration tests use a `fixedClock` helper:

```go
func fixedClock(t time.Time) func() time.Time {
    return func() time.Time { return t }
}
```

Three integration tests exercise the full service-to-HTTP path with time frozen at specific moments:

- **BeforeMiami** — Clock set to April 19, 2026. Next round is Miami (R2).
- **AfterMiami** — Clock set to May 5, 2026. Next round advances to Spanish GP (R3).
- **SeasonOver** — Clock set to December 31, 2026. `next_round` is 0, `countdown_target_utc` is `null`.

The countdown *transitions* are the actual feature. Not "what's the next race right now" but "when does the next race change." The injectable clock makes these transitions testable without waiting for real time to pass.

---

## The Frontend Countdown

The `useCountdown` hook is a 40-line React hook that ticks every second:

```typescript
export function useCountdown(targetUTC: string | null): CountdownValues | null {
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    if (!targetUTC) return;
    const id = setInterval(() => setNow(Date.now()), 1_000);
    return () => clearInterval(id);
  }, [targetUTC]);

  if (!targetUTC) return null;

  const targetMs = new Date(targetUTC).getTime();
  const diffMs = targetMs - now;

  if (diffMs <= 0) {
    return { days: 0, hours: 0, minutes: 0, seconds: 0, total_ms: 0, expired: true };
  }
  // ... compute days, hours, minutes, seconds from diffMs
}
```

The key design decisions: `setInterval` at 1-second granularity (not `requestAnimationFrame` — this isn't animation, it's a clock). Cleanup on unmount via the returned function. Null return when there's no target (end of season). An `expired` flag when the countdown reaches zero ("Race underway!").

The `NextRaceCard` component consumes this hook and renders an accessible countdown card with `role="region"` and `aria-live="polite"`. Screen readers announce countdown changes without interrupting the user. Five component tests verify rendering, ticking, expiry, and accessibility — all using Vitest's `vi.useFakeTimers()` to control time at the millisecond level.

---

## The WSL Fork Timeout

One of Phase 4's tests exposed a tooling issue that had nothing to do with the code itself.

Vitest's default execution pool on Linux is `forks` — it spawns child processes for test isolation. On WSL2 with a workspace path mounted via OneDrive (`/mnt/c/users/.../OneDrive - Microsoft/...`), the fork overhead becomes catastrophic. The test runner would hang for 60+ seconds and then time out. The same tests finished in under a second on native Linux.

The fix was one line in `vite.config.ts`:

```typescript
test: {
  pool: "threads",
  // ...
}
```

Threads instead of forks. No child process spawning. The WSL-to-Windows filesystem round-trips that killed fork performance become irrelevant because everything runs in the same process. Tests went from timing out to completing in 800ms.

This is the kind of problem that wastes hours if you don't recognize it. The symptom — "tests hang" — has dozens of possible causes. The root cause — "forked processes on a cross-OS mount are slow" — is specific to the WSL + OneDrive development environment. It's not a bug in Vitest, the code, or the tests. It's a platform interaction. Once you know, the fix is trivial.

---

## Building Azure: Seven Modules, Seven Minutes

Phase 2 generated the Bicep infrastructure-as-code. Phase 4 is when it actually ran.

The `az deployment sub create` command against the Bicep templates took 7 minutes and 13 seconds. Seven modules, all provisioned:

| Module | Resource | Key Configuration |
|---|---|---|
| **AKS** | Kubernetes cluster | v1.33, 2× Standard_B2s Azure Linux nodes, Calico network policy, workload identity |
| **ACR** | Container registry | Basic SKU, admin disabled, attached to AKS for pull |
| **Cosmos DB** | Serverless account | `meetings` + `standings` containers, partitioned by `/season` |
| **Key Vault** | Secrets store | RBAC-authorized, soft-delete 7-day retention |
| **Log Analytics** | Monitoring workspace | 30-day retention, PerGB2018 pricing |
| **Workload Identity** | Backend managed identity | AcrPull + Cosmos SQL role + Key Vault Secrets User |
| **CI Identity** | GitHub Actions identity | OIDC federation to `master` branch, AcrPush + AKS Cluster User |

The deployment wasn't smooth. Three issues required fixes before the infrastructure would converge.

---

## The Three Infrastructure Fixes

### AKS Version Cascade

The Bicep template originally specified Kubernetes 1.30. By April 2026, Azure had moved 1.30 to LTS-only status — you can't create new clusters with it on the Standard tier. The fix seemed obvious: bump to 1.31. But 1.31 was also LTS-only. So was 1.32.

```
Code: VMExtensionProvisioningError
Azure Kubernetes Service - Subtier KubernetesOfficial:
supported versions are 1.33.x
```

The final fix: Kubernetes 1.33 with the `KubernetesOfficial` tier. Three version bumps to learn that Azure's version lifecycle moves faster than static Bicep templates.

### Cosmos DB Role Assignment

The identity module originally used Azure RBAC (`Microsoft.Authorization/roleAssignments`) to grant the backend identity access to Cosmos DB. This works for the control plane but not the data plane. Cosmos DB has its own role assignment system.

The fix was switching from:
```bicep
resource roleAssignment 'Microsoft.Authorization/roleAssignments@...' = { ... }
```
to:
```bicep
resource sqlRoleAssignment 'Microsoft.DocumentDB/databaseAccounts/sqlRoleAssignments@...' = { ... }
```

Cosmos DB's built-in `00000000-0000-0000-0000-000000000002` role (Data Contributor) works — but only through the `sqlRoleAssignments` resource type, not through generic Azure RBAC.

### ACR Output Property

The ACR module's output referenced `loginServerHost` — a property that doesn't exist on the `Microsoft.ContainerRegistry/registries` resource. The correct property is `loginServer`. One word difference. Twenty minutes of debugging.

---

## The CI/CD Pipeline: From Five Failures to Full Green

With infrastructure provisioned, the focus shifted to the GitHub Actions pipeline. The first five pushes all failed. Each failure revealed a different gap between the generated CI/CD template and the actual project structure.

### Failure 1: Missing Frontend Lint Script

The CI workflow ran `npm run lint` but the frontend's `package.json` had no `lint` script. ESLint was configured (`eslint.config.js` existed) but the dependencies weren't installed and there was no script entry.

Rather than wrestling with ESLint peer dependency conflicts (eslint-plugin-react requires ESLint ≤9, but the latest ESLint is 10), the fix was pragmatic:

```json
"lint": "tsc --noEmit"
```

TypeScript's `--noEmit` flag runs the full type checker without producing output files. It catches type errors, unused imports, and contract mismatches. For a project that already has TypeScript strict mode enabled, this provides meaningful lint coverage with zero additional dependencies.

### Failure 2: Go Cache Path

`actions/setup-go@v4` looks for `go.sum` at the repository root to build a dependency cache key. But `go.sum` lives in `backend/go.sum` — it's a subdirectory module. The upgrade to `setup-go@v5` with explicit `cache-dependency-path` fixed it:

```yaml
- uses: actions/setup-go@v5
  with:
    go-version: "1.25"
    cache-dependency-path: backend/go.sum
```

### Failure 3: Deprecated golangci-lint Linter

The `.golangci.yml` config enabled `varcheck` — a linter that was removed from golangci-lint in a previous version. The pipeline installed `@latest` and promptly complained about an unknown linter. Removed.

### Failure 4: gofmt Formatting

Six Go files had formatting issues that the local development environment didn't catch (no pre-commit hooks). The CI linter caught them. `gofmt -w` on all six files.

### Failure 5: Misspell and Errcheck

The `misspell` linter flagged `cancelled` as a misspelling of `canceled`. In American English, it is. In Formula 1, which is a global sport with British English conventions, "cancelled" is the standard spelling. Rather than renaming every field, struct, and JSON tag across the codebase:

```yaml
misspell:
  locale: US
  ignore-words:
    - cancelled
```

Domain terminology wins over spell-checker defaults.

The `errcheck` linter caught an unchecked error return from `UpsertMeeting` in an integration test. The in-memory repository can't actually fail, but the linter doesn't know that. Added the check.

### The Green Run

Push number six: all six jobs passed. Lint (backend + frontend) → Test (backend + frontend) → Build (backend + frontend) → Push Docker Images to ACR → Deploy to AKS via Helm. End to end in under four minutes.

Nine GitHub secrets configured via a setup script: `AZURE_CLIENT_ID`, `AZURE_TENANT_ID`, `AZURE_SUBSCRIPTION_ID`, `AKS_RESOURCE_GROUP`, `AKS_CLUSTER_NAME`, `ACR_LOGIN_SERVER`, `COSMOS_ENDPOINT`, `KEYVAULT_URI`, `BACKEND_IDENTITY_CLIENT_ID`. Not one of them is a password. Every credential flows through OIDC federation or managed identity. The constitution mandated this from Day 0.

---

## The Proxy Problem

With four pods running on AKS (two backend, two frontend), the next step was verification. Port-forward the frontend to `localhost:3000`, open a browser, and...

```
Error: Unexpected token '<', "<!doctype "... is not valid JSON
```

The frontend was calling `/api/v1/calendar` — a relative URL — which resolved to `localhost:3000/api/v1/calendar`. But `localhost:3000` is the frontend's nginx server, which has a `try_files` rule that serves `index.html` for any path it doesn't recognize. The API request got back HTML. The JSON parser choked.

The fix: a reverse proxy rule in the frontend's nginx config:

```nginx
location /api/ {
    proxy_pass http://f1-backend.f1-race-intelligence.svc.cluster.local/api/;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
}
```

Inside the Kubernetes cluster, the frontend nginx forwards `/api/*` requests to the backend service using its cluster-internal DNS name. No external load balancer needed. No CORS headers needed. The frontend and backend are in the same namespace, communicating over the cluster network.

This is a pattern worth noting. The frontend's `apiClient.ts` uses `/api/v1` as its base URL — no hostname, no port. In development, that hits the dev server proxy. In Kubernetes, it hits the nginx reverse proxy. In both environments, the frontend code is identical. The infrastructure provides the routing.

---

## What's Running Right Now

```
$ kubectl get pods -n f1-race-intelligence
NAME                           READY   STATUS    RESTARTS   AGE
f1-backend-7bb9b6b6bc-gdkkw    1/1     Running   0          99s
f1-backend-7bb9b6b6bc-wn475    1/1     Running   0          94s
f1-frontend-7857cf6d5d-jzt95   1/1     Running   0          90s
f1-frontend-7857cf6d5d-kvvxw   1/1     Running   0          86s
```

Four pods. Two backend replicas behind a ClusterIP service. Two frontend replicas behind another. Both services wired to Helm-managed deployments with health probes, resource limits, and workload identity annotations.

```
$ curl http://localhost:8080/healthz
{"status":"ok","service":"f1-race-intelligence-backend","timestamp":"2026-04-20T03:20:14Z"}

$ curl http://localhost:3000/api/v1/calendar?year=2026
{"year":2026,"data_as_of_utc":"0001-01-01T00:00:00Z","next_round":0,
 "countdown_target_utc":null,"rounds":[]}
```

The calendar returns empty rounds — the Cosmos DB hasn't been seeded by the pollers yet. But the shape is correct. The HTTP contract matches the OpenAPI spec. The proxy works. The services communicate. The infrastructure carries traffic.

---

## The Numbers

Phase 4 added 7 tasks to the completed total (T025–T031), bringing the project to 31 of 47 tasks done.

| Metric | Value |
|---|---|
| Backend tests | 15 (6 new unit + 3 new integration) |
| Frontend tests | 9 (5 new component) |
| Total tests | 24 |
| Azure resources provisioned | 7 modules |
| CI/CD pipeline stages | 6 (lint → test → build → push → deploy) |
| GitHub secrets | 9 (zero passwords) |
| Infrastructure fixes | 3 (AKS version, Cosmos role, ACR output) |
| CI/CD fixes | 5 (lint script, Go cache, varcheck, gofmt, misspell) |
| Lines typed by a human | 0 |

---

## What I Noticed Today

**The gap between "tests pass" and "deployed" is enormous.** Fourteen tests passing locally proves the logic works. Deploying to real infrastructure proves the *system* works — Docker builds, registry pushes, Kubernetes scheduling, service discovery, nginx routing, managed identity, OIDC federation. Each of those is a potential failure point that no unit test covers.

**Infrastructure-as-code ages faster than application code.** The Bicep templates were generated during Phase 2 and never validated against a real Azure subscription until today. In that gap, three of seven modules needed fixes. Kubernetes versions moved to LTS-only. Resource properties changed names. Role assignment mechanisms that work in documentation don't work in practice. The lesson: deploy early, even if the application isn't ready.

**The CI/CD pipeline is a second codebase.** Five consecutive failures, each revealing a different configuration gap. Missing scripts, wrong cache paths, deprecated linters, formatting drift, domain terminology conflicts. The pipeline has its own dependencies, its own compatibility matrix, and its own failure modes. Treating it as "just a YAML file" is a mistake.

**The proxy pattern is the right default for frontend-backend communication in Kubernetes.** No CORS. No environment-specific API URLs. No build-time configuration baking. The frontend always calls `/api/v1/...` and the infrastructure routes it to the right place. The same frontend container works in development, staging, and production without rebuilding.

---

## What's Next

Phase 5 is **Championship Standings and Cancellation Overrides** — the third and final user story. Drivers and constructors standings, standings API endpoints, a new UI page, and the cancelled-race badge component.

Phase 6 is polish — monitoring dashboards, log schema validation, network boundary tests, and end-to-end quickstart verification.

But today was the day the project became real. Not a spec. Not a plan. Not a test suite. A running application on Azure, deployed by a pipeline, serving HTTP, with zero human-typed code.

The constitution said *Firmitas, Utilitas, Venustas* — structural integrity, practical utility, appropriate beauty. Today we proved *Firmitas*. The structure holds. The pipeline delivers. The infrastructure carries.

Now we fill it with data.

---
