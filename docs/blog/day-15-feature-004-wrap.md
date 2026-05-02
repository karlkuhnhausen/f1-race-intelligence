# Day 15: Feature 4 — What a "Pure Frontend" Feature Actually Cost

*Posted May 2, 2026 · Karl Kuhnhausen*

---

[Day 11](day-11-design-system.md) opened Feature 4 with a single claim: every API call, every Cosmos query, every route would stay exactly the same — only the look would change. "A design system in an afternoon." Five days of blog posts later, Feature 4 is closed: 37 tasks across 6 phases, all checked, [PR #29](https://github.com/karlkuhnhausen/f1-race-intelligence/pull/29) merged, branch deleted.

That single claim — "pure frontend" — turned out to be true *and* misleading. Truth: I didn't touch the Go service. Misleading: the design pass found three bugs the old grey-table layout had been politely hiding for weeks, and fixing them was where most of the actual code went.

This post is the retrospective. Not a how-to, not a postmortem of any one bug, but the shape of the feature once it's done.

---

## What got built

The 37 tasks broke into four user stories:

| Story | Priority | What |
|---|---|---|
| US1 — Consistent dark theme | P1 (MVP) | Tailwind v4 + shadcn/ui, design tokens, self-hosted Inter / JetBrains Mono, near-black background everywhere |
| US2 — Team-branded rows | P1 | Team color accents on standings and DriverCard, alternating row stripes |
| US3 — Race-specific components | P2 | TireCompound badges, LapTimeDisplay with green/red deltas, RaceCountdown |
| US4 — Migrate existing pages | P3 | Calendar, standings, round detail rebuilt on the design system |

Phase order was the boring kind: Setup → Foundational → US1 → US2 → US3 → US4. That ordering matters. Tokens before components. Components before page migrations. Each phase has a checkpoint where you can stop, deploy, and have something better than what was on master before.

The actual artifacts that showed up in the repo:

- `frontend/src/globals.css` with a `@theme` block defining nine semantic color tokens, six team-color CSS custom properties, and three font families
- `frontend/src/features/design-system/teamColors.ts` — single canonical map from constructor id to hex
- shadcn/ui primitives in `frontend/src/components/ui/` — table, card, badge
- Race-specific components under `frontend/src/features/design-system/` — `DriverCard`, `LapTimeDisplay`, `TireCompound`, `RaceCountdown`, `StandingsTable`, `SessionResultsTable`
- Self-hosted fonts via `@fontsource/inter` and `@fontsource-variable/jetbrains-mono` so we don't depend on Google's CDN at runtime

Backend: zero lines changed across the full feature. The constitution's three-tier boundary held; nothing leaked into Go because there was nothing for Go to do.

---

## The arc through five blog posts

Looking at the git log, Feature 4 maps cleanly to four posts:

- **[Day 11](day-11-design-system.md)** — design system kickoff. Token install, shadcn add, font import, the visual transformation from generic admin panel to F1 dashboard.
- **[Day 12](day-12-status-badge-bug.md)** — first bug surfaced by the new design. Future races showing **COMPLETED** because the `Status` field was being hardcoded in the ingest transform.
- **[Day 14](day-14-design-polish.md)** — three bugs that the polished round-detail page made obvious: stale meeting status, wrong session sort order on completed weekends, and three near-duplicate result tables that needed consolidating.
- **Day 15** (this one) — the wrap.

Day 13 was a parallel ops track ([allowlist guard](day-13-allowlist-guard.md)), not part of Feature 4. It just happened during the same week.

That's an interesting ratio. One post on the design itself, two posts on bugs the design exposed. The pattern is real: when you put data on a page that *looks* official, you start reading it like it's official. Wrong status badges that were just grey text on grey table for ten days became immediately, viscerally wrong the moment they were pill-shaped and color-coded.

The design system is the bug-finding tool.

---

## What "pure frontend" actually means

Feature 4 was scoped as frontend-only. The constitution's three-tier boundary — UI never calls external APIs, all external integration goes through the backend — meant any change that needed Cosmos or OpenF1 was out of scope. That stayed true. But "frontend-only" doesn't mean "small."

The visible diff:

```
Files changed across Feature 4 PRs: ~80
Lines added:   ~3,200
Lines removed: ~1,400
Net:           +1,800
Frontend tests: 38 → 57 (+19)
Backend tests: unchanged
```

Most of those lines aren't components. They're:

- Token definitions in `globals.css`
- Tailwind v4 config (a different shape from v3 — `@theme` blocks, not `tailwind.config.ts`)
- Test files for every new component (FR-014 in the spec required unit tests for all new components, no exceptions)
- The migration of existing pages onto the new primitives, which mostly meant *deleting* old bespoke styling rather than adding new
- The Day 14 cleanup that consolidated three near-duplicate result tables into one

The deleted lines are the interesting part. Three legacy result components — `RaceResults.tsx`, `QualifyingResults.tsx`, `PracticeResults.tsx` — and their tests came out in [Day 14](day-14-design-polish.md). That was −499 lines on a single commit. The shared `SessionResultsTable` had been written during Phase 5 of Feature 4 but never wired in. Day 14 wired it in and then collected the now-unused legacy code on the way out.

---

## What the constitution caught (and didn't)

A few decisions where the constitution was the deciding factor:

**Self-hosted fonts.** `@fontsource/inter` over a Google Fonts `<link>` tag. Constitutional principle 5 — minimal external dependencies, written justification. The justification: a CDN font is a runtime dependency on a third party we don't control, with no caching strategy and no offline story. Self-hosted bytes ship inside our own image, are versioned with our deploys, and the only way they break is if we break them.

**Tailwind v4 over v3.** v4's `@theme` block is the supported way to define design tokens going forward. Picking v3 would have meant signing up for a future migration. Picking v4 meant accepting a config shape with less Stack Overflow coverage. The constitution doesn't speak to versions, but its spirit ("write justification for any non-trivial dependency choice") nudges toward the choice that ages better.

**No new dependencies for the race-specific components.** TireCompound, LapTimeDisplay, RaceCountdown — all built from primitives. No `react-tire-icons` package, no `@react-oss/lap-timer`. They don't exist anyway, but the principle is that the dependency budget for a frontend feature was: shadcn/ui primitives + `class-variance-authority` + `clsx` + `tailwind-merge` + the two font packages. Stop there.

What the constitution didn't catch: nothing about how a visual refresh could surface backend-shaped bugs. The Day 12 and Day 14 bugs were on the data side and they'd been there for weeks. The constitution governs architecture; it doesn't say "be suspicious of derived state stored as fact." That's a separate engineering instinct that I now have a couple of paragraphs of evidence for.

---

## The bug-finding pattern, formalized

If I had to compress what Feature 4 taught me into one sentence, it would be:

> Polishing a UI is the cheapest way to find data bugs in the system underneath it.

The mechanism: a generic data table treats every cell the same. A designed UI separates *kinds* of data — status pills, monospace numbers, team color accents, podium borders. The moment the visual treatment carries semantic meaning, any cell that's semantically wrong becomes visually wrong. The bug doesn't change. Your ability to see it does.

This is why the order of Feature 4 mattered. If I'd done the bug fixes first and the design pass second, I'd never have found bugs 2 and 3 from Day 14 — they were only visible because the new layout made the *right* answer obvious by contrast.

I don't think this generalizes to "do design before backend correctness." That would be silly. But for a project that's already past correctness in the data layer and into UX, the design pass is a high-ROI bug-finding pass *as a side effect*.

---

## What's live

- **Feature 4**: 37/37 tasks done, [PR #29](https://github.com/karlkuhnhausen/f1-race-intelligence/pull/29) merged, branch deleted.
- **Frontend**: 57 tests passing. Calendar, standings, and round detail all rebuilt on the new design system. Self-hosted Inter and JetBrains Mono. Team colors on every row. Podium accents on race results. "Not Classified" divider for DNFs.
- **Deployed**: `https://f1raceintel.westus3.cloudapp.azure.com` is running the design-system build.
- **Repo cleanup**: [PR #30](https://github.com/karlkuhnhausen/f1-race-intelligence/pull/30) added a Playwright-based screenshot capture tool under `scripts/screenshots/` so future blog posts can grab consistent desktop + mobile snapshots without me hand-cropping browser windows.

---

## What's next

A few things that surfaced during Feature 4 and were deferred:

- **Pre-season testing rounds** — OpenF1 currently numbers Bahrain testing as round 1 and round 2, which pushes Australia to round 3. The calendar shows what OpenF1 sends. The fix is in the ingest transform: filter testing rounds out of the canonical round numbering. Small, contained, probably one PR.
- **Real team logos in result tables** — the team-color swatch is a placeholder for an `<img>`. The licensing work for F1 team marks is the gating item, not the component change.
- **Feature 5** — undecided. Live timing is the obvious "next big thing" if this were a product, but the constitution's caching principle (3) means any live feature lands as a backend poller writing to Cosmos before any UI sees it. So it's more work than it looks.

The branch protection rule on master held through Feature 4 — every change shipped via PR, including the chore commits and this blog post. That feels right.

---

[← Day 14: Three Bugs, One Cause — When Stale Cached State Becomes a UX Smell](day-14-design-polish.md)
