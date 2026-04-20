# Research: Design System and Brand Identity

## Decision 1: Tailwind CSS version and installation approach

- Decision: Use Tailwind CSS v4 with the `@tailwindcss/vite` plugin for native Vite integration. Configure via `tailwind.config.ts` and `globals.css` with `@tailwind` directives.
- Rationale: The project already uses Vite 5 as its build tool. Tailwind v4's Vite plugin provides first-class integration with faster builds compared to PostCSS-only setups. The project currently has zero CSS files, so there is no migration burden.
- Alternatives considered:
  - Tailwind v3 via PostCSS: rejected because v4 is current, offers better performance, and shadcn/ui supports it.
  - CSS Modules or styled-components: rejected because the spec mandates Tailwind CSS and utility-first approach.

## Decision 2: shadcn/ui initialization and component selection

- Decision: Initialize shadcn/ui via `npx shadcn@latest init` with the "new-york" style variant and slate base color. Install only the primitives needed: `table`, `card`, `badge`. Custom F1 components will be built on top of these primitives.
- Rationale: shadcn/ui copies component source into the project (no runtime dependency on shadcn package). The "new-york" style is closer to the desired dark, clean aesthetic. Only installing needed primitives avoids bloat.
- Alternatives considered:
  - Install all shadcn/ui components: rejected due to dependency discipline — unused code is unnecessary.
  - Build primitives from scratch without shadcn: rejected because shadcn provides accessible Radix-based primitives with proper ARIA attributes, saving implementation time.
  - "default" style variant: rejected because "new-york" has tighter spacing and bolder styling that suits a dashboard.

## Decision 3: Font hosting strategy

- Decision: Use `@fontsource/inter` (display + body, weights 400 and 700) and `@fontsource-variable/jetbrains-mono` (monospace, weight 500) as npm packages. Import font CSS in `globals.css`.
- Rationale: Self-hosted fonts satisfy CA-002 (no external third-party API calls from frontend). @fontsource packages are the standard approach for self-hosting Google Fonts in npm-based projects. Fonts are bundled into the Vite build output and served from the same origin.
- Alternatives considered:
  - Google Fonts CDN: rejected because it violates CA-002 (external third-party calls).
  - Geist font for display: rejected because @fontsource/geist-sans availability is less established than Inter, and Inter is explicitly listed in the spec as the primary choice.
  - Manual font file hosting: rejected because @fontsource handles subsetting, formats (woff2), and CSS generation automatically.

## Decision 4: Design token architecture

- Decision: Define all 9 color tokens and team colors as CSS custom properties in `globals.css`, and reference them in `tailwind.config.ts` via `var()` syntax. This enables both Tailwind utility usage (`bg-surface`, `text-muted`) and direct CSS variable access for dynamic team colors.
- Rationale: CSS custom properties allow runtime theme switching and dynamic application of team colors via `style` attributes. Tailwind's `extend.colors` configuration maps these variables to utility classes. This is the standard pattern recommended by shadcn/ui.
- Alternatives considered:
  - Tailwind-only tokens without CSS variables: rejected because team colors must be applied dynamically per row/component based on data.
  - CSS-in-JS (emotion/styled-components): rejected per spec mandate for Tailwind approach.

## Decision 5: Team color mapping implementation

- Decision: Create a `teamColors.ts` module exporting a `Record<string, string>` mapping constructor identifiers to hex colors, plus a `getTeamColor(constructorId: string): string` function with fallback to `#8888aa`. CSS custom properties for the 6 specified teams are also defined in `globals.css` for use in Tailwind utilities.
- Rationale: The mapping needs to be accessible both in Tailwind classes (for static styling) and in inline styles (for dynamic per-row coloring). A TypeScript module provides type-safe access. The fallback color handles unrecognized teams per edge case requirements.
- Alternatives considered:
  - Tailwind-only team colors: rejected because team color application is data-driven and requires runtime resolution.
  - Server-side team color resolution: rejected because this is display-only logic and belongs in the UI tier.

## Decision 6: Component organization

- Decision: Place the 5 new F1-specific components in `src/features/design-system/` and shadcn/ui primitives in `src/components/ui/`. Export a `teamColors.ts` utility alongside the components.
- Rationale: Follows the established `features/{domain}/` convention used by calendar, standings, and rounds features. The `design-system` domain groups reusable visual components distinct from page-level features. shadcn/ui convention places generated primitives in `components/ui/`.
- Alternatives considered:
  - Flat `src/components/` directory: rejected because it breaks the established feature-based structure.
  - Separate `src/design-system/` top-level directory: rejected because `features/` is the established organizational pattern.

## Decision 7: Page migration strategy

- Decision: Migrate existing pages incrementally: first install Tailwind + global styles (US1), then replace inline `style` attributes and CSS class names in existing components with Tailwind utility classes. Existing tests must pass after each migration step.
- Rationale: The existing frontend uses inline `style` attributes (see App.tsx nav, StandingsPage buttons, RoundDetailPage links) and a few CSS class names (like `calendar-table`, `results-table`) that currently have no CSS definitions (no CSS files exist). Migration replaces these with Tailwind classes. Incremental approach minimizes regression risk.
- Alternatives considered:
  - Big-bang rewrite: rejected because it risks breaking existing functionality.
  - Keep old styles alongside Tailwind: rejected because it creates inconsistency and maintenance burden.

## Decision 8: Tailwind CSS v4 compatibility with shadcn/ui

- Decision: Use Tailwind CSS v4 with shadcn/ui's v4-compatible initialization. shadcn/ui v2.x supports Tailwind v4 natively via `npx shadcn@latest init`. The CSS configuration uses `@import "tailwindcss"` syntax instead of the v3 `@tailwind` directives.
- Rationale: Tailwind v4 uses a CSS-first configuration approach. shadcn/ui's latest CLI detects Tailwind v4 and generates compatible configuration. This avoids the need for `tailwind.config.ts` in many cases — tokens are defined directly in CSS via `@theme`.
- Alternatives considered:
  - Stick with Tailwind v3: rejected because v4 is the current version and shadcn/ui supports it; v3 would be immediately outdated.

## Open questions (resolved)

- Geist vs Inter: Resolved — use Inter for both display and body. Geist is not needed.
- Tailwind v3 vs v4: Resolved — use v4 with `@tailwindcss/vite` plugin.
- Component library scope: Resolved — install only `table`, `card`, `badge` primitives from shadcn/ui.
