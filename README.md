# F1 Race Intelligence Dashboard

**Rome wasn't built in a day. This was built in half of one.**

The blog posts say Day 0 through Day 5, but every line of code, every Helm chart, every Bicep module, every CI/CD pipeline, every test, and every blog post was produced on a single Sunday afternoon — after thinking about it on a Saturday. Forty-seven tasks across six phases. Thirty-five tests. A Go backend, a React frontend, Cosmos DB, AKS, Key Vault, nginx ingress, OIDC federation, and a fully green CI/CD pipeline. Deployed and running on Azure.

**Zero lines of code typed by a human.**

GitHub Copilot — powered by Claude Opus 4.6 — wrote every line. The project was driven entirely by [GitHub Spec Kit](https://github.com/github/spec-kit) and a well-defined constitution based on the architectural principles of [Vitruvius](https://en.wikipedia.org/wiki/Vitruvius), the ancient Roman architect who defined the triad of *Firmitas* (structural integrity), *Utilitas* (practical utility), and *Venustas* (appropriate beauty). Those principles governed every decision — from the data model to the deployment topology.

The human provided direction. The machine provided implementation. The constitution provided guardrails.

And on a personal note — I can't wait for the **Italian Grand Prix at Monza on September 4, 2026** (R17). Forza **Lewis Hamilton** in the Ferrari. 🏎️🇮🇹

## What This Repo Contains

- Specification-driven feature planning under `specs/`
- Backend service scaffolding under `backend/`
- Frontend application scaffolding under `frontend/`
- Deployment scaffolding under `deploy/helm/`
- Project documentation and blog posts under `docs/`

## Documentation

This project is being built in public, with architecture decisions and progress captured in the blog.

### Feature 1: Calendar, Standings & Deployment

- [Day 0: From a Roman Architect to a GitHub Repo — Without Writing a Line of Code](docs/blog/day-0-the-constitution.md)
- [Day 1: Laying the Foundation — Phase 2 and the Architecture That Carries Everything](docs/blog/day-1-phase-2-foundation.md)
- [Day 2: The First Thing Anyone Sees — Phase 3 and the Race Calendar MVP](docs/blog/day-2-phase-3-calendar-mvp.md)
- [Day 3: From Localhost to the Cloud — Phase 4 and the First Real Deployment](docs/blog/day-3-phase-4-deployment.md)
- [Day 4: Live Data, Broken Queries, and the Dangers of Round Numbers](docs/blog/day-4-phase-5-live-data.md)
- [Day 5: Forty-Seven Tasks, Zero Lines — The Final Phase](docs/blog/day-5-phase-6-final-polish.md)

### Feature 2: Race Session Results

- [Day 6: Clicking Into the Race — Session Results & Round Detail](docs/blog/day-6-race-session-results.md)
- [Day 7: Race Results, Rate Limits, and the Branch You Forgot You Were On](docs/blog/day-7-phase-4-and-branch-confusion.md)
- [Day 10: The Rate Limit Cascade — Three PRs to Make a Race Page Render](docs/blog/day-10-rate-limit-cascade.md)

### Security & Operations

- [Day 8: The Security Alert I Got at 5 AM — And What I Did About It](docs/blog/day-8-ops-security-lockdown.md)
- [Day 9: The Struggle Bus to a Private Cosmos DB](docs/blog/day-9-cosmos-private-endpoint.md)

### Feature 4: Design System & Brand Identity

- [Day 11: From Generic Data Table to Race Car — A Design System in an Afternoon](docs/blog/day-11-design-system.md)

### Bug Fixes & Hardening

- [Day 12: The Bug That Said "Completed" When the Race Hadn't Started](docs/blog/day-12-status-badge-bug.md)
- [Day 13: An Empty String Is a Wildcard — Closing the AKS Allowlist Hole](docs/blog/day-13-allowlist-guard.md)
- [Day 14: Three Bugs, One Cause — When Stale Cached State Becomes a UX Smell](docs/blog/day-14-design-polish.md)

## Architecture Direction

- Go backend with Chi router
- React frontend consuming backend APIs only
- Cosmos DB serverless for cached OpenF1 data
- AKS for runtime orchestration
- Azure Key Vault with Managed Identity for secrets
- Helm charts for Kubernetes delivery
- GitHub Actions for CI/CD

## Current Status

**Feature 1 — Calendar & Standings:** Complete. All 47 tasks across 6 phases done.

**Feature 2 — Race Session Results:** Complete. All phases shipped. Race, qualifying, and practice components rendering with correct data on the live site. Backend ingests `/v1/session_result` from OpenF1, transforms polymorphic duration/gap fields by session type, derives fastest-lap holder from `/v1/laps`, and finalizes session documents in Cosmos 2 hours after each session ends so the poller skips them on subsequent cycles. Steady-state OpenF1 traffic is 0 requests/cycle for finalized weekends.

**Security Lockdown (April 26, 2026):** Cosmos DB public access disabled; all reads/writes now flow through an Azure Private Endpoint in a dedicated subnet. CI managed identity restricted to `Contributor` only — role grants extracted to a manual Owner-only script. Live URL migrated from `*.nip.io` to Azure FQDN (`f1raceintel.westus3.cloudapp.azure.com`). Subnet NSGs explicit in Bicep to prevent Azure Policy from creating empty defaults that drop ingress traffic.

**Feature 4 — Design System & Brand Identity:** Complete. All 37 tasks across 6 phases done. Tailwind v4 `@theme` tokens, shadcn/ui primitives, self-hosted Inter and JetBrains Mono fonts. Near-black F1 theme with team-color accents on every standings/results row. New atomic components (`DriverCard`, `LapTimeDisplay`, `TireCompound`, `RaceCountdown`, `StandingsTable`). Same data, same routes — now it looks like motorsport.

**Bug Fix (April 27, 2026):** Future round sessions were displaying a green "Completed" badge instead of "Upcoming." Two stacked root causes: the ingest transform hardcoded `Status: "completed"` for every session it wrote to Cosmos, and the poller's future-skip guard silently fell through when OpenF1 returned a null `date_end`. Fixed by deriving status at read time in the rounds API service (immediate user-visible correction, no Cosmos backfill needed) and stopping the hardcoded value at write time (durable prevention). Frontend now maps the wire-level `in_progress` enum to a friendly "Live" label.

**CI Hardening (April 28, 2026):** Closed an AKS API-server allowlist hole surfaced on Day 12. The deploy cleanup step passed `${{ secrets.ADMIN_IP_RANGES }}` directly to `az aks update --api-server-authorized-ip-ranges`, and Azure interprets an empty value as "remove all restrictions" — one missing secret away from publishing the cluster's control plane to the Internet. Fix layered three guards: a preflight check at the top of each deploy job in `ci-cd.yml` and `infra-deploy.yml` that fails fast before any cluster mutation, an emptiness check on the cleanup `az aks update` (refuses with `exit 1` rather than silently opening the cluster), and a `gh secret set --body ""` refusal in `sync-ip.sh`. Both workflows reference the secret via `env:` rather than inline template substitution. PR [#25](https://github.com/karlkuhnhausen/f1-race-intelligence/pull/25).

**Design Polish (April 29, 2026):** Three bugs with one shared root cause — derived state cached as fact. (1) Past races (Australian, Chinese, Japanese GPs) kept showing "Scheduled" because meeting status was written once at ingest and never updated; fixed by deriving status at read time in the calendar service from start/end dates plus the wall clock, mirroring the Day 12 session-level pattern. (2) Round detail page listed sessions chronologically even after the round was over, burying the race result at the bottom; fixed by sorting by `date_start_utc` descending when every session is `completed`, naturally handling sprint weekends. (3) Session results tables were squished and had drifted across three near-duplicate components (`RaceResults`, `QualifyingResults`, `PracticeResults`); consolidated onto a single `SessionResultsTable` with `table-fixed` + `<colgroup>` widths, right-aligned `font-mono tabular-nums` for numeric/time columns, team color swatches via `getTeamColor()`, podium border accents on race sessions, and an absorbed "Not Classified" divider for DNFs. The three legacy components and their test files were deleted. PR [#27](https://github.com/karlkuhnhausen/f1-race-intelligence/pull/27).

- **Frontend**: http://f1raceintel.westus3.cloudapp.azure.com/
- **API**: http://f1raceintel.westus3.cloudapp.azure.com/api/v1/calendar?year=2026
- **Round Detail**: http://f1raceintel.westus3.cloudapp.azure.com/rounds/3?year=2026
- **Pipeline**: Fully green — lint → test → build → push → deploy
- **Tests**: 70 passing (22 backend + 48 frontend)

## Why Spec-Driven Development

The project treats specifications as the source of truth. Architecture rules are defined first, implementation plans are derived from them, and code follows those decisions instead of inventing them ad hoc.