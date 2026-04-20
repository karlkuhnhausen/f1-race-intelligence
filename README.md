# F1 Race Intelligence Dashboard

F1 Race Intelligence Dashboard is a spec-driven project for building a Formula 1 analytics application on Azure Kubernetes Service with a Go backend, React frontend, and Cosmos DB.

## What This Repo Contains

- Specification-driven feature planning under `specs/`
- Backend service scaffolding under `backend/`
- Frontend application scaffolding under `frontend/`
- Deployment scaffolding under `deploy/helm/`
- Project documentation and blog posts under `docs/`

## Documentation

This project is being built in public, with architecture decisions and progress captured in the blog.

- [Day 0: From a Roman Architect to a GitHub Repo — Without Writing a Line of Code](docs/blog/day-0-the-constitution.md)
- [Day 1: Laying the Foundation — Phase 2 and the Architecture That Carries Everything](docs/blog/day-1-phase-2-foundation.md)
- [Day 2: The First Thing Anyone Sees — Phase 3 and the Race Calendar MVP](docs/blog/day-2-phase-3-calendar-mvp.md)
- [Day 3: From Localhost to the Cloud — Phase 4 and the First Real Deployment](docs/blog/day-3-phase-4-deployment.md)

## Architecture Direction

- Go backend with Chi router
- React frontend consuming backend APIs only
- Cosmos DB serverless for cached OpenF1 data
- AKS for runtime orchestration
- Azure Key Vault with Managed Identity for secrets
- Helm charts for Kubernetes delivery
- GitHub Actions for CI/CD

## Current Status

The repository currently contains the initial constitution, feature specification, implementation plan, task breakdown, and Phase 1 project scaffolding for the Race Calendar and Championship Standings feature.

## Why Spec-Driven Development

The project treats specifications as the source of truth. Architecture rules are defined first, implementation plans are derived from them, and code follows those decisions instead of inventing them ad hoc.