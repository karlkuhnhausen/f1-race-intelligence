<!--
Sync Impact Report
- Version change: template -> 1.0.0
- Modified principles:
	- [PRINCIPLE_1_NAME] -> I. Prescribed Platform Stack
	- [PRINCIPLE_2_NAME] -> II. Enforced Three-Tier Boundaries
	- [PRINCIPLE_3_NAME] -> III. OpenF1 Data Residency and Caching
	- [PRINCIPLE_4_NAME] -> IV. Security and Secrets Baseline
	- [PRINCIPLE_5_NAME] -> V. Delivery, Operations, and Dependency Discipline
- Added sections:
	- Architecture and Platform Constraints
	- Delivery and Runtime Standards
- Removed sections:
	- None
- Templates requiring updates:
	- ✅ .specify/templates/plan-template.md
	- ✅ .specify/templates/spec-template.md
	- ✅ .specify/templates/tasks-template.md
	- ✅ .specify/templates/commands/*.md (not present in repository; no changes required)
- Follow-up TODOs:
	- None
-->

# F1 Race Intelligence Constitution

## Core Principles

### I. Prescribed Platform Stack
All production features MUST use a Go backend, React frontend, and Azure Cosmos DB running on Azure Kubernetes Service (AKS). Alternative runtimes, databases, or orchestration platforms MUST NOT be introduced for production paths unless approved by a constitution amendment. This preserves operational consistency and predictable ownership.

### II. Enforced Three-Tier Boundaries
The system MUST follow a strict three-tier architecture: UI tier, API/service tier, and data tier. The UI tier MUST NOT call external APIs directly; all external integration MUST flow through the backend API/service tier. This keeps security controls, caching, and observability centralized.

### III. OpenF1 Data Residency and Caching
The backend MUST cache all OpenF1 data in Cosmos DB before serving consumer requests. Backend endpoints MUST prefer Cosmos DB as the system of record for OpenF1-derived responses and MUST define freshness and refresh behavior explicitly in specs. Direct pass-through from OpenF1 to clients is prohibited except for documented break-glass operations.

### IV. Security and Secrets Baseline
All secrets MUST be stored in Azure Key Vault and retrieved through Managed Identity. Static secrets in source code, manifests, CI variables, or local config files are forbidden. HTTPS MUST be enforced at NGINX ingress, and Azure Firewall MUST control outbound egress with explicit allow-lists. Any exception requires documented risk acceptance and expiration.

### V. Delivery, Operations, and Dependency Discipline
Kubernetes manifests MUST be delivered as Helm charts. CI/CD MUST run in GitHub Actions with gates in this order: lint -> test -> build -> push -> deploy. Services MUST emit structured JSON logs to Azure Monitor. Dependencies MUST be minimized; every added package MUST include a written justification and ownership. Specs are the source of truth and code is a derived artifact: implementation, plans, and tasks MUST trace back to approved specifications.

## Architecture and Platform Constraints

- Backend services MUST be implemented in Go and expose versioned APIs consumed by React clients.
- React applications MUST call only internal backend APIs; browser-side direct calls to OpenF1 or other third-party APIs are disallowed.
- Cosmos DB MUST use the serverless tier for production and non-production workloads unless a documented capacity exception is approved.
- Data access paths MUST preserve tenant and environment isolation.
- Helm charts MUST define ingress, service, deployment/stateful resources, secret references, health probes, and network policies.
- All secret material MUST be referenced from Key Vault-backed mechanisms with Managed Identity; plaintext secrets in repos are prohibited.

## Delivery and Runtime Standards

- Every spec MUST include architecture boundary assertions, data caching behavior, security controls, and operational requirements aligned to this constitution.
- Every plan MUST include a constitution check that verifies stack compliance, tier boundaries, caching strategy, Helm delivery, CI/CD stage order, logging, and dependency justification.
- Every tasks list MUST include explicit tasks for Helm artifacts, GitHub Actions pipeline sequencing, Key Vault/Managed Identity wiring, ingress TLS and egress firewall policy, Cosmos serverless configuration, OpenF1 cache persistence, and structured JSON logging.
- Pull requests MUST fail review if they violate mandatory principles, omit dependency justifications, or introduce implementation behavior not represented in specification artifacts.

## Governance

This constitution is authoritative for architecture, delivery, and security policy in this repository.

Amendment procedure:
1. Propose changes through a documented constitution update.
2. Include rationale, impact analysis, and template synchronization updates.
3. Obtain explicit approval from repository maintainers before merge.

Versioning policy:
- MAJOR: incompatible governance changes or principle removals/redefinitions.
- MINOR: new principle/section or materially expanded mandatory guidance.
- PATCH: wording clarifications and non-semantic refinements.

Compliance review expectations:
- Constitution compliance MUST be checked during spec, plan, tasks, and pull request review.
- Violations MUST be corrected before deployment unless an explicit, time-bounded exception is documented and approved.
- Runtime guidance in .github/copilot-instructions.md and Spec Kit templates MUST remain aligned with this constitution.

**Version**: 1.0.0 | **Ratified**: 2026-04-19 | **Last Amended**: 2026-04-19
