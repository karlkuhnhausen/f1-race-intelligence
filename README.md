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

### Security & Operations

- [Day 8: The Security Alert I Got at 5 AM — And What I Did About It](docs/blog/day-8-ops-security-lockdown.md)
- [Day 9: The Struggle Bus to a Private Cosmos DB](docs/blog/day-9-cosmos-private-endpoint.md)

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

**Feature 2 — Race Session Results:** In progress. Phases 1–4 complete (21/31 tasks). Backend data pipeline, Cosmos storage, rounds API, frontend routing, round detail page, and dedicated race results component deployed. Rate limiting added to session poller. Phases 5–7 (qualifying/practice components and polish) remaining.

**Security Lockdown (April 26, 2026):** Cosmos DB public access disabled; all reads/writes now flow through an Azure Private Endpoint in a dedicated subnet. CI managed identity restricted to `Contributor` only — role grants extracted to a manual Owner-only script. Live URL migrated from `*.nip.io` to Azure FQDN (`f1raceintel.westus3.cloudapp.azure.com`). Subnet NSGs explicit in Bicep to prevent Azure Policy from creating empty defaults that drop ingress traffic.

- **Frontend**: http://f1raceintel.westus3.cloudapp.azure.com/
- **API**: http://f1raceintel.westus3.cloudapp.azure.com/api/v1/calendar?year=2026
- **Round Detail**: http://f1raceintel.westus3.cloudapp.azure.com/api/v1/rounds/3?year=2026
- **Pipeline**: Fully green — lint → test → build → push → deploy
- **Tests**: 51 passing (22 backend + 29 frontend)

## Why Spec-Driven Development

The project treats specifications as the source of truth. Architecture rules are defined first, implementation plans are derived from them, and code follows those decisions instead of inventing them ad hoc.