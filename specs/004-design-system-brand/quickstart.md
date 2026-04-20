# Quickstart: Design System and Brand Identity (Feature 004)

## Prerequisites

- Node.js 18+
- npm or pnpm
- Working frontend dev server (`cd frontend && npm run dev`)

## Installation Steps

### 1. Install Tailwind CSS v4 with Vite plugin

```bash
cd frontend
npm install tailwindcss @tailwindcss/vite
```

### 2. Initialize shadcn/ui

```bash
npx shadcn@latest init
```

When prompted:
- Style: **New York**
- Base color: **Slate**
- CSS variables: **Yes**

This creates `components.json`, `src/lib/utils.ts`, and updates CSS with Tailwind v4 configuration.

### 3. Install fonts

```bash
npm install @fontsource/inter @fontsource-variable/jetbrains-mono
```

### 4. Install needed shadcn/ui components

```bash
npx shadcn@latest add table card badge
```

### 5. Configure Vite for Tailwind

Update `vite.config.ts` to include the Tailwind plugin:

```typescript
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  // ... existing config
});
```

### 6. Set up globals.css with design tokens

Create `src/globals.css` with Tailwind imports and CSS custom properties for all design tokens (colors, team colors, font families).

### 7. Import fonts and globals in main.tsx

```typescript
import "@fontsource/inter/400.css";
import "@fontsource/inter/700.css";
import "@fontsource-variable/jetbrains-mono";
import "./globals.css";
```

## Development Workflow

```bash
cd frontend
npm run dev          # Start Vite dev server with hot reload
npm run test         # Run Vitest
npm run lint         # TypeScript type check
npm run build        # Production build
```

## Verification

1. Open `http://localhost:5173` — page background should be near-black (#0f0f13)
2. Navigate to `/calendar` — table should use design system styling
3. Navigate to `/standings` — rows should show team color accents
4. Check browser DevTools — CSS custom properties should be visible on `:root`
5. Run `npm run test` — all existing + new component tests pass

## Key Files

| File | Purpose |
|------|---------|
| `src/globals.css` | Tailwind directives, CSS custom properties, font imports |
| `src/lib/utils.ts` | `cn()` utility for class merging (shadcn convention) |
| `src/features/design-system/teamColors.ts` | Team color mapping + fallback |
| `src/features/design-system/DriverCard.tsx` | Driver card with team color border |
| `src/features/design-system/LapTimeDisplay.tsx` | Monospace time + delta coloring |
| `src/features/design-system/TireCompound.tsx` | Colored compound badges |
| `src/features/design-system/RaceCountdown.tsx` | Large monospace countdown |
| `src/features/design-system/StandingsTable.tsx` | Team-branded standings table |
