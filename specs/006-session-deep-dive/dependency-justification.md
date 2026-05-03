# Dependency Justification: recharts (Feature 006)

**Constitution Reference**: CA-009 — "New frontend charting dependency (recharts) MUST include explicit justification documenting why it's needed and why alternatives were not chosen."

## Summary

| Field | Value |
|-------|-------|
| **Package** | `recharts` |
| **Version** | ^2.12 |
| **Registry** | npm |
| **Owner** | @karlkuhnhausen |
| **Purpose** | React charting library for session analysis visualizations |
| **License** | MIT |
| **Weekly Downloads** | ~2.5M (as of 2026-05) |
| **Bundle Size Impact** | ~150KB gzipped (tree-shakeable — only imported components bundled) |

## Why It's Needed

Feature 006 requires **five distinct chart types** for the session deep dive page:

1. **Position Chart** — multi-line chart (20 series × 60 data points) with inverted Y-axis
2. **Gap-to-Leader Chart** — multi-line chart showing time gaps in seconds
3. **Tire Strategy Swimlane** — horizontal stacked bar chart with compound-colored segments
4. **Pit Stop Timeline** — scatter/bubble chart with categorical Y-axis
5. **Overtake Annotations** — reference markers overlaid on the position chart

Building these from scratch with raw SVG or Canvas would require significant implementation effort (estimated 2-3x development time) and ongoing maintenance burden. A charting library is the standard approach for this class of data visualization.

## Why recharts

| Criterion | recharts | Assessment |
|-----------|----------|------------|
| React-native API | ✅ Declarative React components | Works naturally with React 18 state/props |
| TypeScript support | ✅ Built-in type definitions | No `@types/` package needed |
| Chart types needed | ✅ LineChart, BarChart, ScatterChart, ReferenceDot | All 5 required chart types covered |
| Performance (1200 points) | ✅ SVG-based, handles 10k+ points | Well within comfort zone |
| Responsive design | ✅ `ResponsiveContainer` built-in | Mobile layouts with zero extra work |
| Customization | ✅ Composable — individual axis, tooltip, legend control | Inverted Y-axis, custom colors supported |
| Bundle size | ✅ ~150KB gzipped, tree-shakeable | Reasonable for the functionality provided |
| Maintenance | ✅ Active development, 22k+ GitHub stars | Well-maintained, large community |
| Learning curve | ✅ Declarative API, extensive docs | Low barrier for new contributors |

## Alternatives Considered

### chart.js + react-chartjs-2
- **Rejected because**: Canvas-based (harder to style with CSS/design system), imperative API wrapped in React adapters, less idiomatic React patterns, larger effective bundle when including the wrapper
- **Strengths**: Slightly better raw performance for very large datasets (>100k points — not our case)

### d3 (directly)
- **Rejected because**: Extremely low-level, requires manual DOM manipulation, not React-native (imperative approach conflicts with React reconciliation), estimated 3-4x more code for equivalent functionality
- **Strengths**: Maximum flexibility, smallest possible bundle (only import what you use)

### visx (Airbnb)
- **Rejected because**: Also low-level d3 primitives exposed as React components — more code for standard chart types, smaller community, less documentation for common patterns
- **Strengths**: More granular control, smaller individual component sizes

### nivo
- **Rejected because**: Heavier overall bundle (~300KB), opinionated theming system that conflicts with our existing Tailwind/design system approach, limited customization for non-standard chart types (swimlane)
- **Strengths**: Beautiful defaults, good animation support

### victory (Formidable)
- **Rejected because**: Less active maintenance (slower release cadence), similar API surface to recharts but smaller community for troubleshooting
- **Strengths**: Good documentation, clean API

## Risk Assessment

- **Dependency risk**: LOW — MIT license, large community (22k stars), actively maintained, no known security issues
- **Bundle risk**: MODERATE — 150KB gzipped; mitigated by tree-shaking (only imported chart types are bundled)
- **Lock-in risk**: LOW — chart components are isolated in `frontend/src/features/analysis/`; swapping library would only affect those files
- **Breaking changes risk**: LOW — recharts follows semver; pin to ^2.12 for stability

## Compliance

- ✅ Written justification provided (this document)
- ✅ Owner assigned (@karlkuhnhausen)
- ✅ License compatible (MIT)
- ✅ Will be added to the project dependency ledger upon merge
