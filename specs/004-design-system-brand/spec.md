# Feature Specification: Design System and Brand Identity

**Feature Branch**: `004-design-system-brand`
**Created**: April 20, 2026
**Status**: Draft
**Input**: User description: "Install shadcn/ui components via CLI. Establish design tokens in tailwind.config.ts with specified colors, typography, and team colors. Build DriverCard, LapTimeDisplay, TireCompound, RaceCountdown, and StandingsTable components."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Consistent Dark Theme Across All Pages (Priority: P1) 🎯 MVP

A user visiting the F1 Race Intelligence dashboard sees a cohesive dark racing theme across every page — calendar, standings, and round detail. Background colors, text styles, borders, and spacing follow a unified design language. The near-black background with muted borders and bright accent colors creates a premium motorsport aesthetic.

**Why this priority**: Without the foundational theme tokens and global styles, no individual component can look correct. This is the design infrastructure that everything else depends on.

**Independent Test**: Navigate to the calendar page and verify the background is near-black (#0f0f13), card surfaces are dark (#1a1a23), borders are subtle (#2a2a38), primary text is white, and muted text is dimmed. Headings use the display font at weight 700. Body text uses the body font at weight 400. Numeric data (positions, lap times, points) renders in a monospace font.

**Acceptance Scenarios**:

1. **Given** a user loads any page, **When** the page renders, **Then** the background color is near-black, text is white, and borders are subtle dark tones
2. **Given** a user views a data table, **When** numeric values are displayed, **Then** they render in a monospace font distinct from body text
3. **Given** a user views headings, **When** the page renders, **Then** headings use the display font at bold weight (700) and body text uses the regular weight (400)
4. **Given** a user views the application on any supported screen size, **When** the theme loads, **Then** all design tokens are applied consistently via CSS custom properties

---

### User Story 2 - Team-Branded Standings and Result Rows (Priority: P1)

A user viewing standings or race results sees each row accented with the correct constructor team color. A Mercedes row has a teal accent, a Ferrari row has red, a McLaren row has orange. The StandingsTable component uses alternating dark row stripes with team color indicators, and the DriverCard component shows a team-colored left border strip.

**Why this priority**: Team color identity is the single most recognizable visual element in F1 data presentation. Without it, the dashboard looks like a generic data table.

**Independent Test**: Navigate to the standings page and verify each constructor row has the correct team color accent. Navigate to a round detail page, verify the DriverCard components display a left border in the driver's team color and numeric data (gap, points) in monospace font.

**Acceptance Scenarios**:

1. **Given** a user views the standings table, **When** results render, **Then** each row displays an accent matching the driver's constructor color (Mercedes=#00d2be, Red Bull=#3671c6, Ferrari=#e8002d, McLaren=#ff8000, Aston Martin=#358c75, Alpine=#ff87bc)
2. **Given** a user views a round detail page, **When** a DriverCard renders, **Then** it shows a dark surface background, a left border strip in the driver's team color, and gap/points values in monospace font
3. **Given** a user views the standings table, **When** rows render, **Then** alternating rows use slightly different dark background shades for readability

---

### User Story 3 - Race-Specific Visual Components (Priority: P2)

A user viewing race data sees purpose-built visual elements: tire compound indicators as colored circle badges (Soft=red, Medium=yellow, Hard=white, Intermediate=blue, Wet=green), lap time displays in monospace with green/red delta coloring for faster/slower comparisons, and a race countdown with large monospace digits in accent color.

**Why this priority**: These components make F1 data immediately legible to fans who are accustomed to specific visual conventions from broadcast and official timing screens. They are high-impact but depend on US1 and US2 being in place first.

**Independent Test**: Render a TireCompound badge for each compound type and verify the correct color is displayed. Render a LapTimeDisplay with a positive delta and verify green coloring; render with a negative delta and verify red coloring. Render a RaceCountdown and verify large monospace digits with accent highlight.

**Acceptance Scenarios**:

1. **Given** a tire compound value of "soft", **When** the TireCompound badge renders, **Then** it displays a red circle badge with "S" label
2. **Given** a tire compound value of "medium", **When** the TireCompound badge renders, **Then** it displays a yellow circle badge with "M" label
3. **Given** a tire compound value of "hard", **When** the TireCompound badge renders, **Then** it displays a white circle badge with "H" label
4. **Given** a tire compound value of "intermediate", **When** the TireCompound badge renders, **Then** it displays a blue circle badge with "I" label
5. **Given** a tire compound value of "wet", **When** the TireCompound badge renders, **Then** it displays a green circle badge with "W" label
6. **Given** a lap time delta that is faster than reference, **When** the LapTimeDisplay renders, **Then** the delta value is shown in green (#00c853)
7. **Given** a lap time delta that is slower than reference, **When** the LapTimeDisplay renders, **Then** the delta value is shown in red (#ff1744)
8. **Given** a future race event, **When** the RaceCountdown renders, **Then** it displays days/hours/minutes/seconds in large monospace digits with the accent-cyan color

---

### Edge Cases

- What happens when a team color is not defined for a constructor? Falls back to a neutral muted color (#8888aa)
- What happens when a tire compound is unknown or null? Renders a grey circle badge with "?" label
- What happens when a lap time delta is exactly zero? Renders in the default text color (white), not green or red
- What happens when the countdown reaches zero? Displays "RACE LIVE" or equivalent indicator instead of 00:00:00:00
- What happens when the application loads before web fonts are available? System font fallback stack prevents layout shift; monospace falls back to system monospace

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST install shadcn/ui component library via CLI and configure it for the project
- **FR-002**: System MUST install and configure Tailwind CSS with design tokens defined in `tailwind.config.ts`
- **FR-003**: System MUST define background (#0f0f13), surface (#1a1a23), border (#2a2a38), accent-red (#e8002d), accent-cyan (#00d2ff), text-primary (#ffffff), text-muted (#8888aa), positive (#00c853), and negative (#ff1744) as Tailwind color tokens
- **FR-004**: System MUST configure display/heading typography using Inter or Geist at weight 700
- **FR-005**: System MUST configure body typography using Inter at weight 400
- **FR-006**: System MUST configure data/number typography using JetBrains Mono at weight 500
- **FR-007**: System MUST define team colors as CSS custom properties: mercedes (#00d2be), redbull (#3671c6), ferrari (#e8002d), mclaren (#ff8000), aston (#358c75), alpine (#ff87bc)
- **FR-008**: System MUST provide a DriverCard component with dark surface background, team color left border, and monospace font for gap and points values
- **FR-009**: System MUST provide a LapTimeDisplay component with monospace font rendering, and green/red delta coloring for positive/negative time differences
- **FR-010**: System MUST provide a TireCompound component rendering colored circle badges — Soft=red, Medium=yellow, Hard=white, Intermediate=blue, Wet=green — with compound letter labels
- **FR-011**: System MUST provide a RaceCountdown component displaying large monospace digits in accent color
- **FR-012**: System MUST provide a StandingsTable component with alternating dark row stripes, team color accents, and position number styling
- **FR-013**: All new components MUST be reusable and accept props for data binding (not hardcoded)
- **FR-014**: All new components MUST include unit tests validating rendering and visual states
- **FR-015**: System MUST apply the design system to existing pages (calendar, standings, round detail) as a migration step

### Constitution Alignment *(mandatory)*

- **CA-001**: Feature runs on React frontend; no backend changes required. shadcn/ui and Tailwind CSS are frontend-only dependencies.
- **CA-002**: UI components call backend APIs only — no new third-party API calls introduced.
- **CA-003**: Not applicable — no new OpenF1 data access patterns.
- **CA-004**: Not applicable — no secrets or Key Vault changes.
- **CA-005**: Not applicable — no ingress or firewall changes.
- **CA-006**: Not applicable — no new Kubernetes resources. Frontend container image is rebuilt via existing Dockerfile.
- **CA-007**: Feature delivery fits existing GitHub Actions pipeline: lint (TypeScript) → test (Vitest) → build (Vite) → push → deploy.
- **CA-008**: Not applicable — no backend logging changes.
- **CA-009**: New dependencies: shadcn/ui (component primitives based on Radix UI), Tailwind CSS (utility-first CSS framework), @fontsource/inter, @fontsource/jetbrains-mono (self-hosted web fonts). Justification: shadcn/ui provides accessible, unstyled component primitives that are customized via Tailwind; Tailwind enables token-based theming without custom CSS; self-hosted fonts ensure consistent rendering without external font CDN calls.
- **CA-010**: All work is traceable to this specification.

### Key Entities

- **DesignToken**: A named color, typography, or spacing value defined in Tailwind config and consumed as CSS custom properties. Key attributes: token name, hex value, usage context.
- **TeamColor**: A mapping from constructor identifier to hex color value, defined as CSS custom properties and consumed by team-branded components.
- **TireCompound**: An enumeration of five tire types (soft, medium, hard, intermediate, wet), each mapped to a display color and letter label.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All 9 design token colors are defined in Tailwind config and render correctly in the browser
- **SC-002**: All 6 team colors are accessible as CSS custom properties and render in standings/result rows
- **SC-003**: Three font families (display, body, monospace) load and render correctly with no layout shift on page load
- **SC-004**: All 5 new components (DriverCard, LapTimeDisplay, TireCompound, RaceCountdown, StandingsTable) render correctly with test data
- **SC-005**: Each new component has at least one passing unit test covering its primary visual states
- **SC-006**: Existing pages (calendar, standings, round detail) adopt the new design system with no functional regressions
- **SC-007**: All existing tests continue to pass after design system migration

## Assumptions

- The project does not currently use Tailwind CSS; it will be installed as part of this feature
- shadcn/ui will be installed via its CLI (`npx shadcn@latest init`) which sets up Tailwind, CSS variables, and the `components/ui` directory convention
- Font files will be self-hosted via `@fontsource` packages rather than loaded from Google Fonts CDN, consistent with CA-002 (no external third-party calls from the frontend)
- Geist font is the preferred display font if available in @fontsource; otherwise Inter is used for both display and body
- The existing CSS (likely in `src/index.css` or similar) will be migrated to Tailwind utilities; custom CSS is minimized
- Mobile responsiveness is a secondary concern — the dashboard targets desktop-first but should not break on smaller screens
- The 6 team colors listed cover the top constructor teams; remaining teams will be added incrementally as needed, with a neutral fallback color
