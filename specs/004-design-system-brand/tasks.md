# Tasks: Design System and Brand Identity

**Input**: Design documents from `/specs/004-design-system-brand/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/component-props.md, quickstart.md

**Tests**: Included — FR-014 requires unit tests for all new components.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

**Constitution Alignment**: Frontend-only feature. No backend, Cosmos DB, Key Vault, ingress, or Helm changes. Existing CI pipeline (lint → test → build → push → deploy) and Dockerfile handle the rebuilt frontend image. All new dependencies justified per CA-009.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Frontend**: `frontend/src/`, `frontend/tests/`
- **Design system components**: `frontend/src/features/design-system/`
- **shadcn/ui primitives**: `frontend/src/components/ui/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Install and configure Tailwind CSS v4, shadcn/ui, and self-hosted fonts

- [x] T001 Install `tailwindcss` and `@tailwindcss/vite` as dependencies in `frontend/package.json`
- [x] T002 Add `@tailwindcss/vite` plugin to `frontend/vite.config.ts` alongside existing config
- [x] T003 Initialize shadcn/ui via `npx shadcn@latest init` with New York style, Slate base color, and CSS variables enabled — creates `frontend/components.json` and `frontend/src/lib/utils.ts`
- [x] T004 [P] Add path alias `"@/*": ["./src/*"]` to `frontend/tsconfig.json` compilerOptions.paths (shadcn convention)
- [x] T005 [P] Install `@fontsource/inter` (weights 400, 700) and `@fontsource-variable/jetbrains-mono` in `frontend/package.json`
- [x] T006 [P] Install shadcn/ui primitives: `npx shadcn@latest add table card badge` — generates files in `frontend/src/components/ui/`
- [x] T007 [P] Install `class-variance-authority`, `clsx`, and `tailwind-merge` in `frontend/package.json` (if not already added by shadcn init)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Define design tokens, global styles, and shared utilities that ALL user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T008 Create `frontend/src/globals.css` with Tailwind v4 `@import "tailwindcss"` directive, `@theme` block defining all 9 color tokens (background=#0f0f13, surface=#1a1a23, border=#2a2a38, accent-red=#e8002d, accent-cyan=#00d2ff, foreground=#ffffff, muted-foreground=#8888aa, positive=#00c853, negative=#ff1744), 6 team color CSS custom properties (--team-mercedes=#00d2be, --team-redbull=#3671c6, --team-ferrari=#e8002d, --team-mclaren=#ff8000, --team-aston=#358c75, --team-alpine=#ff87bc), and font-family declarations (display: Inter 700, body: Inter 400, mono: JetBrains Mono 500)
- [x] T009 Update `frontend/src/main.tsx` to import `@fontsource/inter/400.css`, `@fontsource/inter/700.css`, `@fontsource-variable/jetbrains-mono`, and `./globals.css`
- [x] T010 [P] Update `frontend/index.html` to add `class="dark"` on `<html>` element and set `<body>` background-color to #0f0f13 as inline fallback
- [x] T011 [P] Create `frontend/src/features/design-system/teamColors.ts` exporting `TEAM_COLORS` record (mercedes, redbull, red_bull, ferrari, mclaren, aston_martin, alpine), `FALLBACK_TEAM_COLOR` (#8888aa), and `getTeamColor(constructorId: string): string` function per data-model.md

**Checkpoint**: Foundation ready — design tokens, fonts, and team color utilities available for all components

---

## Phase 3: User Story 1 — Consistent Dark Theme Across All Pages (Priority: P1) 🎯 MVP

**Goal**: Apply the dark racing theme with cohesive typography, token colors, and spacing to every existing page (calendar, standings, round detail). Background is near-black, text is white, headings use display font at 700, body uses body font at 400, numeric data uses monospace.

**Independent Test**: Navigate to `/calendar` — background is #0f0f13, card surfaces are #1a1a23, borders are #2a2a38, headings use Inter Bold, body uses Inter Regular, numbers use JetBrains Mono. Repeat for `/standings` and `/rounds/:round`.

### Implementation for User Story 1

- [x] T012 [US1] Update `frontend/src/App.tsx` to apply dark theme classes to layout shell: `bg-background text-foreground font-body min-h-screen` on `<main>`, style nav links with Tailwind utilities replacing inline `style` attributes
- [x] T013 [US1] Migrate `frontend/src/features/calendar/CalendarPage.tsx` from CSS class names (`calendar-page`, `calendar-table`, `data-freshness`, `badge`, `cancelled`, `next-race`) to Tailwind utility classes using design tokens (bg-surface for cards, border-border for table, font-mono for round numbers, text-muted for secondary text)
- [x] T014 [P] [US1] Migrate `frontend/src/features/calendar/NextRaceCard.tsx` from CSS class names (`next-race-card`, `countdown`, `countdown-segment`, `countdown-label`) to Tailwind utility classes (bg-surface, border border-border, rounded-lg, font-mono text-accent-cyan for countdown digits)
- [x] T015 [US1] Migrate `frontend/src/features/standings/StandingsPage.tsx` from inline `style` attributes to Tailwind utility classes: tab buttons with Tailwind styling, table with bg-surface headers, alternating row backgrounds (odd:bg-surface even:bg-background), font-mono for position/points columns
- [x] T016 [US1] Migrate `frontend/src/features/rounds/RoundDetailPage.tsx` from CSS class names (`round-detail-page`, `session-card`, `data-freshness`, `badge`) and inline styles to Tailwind utility classes (bg-surface for session cards, border-border, text-accent-cyan for back link)
- [x] T017 [P] [US1] Migrate `frontend/src/features/rounds/SessionResultsTable.tsx` from CSS class names (`results-table`, `fastest-lap`, `driver-acronym`) to Tailwind utility classes (font-mono for times/positions, bg-surface/bg-background alternating rows)
- [x] T018 [US1] Run `npm run test` in `frontend/` — verify all existing tests pass after theme migration; fix any test selectors broken by class name changes

**Checkpoint**: All pages render with unified dark theme. Fonts, colors, and spacing are consistent.

---

## Phase 4: User Story 2 — Team-Branded Standings and Result Rows (Priority: P1)

**Goal**: StandingsTable shows team color accents on each row with alternating dark stripes. DriverCard shows a team-colored left border strip with monospace gap/points. Both components are integrated into existing pages.

**Independent Test**: Navigate to `/standings` — each row has a left border accent in the correct team color. Navigate to `/rounds/:round` — DriverCard renders with team color left border, position in bold monospace, points/gap in monospace.

### Tests for User Story 2

- [x] T019 [P] [US2] Create `frontend/tests/design-system/DriverCard.test.tsx` — test renders name, number, position, points, gap; test team color left border via `getTeamColor`; test fallback color for unknown constructor; test gap defaults to empty when omitted
- [x] T020 [P] [US2] Create `frontend/tests/design-system/StandingsTable.test.tsx` — test renders title and all rows; test team color accent per row; test alternating row backgrounds; test optional wins column visibility; test empty rows state

### Implementation for User Story 2

- [x] T021 [P] [US2] Create `frontend/src/features/design-system/DriverCard.tsx` implementing `DriverCardProps` from contracts/component-props.md — dark surface bg, 4px left border from `getTeamColor(constructorId)`, position in large bold monospace, points/gap in monospace, name in body font
- [x] T022 [P] [US2] Create `frontend/src/features/design-system/StandingsTable.tsx` implementing `StandingsTableProps` from contracts/component-props.md — header row, alternating odd:bg-surface/even:bg-background rows, 3px left border from `getTeamColor(constructorId)`, position/points in monospace, optional wins column
- [x] T023 [US2] Integrate `StandingsTable` into `frontend/src/features/standings/StandingsPage.tsx` — replace raw HTML tables for both drivers and constructors tabs with StandingsTable component, mapping existing API data to `StandingsRow[]`
- [x] T024 [US2] Run all tests (`npm run test` in `frontend/`) — verify new component tests pass and existing standings tests still pass

**Checkpoint**: Standings page uses StandingsTable with team colors. DriverCard is available for round detail integration.

---

## Phase 5: User Story 3 — Race-Specific Visual Components (Priority: P2)

**Goal**: TireCompound renders colored circle badges, LapTimeDisplay shows monospace time with green/red delta coloring, RaceCountdown displays large monospace digits in accent-cyan. Components are integrated into existing pages.

**Independent Test**: Render each TireCompound type and verify correct color/letter. Render LapTimeDisplay with positive and negative deltas and verify green/red coloring. Render RaceCountdown with a future target and verify monospace accent-cyan digits; render with past target and verify "RACE LIVE" display.

### Tests for User Story 3

- [x] T025 [P] [US3] Create `frontend/tests/design-system/TireCompound.test.tsx` — test renders correct letter/color for each of 5 compounds (soft=S/red, medium=M/yellow, hard=H/white, intermediate=I/blue, wet=W/green); test unknown/null renders grey "?"; test sm and md sizes
- [x] T026 [P] [US3] Create `frontend/tests/design-system/LapTimeDisplay.test.tsx` — test renders time in M:SS.mmm format; test positive delta renders green (#00c853); test negative delta renders red (#ff1744); test zero delta renders white; test deltaOnly mode; test undefined delta shows no delta
- [x] T027 [P] [US3] Create `frontend/tests/design-system/RaceCountdown.test.tsx` — test renders countdown digits in monospace; test expired state shows "RACE LIVE"; test days omitted when zero; test label prop renders below digits

### Implementation for User Story 3

- [x] T028 [P] [US3] Create `frontend/src/features/design-system/TireCompound.tsx` implementing `TireCompoundProps` from contracts/component-props.md — circular badge with compound color bg, white letter label (black on white for hard), grey "?" for unknown/null, sm (24×24) and md (32×32) sizes
- [x] T029 [P] [US3] Create `frontend/src/features/design-system/LapTimeDisplay.tsx` implementing `LapTimeDisplayProps` from contracts/component-props.md — monospace font, time formatted as M:SS.mmm, delta colored positive/negative/neutral, deltaOnly mode, "+" prefix for faster and "−" for slower
- [x] T030 [P] [US3] Create `frontend/src/features/design-system/RaceCountdown.tsx` implementing `RaceCountdownProps` from contracts/component-props.md — large monospace digits in accent-cyan, format DDd HHh MMm SSs (days omitted if zero), "RACE LIVE" in accent-red when expired, optional label below digits
- [x] T031 [US3] Integrate `RaceCountdown` into `frontend/src/features/calendar/NextRaceCard.tsx` — replace raw countdown markup with RaceCountdown component, passing `countdownTarget` as `targetUtc` and "until lights out" as `label`
- [x] T032 [US3] Integrate `LapTimeDisplay` into `frontend/src/features/rounds/SessionResultsTable.tsx` — replace raw `formatLapTime()` calls with LapTimeDisplay component for qualifying times (Q1/Q2/Q3) and practice best lap times; add delta coloring for gap columns
- [x] T033 [US3] Run all tests (`npm run test` in `frontend/`) — verify new component tests pass and existing calendar/rounds tests still pass

**Checkpoint**: All 5 design system components (DriverCard, StandingsTable, TireCompound, LapTimeDisplay, RaceCountdown) built, tested, and integrated.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final validation, build verification, and dependency hygiene

- [x] T034 [P] Run `npm run build` in `frontend/` — verify production Vite build succeeds with no errors or warnings
- [x] T035 [P] Run `npm run lint` in `frontend/` — verify TypeScript compilation has no type errors
- [x] T036 Verify design tokens in browser DevTools: inspect `:root` for all 9 color CSS custom properties and 6 team color custom properties
- [x] T037 Run quickstart.md verification checklist: (1) page background is #0f0f13, (2) `/calendar` uses design system, (3) `/standings` shows team color accents, (4) CSS custom properties visible in DevTools, (5) all tests pass

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion — BLOCKS all user stories
- **US1 — Dark Theme (Phase 3)**: Depends on Foundational (Phase 2) — provides themed pages for US2/US3 integration
- **US2 — Team Branding (Phase 4)**: Depends on Foundational (Phase 2) and US1 (Phase 3) for themed page shells
- **US3 — Race Components (Phase 5)**: Depends on Foundational (Phase 2) and US1 (Phase 3) for themed pages; independent of US2
- **Polish (Phase 6)**: Depends on all user stories being complete

### User Story Dependencies

```
Phase 1: Setup
    ↓
Phase 2: Foundational (tokens, fonts, teamColors.ts)
    ↓
Phase 3: US1 — Dark Theme (page migrations)
   ↓ ↘
Phase 4: US2    Phase 5: US3
(StandingsTable, (TireCompound, LapTimeDisplay,
 DriverCard)      RaceCountdown)
         ↘ ↙
Phase 6: Polish
```

### Within Each User Story

- Tests written FIRST, verified to FAIL before implementation (US2, US3)
- Components before page integrations
- Core implementation before integration
- Verification pass at end of each phase

### Parallel Opportunities

**Phase 1**: T004, T005, T006, T007 can all run in parallel after T001–T003
**Phase 2**: T010, T011 can run in parallel with T008–T009
**Phase 3**: T014, T017 can run in parallel (different files); T013 depends on CalendarPage migration
**Phase 4**: T019, T020, T021, T022 can all run in parallel (different files)
**Phase 5**: T025–T030 can all run in parallel (6 independent files)
**Phase 6**: T034, T035 can run in parallel

### Implementation Strategy

- **MVP scope**: Phase 1 + Phase 2 + Phase 3 (US1) — delivers the unified dark theme across all pages
- **Increment 2**: Phase 4 (US2) — adds team color branding to standings
- **Increment 3**: Phase 5 (US3) — adds race-specific visual components
- **Final**: Phase 6 — build/lint/verification pass
