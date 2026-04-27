# Data Model: Design System and Brand Identity

This feature is frontend-only. There are no backend data model changes. The "entities" below are TypeScript types and design token definitions consumed by UI components.

## Entity: DesignToken

Design tokens are named values defined as CSS custom properties and mapped to Tailwind utility classes.

### Color Tokens

| Token Name       | CSS Variable           | Hex Value  | Usage                              | Spec Ref |
|------------------|------------------------|------------|------------------------------------|----------|
| background       | `--background`         | `#0f0f13`  | Page/app background                | FR-003   |
| surface          | `--surface`            | `#1a1a23`  | Card/panel backgrounds             | FR-003   |
| border           | `--border`             | `#2a2a38`  | Dividers, table borders            | FR-003   |
| accent-red       | `--accent-red`         | `#e8002d`  | Primary red accent (F1 brand)      | FR-003   |
| accent-cyan      | `--accent-cyan`        | `#00d2ff`  | Highlight, countdown, links        | FR-003   |
| text-primary     | `--foreground`         | `#ffffff`  | Primary body text                  | FR-003   |
| text-muted       | `--muted-foreground`   | `#8888aa`  | Secondary/dimmed text              | FR-003   |
| positive         | `--positive`           | `#00c853`  | Faster delta, gains                | FR-003   |
| negative         | `--negative`           | `#ff1744`  | Slower delta, losses               | FR-003   |

### Typography Tokens

| Token Name  | Font Family           | Weight | Usage                        | Spec Ref |
|-------------|-----------------------|--------|------------------------------|----------|
| display     | Inter                 | 700    | Headings (h1–h3)            | FR-004   |
| body        | Inter                 | 400    | Body text, labels            | FR-005   |
| mono        | JetBrains Mono        | 500    | Positions, lap times, points | FR-006   |

## Entity: TeamColor

A mapping from constructor identifier string to hex color value, defined as CSS custom properties and available via TypeScript utility.

| Constructor ID  | CSS Variable         | Hex Value  | Spec Ref |
|-----------------|----------------------|------------|----------|
| mercedes        | `--team-mercedes`    | `#00d2be`  | FR-007   |
| redbull         | `--team-redbull`     | `#3671c6`  | FR-007   |
| ferrari         | `--team-ferrari`     | `#e8002d`  | FR-007   |
| mclaren         | `--team-mclaren`     | `#ff8000`  | FR-007   |
| aston_martin    | `--team-aston`       | `#358c75`  | FR-007   |
| alpine          | `--team-alpine`      | `#ff87bc`  | FR-007   |
| _fallback_      | —                    | `#8888aa`  | Edge case |

### TypeScript type

```typescript
// src/features/design-system/teamColors.ts
export const TEAM_COLORS: Record<string, string> = {
  mercedes: "#00d2be",
  redbull: "#3671c6",
  red_bull: "#3671c6",
  ferrari: "#e8002d",
  mclaren: "#ff8000",
  aston_martin: "#358c75",
  alpine: "#ff87bc",
};

export const FALLBACK_TEAM_COLOR = "#8888aa";

export function getTeamColor(constructorId: string): string {
  return TEAM_COLORS[constructorId.toLowerCase()] ?? FALLBACK_TEAM_COLOR;
}
```

## Entity: TireCompound

An enumeration of tire types with associated display properties.

| Compound       | Letter | Color    | Hex Value  | Spec Ref |
|----------------|--------|----------|------------|----------|
| soft           | S      | Red      | `#e8002d`  | FR-010   |
| medium         | M      | Yellow   | `#ffc107`  | FR-010   |
| hard           | H      | White    | `#ffffff`  | FR-010   |
| intermediate   | I      | Blue     | `#2196f3`  | FR-010   |
| wet            | W      | Green    | `#4caf50`  | FR-010   |
| _unknown/null_ | ?      | Grey     | `#8888aa`  | Edge case |

### TypeScript type

```typescript
// src/features/design-system/TireCompound.tsx (inline type)
interface TireCompoundConfig {
  letter: string;
  color: string;
  label: string;
}

const TIRE_COMPOUNDS: Record<string, TireCompoundConfig> = {
  soft:         { letter: "S", color: "#e8002d", label: "Soft" },
  medium:       { letter: "M", color: "#ffc107", label: "Medium" },
  hard:         { letter: "H", color: "#ffffff", label: "Hard" },
  intermediate: { letter: "I", color: "#2196f3", label: "Intermediate" },
  wet:          { letter: "W", color: "#4caf50", label: "Wet" },
};

const UNKNOWN_COMPOUND: TireCompoundConfig = { letter: "?", color: "#8888aa", label: "Unknown" };
```

## Relationships

- **DriverCard** consumes `TeamColor` (left border color) and `DesignToken.mono` (gap/points display)
- **StandingsTable** consumes `TeamColor` (row accent) and `DesignToken` (alternating row backgrounds: surface, background)
- **LapTimeDisplay** consumes `DesignToken.positive`, `DesignToken.negative`, and `DesignToken.mono`
- **TireCompound** consumes its own color mapping (self-contained)
- **RaceCountdown** consumes `DesignToken.accent-cyan` and `DesignToken.mono`

## State Transitions

### RaceCountdown states

| State        | Condition                    | Display                                  |
|--------------|------------------------------|------------------------------------------|
| counting     | target > now                 | DD:HH:MM:SS in monospace accent-cyan     |
| expired      | target <= now                | "RACE LIVE" indicator                    |

### LapTimeDisplay delta states

| State    | Condition     | Color    |
|----------|---------------|----------|
| faster   | delta > 0     | positive (#00c853) |
| slower   | delta < 0     | negative (#ff1744) |
| neutral  | delta === 0   | text-primary (#ffffff) |
