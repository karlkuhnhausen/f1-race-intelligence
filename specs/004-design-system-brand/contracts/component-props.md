# UI Component Contracts: Design System and Brand Identity

This feature exposes no external APIs. The contracts below define the public TypeScript prop interfaces for each reusable component, serving as the interface contract between page-level features and the design system.

## DriverCard

```typescript
interface DriverCardProps {
  /** Driver's display name (e.g., "Max Verstappen") */
  name: string;
  /** Driver's racing number */
  number: number;
  /** Constructor/team identifier for color lookup (e.g., "redbull") */
  constructorId: string;
  /** Current championship position */
  position: number;
  /** Championship points total */
  points: number;
  /** Gap to leader as formatted string (e.g., "+12.5s" or "LEADER") */
  gap?: string;
}
```

**Visual contract**:
- Dark surface background (`--surface`)
- Left border strip: 4px solid, color from `getTeamColor(constructorId)`
- Position number: large, bold, monospace
- Points and gap: monospace font
- Name: body font

## LapTimeDisplay

```typescript
interface LapTimeDisplayProps {
  /** Lap time in seconds (e.g., 87.654) */
  time: number;
  /** Time delta in seconds; positive = faster, negative = slower, zero = neutral */
  delta?: number;
  /** If true, display only the delta (not the absolute time) */
  deltaOnly?: boolean;
}
```

**Visual contract**:
- Time rendered in monospace font, formatted as `M:SS.mmm`
- Delta rendered adjacent to time (or standalone if `deltaOnly`)
- Delta color: `--positive` (#00c853) if delta > 0, `--negative` (#ff1744) if delta < 0, `--foreground` (#ffffff) if delta === 0
- Delta prefix: `+` for faster, `-` for slower

## TireCompound

```typescript
interface TireCompoundProps {
  /** Tire compound identifier: "soft" | "medium" | "hard" | "intermediate" | "wet" | null */
  compound: string | null;
  /** Optional: render as small inline badge (default) or larger display */
  size?: "sm" | "md";
}
```

**Visual contract**:
- Circular badge with compound color background
- White letter label centered (except hard compound: black letter on white background)
- Unknown/null compound: grey badge with "?"
- `sm` size: 24×24px; `md` size: 32×32px

## RaceCountdown

```typescript
interface RaceCountdownProps {
  /** ISO 8601 UTC datetime string of the countdown target */
  targetUtc: string;
  /** Label shown below the countdown (e.g., "until lights out") */
  label?: string;
}
```

**Visual contract**:
- Digits rendered in large monospace font
- Accent-cyan color (`--accent-cyan`) for digits
- Format: `DDd HHh MMm SSs` (days omitted if zero)
- When expired: displays "RACE LIVE" in accent-red

## StandingsTable

```typescript
interface StandingsRow {
  position: number;
  name: string;
  constructorId: string;
  points: number;
  wins?: number;
}

interface StandingsTableProps {
  /** Table heading */
  title: string;
  /** Ordered rows of standings data */
  rows: StandingsRow[];
  /** Column configuration — which optional columns to show */
  columns?: ("wins")[];
}
```

**Visual contract**:
- Table with header row
- Alternating row backgrounds: odd rows `--surface`, even rows `--background`
- Left border accent on each row: 3px solid, color from `getTeamColor(constructorId)`
- Position column: monospace, bold
- Points column: monospace
- Name column: body font
