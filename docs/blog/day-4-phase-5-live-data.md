# Day 4: Live Data, Broken Queries, and the Dangers of Round Numbers

*Posted April 19, 2026 · Karl Kuhnhausen*

---

Phase 5 was supposed to be straightforward. Add championship standings endpoints, mark two cancelled races, build a standings UI, deploy. The feature code took less than an hour. But the three bugs that followed — a disconnected poller, a silent type mismatch in Cosmos DB, and a round-numbering assumption that cancelled the wrong Grand Prix — are the kind of production lessons that no amount of testing in isolation can catch.

This is the story of the day the dashboard got real data — and immediately showed that data is harder than code.

---

## The Feature: Championship Standings and Cancellation Overrides

Phase 5 implements User Story 3: drivers and constructors championship standings, plus cancellation overrides for the Bahrain and Saudi Arabian Grands Prix.

The backend changes follow the exact pattern established in Phase 3:

```
backend/internal/api/standings/
├── dto.go      — JSON response envelopes
├── service.go  — repo wrapper, data shaping
└── handler.go  — thin HTTP layer, parse year, delegate, encode
```

The standings service queries the `StandingsRepository` interface (already defined in Phase 2 and implemented by the Cosmos DB client), aggregates the `data_as_of_utc` watermark across rows, and returns typed DTOs:

```go
func (s *Service) GetDrivers(ctx context.Context, season int) (*DriversStandingsResponse, error) {
    rows, err := s.repo.GetDriverStandings(ctx, season)
    if err != nil {
        return nil, err
    }
    var latestDataAsOf time.Time
    dtos := make([]DriverStandingDTO, 0, len(rows))
    for _, r := range rows {
        dtos = append(dtos, DriverStandingDTO{
            Position:   r.Position,
            DriverName: r.DriverName,
            TeamName:   r.TeamName,
            Points:     r.Points,
            Wins:       r.Wins,
        })
        if r.DataAsOfUTC.After(latestDataAsOf) {
            latestDataAsOf = r.DataAsOfUTC
        }
    }
    return &DriversStandingsResponse{Year: season, DataAsOfUTC: latestDataAsOf, Rows: dtos}, nil
}
```

The router gained a `standingsRepo` parameter, and two new routes were registered:

```go
r.Get("/standings/drivers", standingsHandler.GetDrivers)
r.Get("/standings/constructors", standingsHandler.GetConstructors)
```

The frontend got navigation tabs (Calendar / Standings) in `App.tsx`, a `StandingsPage` with drivers/constructors sub-tabs, and a `CancelledRaceBadge` component for marking cancelled races in the calendar table.

All 13 new tests passed on the first run — 6 backend contract tests, 5 integration tests for cancellation overrides, and 4 frontend component tests for the standings page. Combined with the existing suite: **27 tests total across the project**.

The feature code was clean. The problems started at deployment.

---

## Bug 1: The Pollers That Were Never Started

The OpenF1 calendar poller and Hyprace standings poller had been implemented in Phase 2. The `OpenF1Poller.Start()` method was written. The `HypraceClient.Start()` method was written. Both had structured logging. Both had error handling. Both had been sitting in their packages, unreferenced by `main.go`, for three entire phases.

The symptom: the frontend showed "Data as of: 12/31/1, 4:07:02 PM" — the zero-value of `time.Time`, formatted to a locale date. The calendar table was empty.

The backend pod logs told the story in one line:

```
2026/04/20 03:51:43 backend starting on :8080
```

No poller started. No data fetched. No upserts attempted. The API was serving correctly — it was returning exactly what was in the database, which was nothing.

The fix was two `go` statements in `main.go`:

```go
if calendarRepo != nil {
    season := time.Now().Year()
    ctx := context.Background()

    calendarPoller := ingest.NewOpenF1Poller(calendarRepo, logger)
    go calendarPoller.Start(ctx, season)

    standingsPoller := standings.NewHypraceClient(standingsRepo, logger)
    go standingsPoller.Start(ctx, season)
}
```

After redeployment, the logs changed:

```
{"level":"INFO","msg":"openf1 poller starting","season":2026,"interval":300000000000}
{"level":"INFO","msg":"openf1 poll complete","season":2026,"meetings":26}
```

Twenty-six meetings fetched from OpenF1 and upserted into Cosmos DB. But the API still returned empty.

---

## Bug 2: The Type Mismatch That Cosmos Forgave But Never Matched

The OpenF1 poller was logging successful upserts. No errors. Twenty-six meetings written. But `GET /api/v1/calendar?year=2026` returned `{"rounds":[]}`.

The upsert code was correct:

```go
pk := azcosmos.NewPartitionKeyNumber(float64(m.Season))
_, err = c.meetings.UpsertItem(ctx, pk, data, nil)
```

Partition key: numeric `2026`. Document JSON: `"season": 2026` (integer). The write path was consistent.

The query was not:

```go
query := "SELECT * FROM c WHERE c.season = @season ORDER BY c.round"
queryOpts := &azcosmos.QueryOptions{
    QueryParameters: []azcosmos.QueryParameter{
        {Name: "@season", Value: strconv.Itoa(season)},
    },
}
```

`strconv.Itoa(season)` produces the **string** `"2026"`. The document field `c.season` is the **integer** `2026`. In Cosmos DB's SQL dialect, `2026 = "2026"` evaluates to `false` — there is no implicit type coercion. Every query returned zero results. Every upsert succeeded. The data was there, invisible.

The fix was a one-word change, applied in three places:

```go
{Name: "@season", Value: season},
```

Pass the `int` directly. The Azure SDK handles the JSON serialization. After redeployment, the calendar API returned all 26 rounds with real race names, dates, and circuit data.

This bug is pernicious because the write path and read path used different type representations for the same field. The upsert serialized the struct (integer), but the query parameter used `strconv.Itoa` (string). Both were "correct" in isolation. Only the combination failed — and Cosmos DB returned empty results rather than an error.

---

## Bug 3: The Round Numbers That Cancelled the Wrong Races

With data flowing, the cancellation overrides appeared to work — R4 and R5 were marked cancelled. But the user noticed something wrong: R4 was the Chinese Grand Prix and R5 was the Japanese Grand Prix. The actual cancelled races were the Bahrain GP and Saudi Arabian GP.

The root cause: OpenF1 includes pre-season testing events as "meetings". The 2026 data starts with:

| Round | Event |
|-------|-------|
| 1 | Pre-Season Testing (Sakhir) |
| 2 | Pre-Season Testing (Sakhir) |
| 3 | Australian Grand Prix |
| 4 | Chinese Grand Prix |
| 5 | Japanese Grand Prix |
| 6 | **Bahrain Grand Prix** |
| 7 | **Saudi Arabian Grand Prix** |

The original cancellation override was round-based:

```go
func CancellationOverrides() []CancellationOverride {
    return []CancellationOverride{
        {Season: 2026, Round: 4, ...},
        {Season: 2026, Round: 5, ...},
    }
}
```

Round 4 in OpenF1 is not the same as "the fourth Grand Prix". The two pre-season testing events shifted everything by two positions. The spec said "R4 Bahrain and R5 Saudi Arabia" because the spec was written against FIA round numbers, which don't count testing. OpenF1 counts everything.

The fix was to match by race name instead of round number:

```go
func CancellationOverrides() []CancellationOverride {
    return []CancellationOverride{
        {Season: 2026, RaceName: "Bahrain Grand Prix", Label: "Cancelled", Reason: "Race removed from 2026 calendar"},
        {Season: 2026, RaceName: "Saudi Arabian Grand Prix", Label: "Cancelled", Reason: "Race removed from 2026 calendar"},
    }
}

func IsCancelled(season int, raceName string) (CancellationOverride, bool) {
    for _, o := range CancellationOverrides() {
        if o.Season == season && strings.EqualFold(o.RaceName, raceName) {
            return o, true
        }
    }
    return CancellationOverride{}, false
}
```

Case-insensitive comparison via `strings.EqualFold`. The calendar service now passes the race name from the stored meeting rather than the round number:

```go
if override, ok := domain.IsCancelled(m.Season, m.RaceName); ok {
    m.IsCancelled = true
    m.Status = string(domain.StatusCancelled)
    m.CancelledLabel = override.Label
    m.CancelledReason = override.Reason
}
```

After redeployment: R6 Bahrain Grand Prix and R7 Saudi Arabian Grand Prix correctly show as cancelled. The Chinese and Japanese Grands Prix are back to scheduled.

---

## The Nginx Ingress Controller

Phase 5 also eliminated the need for `kubectl port-forward` during testing. An nginx ingress controller was installed via Helm:

```bash
helm install ingress-nginx ingress-nginx/ingress-nginx \
  --namespace ingress-nginx --create-namespace \
  --set controller.service.annotations."service\.beta\.kubernetes\.io/azure-load-balancer-health-probe-request-path"=/healthz
```

Azure provisioned a public Load Balancer with IP `20.171.233.61`. Using [nip.io](https://nip.io) wildcard DNS, both services are accessible without any DNS configuration:

- **Frontend**: `http://f1.20.171.233.61.nip.io/`
- **Backend API**: `http://api-f1.20.171.233.61.nip.io/api/v1/calendar?year=2026`

The Helm ingress templates were updated to conditionally render TLS blocks (disabled for nip.io) and force-ssl-redirect was set to `false`:

```yaml
annotations:
  nginx.ingress.kubernetes.io/force-ssl-redirect: "false"
spec:
  ingressClassName: {{ .Values.ingress.className }}
  {{- if .Values.ingress.tls }}
  tls:
    - hosts:
        - {{ .Values.ingress.host }}
      secretName: {{ .Values.ingress.tlsSecretName }}
  {{- end }}
```

Two ingress resources route traffic: `api-f1.20.171.233.61.nip.io` → `f1-backend:80` and `f1.20.171.233.61.nip.io` → `f1-frontend:80`. The frontend's nginx already proxies `/api/` to the backend's cluster-internal DNS, so the same container works behind both the ingress and via port-forward.

---

## The Numbers

| Metric | Value |
|--------|-------|
| Phase 5 feature commit | 825 lines changed across 20 files |
| Bug fixes after deployment | 3 (pollers, Cosmos query, round numbering) |
| Total tests | 27 (13 backend + 13 frontend + 1 case-insensitive) |
| Pipeline runs today | All green after gofmt fix |
| External endpoints | 2 (frontend + API via ingress) |
| OpenF1 meetings ingested | 26 |
| Lines typed by a human | 0 |

---

## What I Noticed Today

**The pollers were the integration equivalent of a dead letter queue.** Code that compiles, passes lint, and has the right method signatures — but is never called. No test caught this because the tests use mock repositories injected at the handler level. The poller-to-main wiring is an integration seam that only manifests at runtime. In a production system, you'd want a startup health check that verifies the pollers have completed at least one cycle.

**Cosmos DB's type strictness is silent.** The query didn't fail. It returned zero results. If you're used to SQL databases that coerce types in WHERE clauses, this is a trap. The Azure SDK's `QueryParameter.Value` accepts `interface{}` — it will happily serialize a string, an int, a float, or a struct. There's no compile-time protection against passing the wrong type. The only defense is testing against a real database, which we deliberately avoid in the fast test suite.

**Round numbers are not identity.** The spec said "R4 and R5". The FIA says those are the Bahrain and Saudi Arabian Grands Prix. OpenF1 says the fourth meeting of 2026 is the Chinese Grand Prix because it counts testing sessions. Round numbers are an ordinal assigned by a specific source, not a stable identifier. Race names are closer to stable — "Bahrain Grand Prix" doesn't change based on what comes before it. This is a general lesson: if you're building overrides or rules that reference external data, match on the most semantically stable attribute, not the most convenient numeric index.

**Nip.io is underrated for pre-DNS testing.** It resolves `anything.1.2.3.4.nip.io` to `1.2.3.4`. No DNS records, no registrars, no propagation delays. Combined with nginx ingress, it gives you host-based routing on a bare IP within seconds of the Load Balancer provisioning. For internal demos and development, it eliminates an entire class of "it works on my machine" problems.

---

## What's Next

Phase 6 is the final phase: **Polish and Cross-Cutting Concerns**. Monitoring dashboards, structured log schema validation, network boundary tests ensuring the frontend never calls upstream APIs directly, Helm value verification, CI/CD deployment guard validation, and a full quickstart end-to-end run.

After Phase 6, every task in the 47-task plan will be complete. The project will have gone from a Vitruvian constitution to a running, deployed, monitored application — with zero lines of human-typed code.

---

*Previous: [Day 3: From Localhost to the Cloud — Phase 4 and the First Real Deployment](day-3-phase-4-deployment.md)*
