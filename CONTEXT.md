# Project Context — F1 Race Intelligence Dashboard

## Origin and philosophy
This project is built on spec-driven development inspired by Vitruvius's
six architectural principles from De Architectura (25 BCE). The constitution
in .specify/memory/constitution.md maps those six principles directly to
Azure engineering decisions. Read the constitution before any implementation.

## What has been built so far
- GitHub repo initialized and public
- Spec Kit scaffolding complete (.specify/ folder structure)
- Constitution written covering all six Vitruvian principles
- First feature spec written: race calendar and championship standings
- Initial Go backend scaffolding generated (Chi router, Cosmos DB client)
- Initial React frontend scaffolding generated (Vite, Tailwind, routing)
- Day 0 blog post written in docs/blog/

## Roadmap
- Day 1: Review and validate generated code against constitution
- Day 2: Azure infrastructure with Bicep (AKS, Cosmos DB, Key Vault)
- Day 3: OpenF1 API integration and Cosmos DB data ingestion
- Day 4: First working UI — race calendar and Miami GP countdown

## Key decisions made
- Go backend with Chi router — not Python, not Node
- Cosmos DB serverless tier for cost efficiency
- Backend is the only service that calls OpenF1 — never the frontend
- Azure Key Vault via Managed Identity — no service principal passwords
- All K8s manifests in Helm — no raw YAML in production
- OpenF1 free tier (no API key required) for historical data
- Monthly Azure budget alert at $150

## Blog narrative
This project documents a 40-year developer building entirely hands-free
due to a hand injury — using voice and specifications instead of typing.
Every architectural decision connects back to the Vitruvian principles.
Blog posts live in docs/blog/