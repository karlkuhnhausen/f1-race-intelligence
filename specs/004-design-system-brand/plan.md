# Implementation Plan: Design System and Brand Identity

**Branch**: `004-design-system-brand` | **Date**: 2026-04-20 | **Spec**: `/specs/004-design-system-brand/spec.md`
**Input**: Feature specification from `/specs/004-design-system-brand/spec.md`

## Summary

Install shadcn/ui and Tailwind CSS to establish a cohesive dark racing theme design system for the F1 Race Intelligence Dashboard. Define design tokens (colors, typography, team colors) in `tailwind.config.ts` with CSS custom properties. Build five reusable UI components — DriverCard, LapTimeDisplay, TireCompound, RaceCountdown, and StandingsTable — and migrate existing pages (calendar, standings, round detail) to the new design system. This is a frontend-only feature; no backend changes required.

## Technical Context

**Language/Version**: TypeScript with React 18+ (frontend only)
**Primary Dependencies**: shadcn/ui (Radix UI primitives), Tailwind CSS v4, @fontsource/inter, @fontsource/jetbrains-mono, class-variance-authority, clsx, tailwind-merge
**Storage**: N/A — no storage changes
**Testing**: Vitest + @testing-library/react (existing frontend test stack)
**Target Platform**: Linux containers on Azure Kubernetes Service (existing frontend workload)
**Project Type**: Web application — frontend design system layer
**Performance Goals**: No layout shift on font load (FOUT prevention via self-hosted fonts); no measurable performance regression on existing pages
**Constraints**: No external CDN calls for fonts (CA-002); all styling via Tailwind utilities; minimize custom CSS; desktop-first responsive
**Scale/Scope**: 5 new components, 9 design tokens, 6 team colors, 3 font families, migration of 3 existing pages

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- Stack gate: Uses Go backend, React frontend, Cosmos DB, and AKS for production workloads.
- Architecture gate: UI communicates only with backend APIs; external API calls are backend-only.
- Data gate: OpenF1 integration includes Cosmos DB caching strategy, data freshness policy, and backfill behavior.
- Security gate: Secrets flow through Azure Key Vault with Managed Identity; no plaintext secret handling.
- Network gate: HTTPS termination at NGINX ingress and Azure Firewall egress policy are defined.
- Delivery gate: Kubernetes delivery uses Helm charts; GitHub Actions stages are lint -> test -> build -> push -> deploy.
- Observability gate: Structured JSON logging and Azure Monitor ingestion are specified.
- Dependency gate: Every added dependency includes explicit justification and maintenance owner.
- Spec authority gate: Implementation plan traces all major work items back to specification requirements.

Constitution gates status (initial):
- Stack gate: **PASS** — Frontend-only feature using React; no new runtimes, databases, or orchestration changes
- Architecture gate: **PASS** — No new API calls; components consume data already provided by existing backend endpoints
- Data gate: **N/A** — No new data integration or caching; existing OpenF1 pipeline unchanged
- Security gate: **N/A** — No new secrets; self-hosted fonts bundled in the frontend build artifact
- Network gate: **N/A** — No ingress or firewall changes; fonts served from same origin
- Delivery gate: **PASS** — Existing Helm charts and CI/CD pipeline; frontend rebuilds via same Dockerfile and image push flow
- Observability gate: **N/A** — No backend changes; no new logging requirements
- Dependency gate: **PASS** — New dependencies justified below:
  - `tailwindcss` + `@tailwindcss/vite`: utility-first CSS framework for token-based theming (FR-002, FR-003)
  - `shadcn/ui` (via CLI, installs Radix UI primitives): accessible component primitives (FR-001)
  - `class-variance-authority` + `clsx` + `tailwind-merge`: standard shadcn/ui utility stack for variant composition
  - `@fontsource/inter`: self-hosted display/body font (FR-004, FR-005, CA-002)
  - `@fontsource/jetbrains-mono`: self-hosted monospace font for data display (FR-006, CA-002)
- Spec authority gate: **PASS** — All work items trace to FR-001–FR-015 and SC-001–SC-007

Constitution gates status (post-design):
- All gates remain **PASS** or **N/A** as assessed above. No design decisions introduced new gate concerns.

## Project Structure

### Documentation (this feature)

```text
specs/004-design-system-brand/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (UI component prop contracts)
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```text
frontend/
├── index.html                         # Updated: add font preload hints
├── tailwind.config.ts                 # NEW: design tokens, team colors, font families
├── postcss.config.js                  # NEW: Tailwind PostCSS plugin (if needed by shadcn init)
├── components.json                    # NEW: shadcn/ui configuration
├── src/
│   ├── main.tsx                       # Updated: import global CSS
│   ├── App.tsx                        # Updated: apply theme classes to layout shell
│   ├── globals.css                    # NEW: Tailwind directives, CSS custom properties, font imports
│   ├── lib/
│   │   └── utils.ts                   # NEW: cn() utility (shadcn convention)
│   ├── components/
│   │   └── ui/                        # NEW: shadcn/ui generated primitives (table, card, badge)
│   ├── features/
│   │   ├── calendar/
│   │   │   ├── CalendarPage.tsx       # Updated: migrate to Tailwind classes
│   │   │   └── NextRaceCard.tsx       # Updated: migrate to design system
│   │   ├── rounds/
│   │   │   ├── RoundDetailPage.tsx    # Updated: migrate to design system
│   │   │   └── SessionResultsTable.tsx # Updated: migrate to design system
│   │   ├── standings/
│   │   │   └── StandingsPage.tsx      # Updated: migrate to design system
│   │   └── design-system/            # NEW: F1-specific design system components
│   │       ├── DriverCard.tsx
│   │       ├── LapTimeDisplay.tsx
│   │       ├── TireCompound.tsx
│   │       ├── RaceCountdown.tsx
│   │       ├── StandingsTable.tsx
│   │       └── teamColors.ts         # NEW: team color mapping + fallback logic
│   └── services/
│       └── apiClient.ts              # Existing (unchanged)
├── tests/
│   ├── design-system/                # NEW: component unit tests
│   │   ├── DriverCard.test.tsx
│   │   ├── LapTimeDisplay.test.tsx
│   │   ├── TireCompound.test.tsx
│   │   ├── RaceCountdown.test.tsx
│   │   └── StandingsTable.test.tsx
│   ├── calendar/                     # Existing (updated if migration breaks tests)
│   ├── rounds/                       # Existing (updated if migration breaks tests)
│   └── standings/                    # Existing (updated if migration breaks tests)
└── package.json                      # Updated: new dependencies

deploy/
└── helm/
    └── frontend/                     # Existing (no changes expected — image rebuild handles new CSS/fonts)
```

**Structure Decision**: Extends the existing frontend structure. New design system components live in `src/features/design-system/` following the established `features/{domain}/` convention. shadcn/ui primitives go in `src/components/ui/` per shadcn convention. No backend or infrastructure changes.

## Complexity Tracking

No constitution violations. No complexity justifications needed.
