# Day 14: Three Bugs, One Cause — When Stale Cached State Becomes a UX Smell

*Posted April 29, 2026 · Karl Kuhnhausen*

---

[Day 13](day-13-allowlist-guard.md) closed an infrastructure hole — an empty CI secret that could unlatch the AKS API server. That fix was a single PR, three guards, satisfying. This post is the opposite shape: three small visible bugs, all on the same dashboard, that turned out to share one root cause. The kind of session where the debugging is cheap and the *spotting* is what matters.

The bugs:

1. **The Australian, Chinese, and Japanese Grands Prix all said "Scheduled"** on the calendar, two months after they were over.
2. **The round detail page listed sessions in chronological order even for finished race weekends** — so on a completed round, FP1 was at the top and the race I actually wanted to look at was buried at the bottom.
3. **The session results tables were squished.** Position, number, driver, team, gap, points all crowded into the left half of a card, with most of the right half empty.

PR [#27](https://github.com/karlkuhnhausen/f1-race-intelligence/pull/27). Four commits, +604 / −582. The third bug ate most of the diff because consolidating it meant deleting three near-duplicate components.

---

## Bug 1: "It's been a month and the calendar still says scheduled"

The Japanese Grand Prix happened on April 12. I was looking at the dashboard on April 28 and the badge in the status column still read **SCHEDULED**.

I'd seen this shape of bug before. On [Day 12](day-12-status-badge-bug.md) I'd argued that session status is a *derived* value — a function of the current time and the session's scheduled times — and that storing it in Cosmos is a denormalization. A cache. And like all caches, it needs an authoritative source to fall back on. That fix went into the rounds API: it computes session status at read time from dates rather than trusting the cached field.

The same principle applied here. I'd just never extended it to the meeting layer. The ingest path hard-coded `Status: "scheduled"` for every meeting it wrote to Cosmos:

```go
// backend/internal/ingest/meeting_transform.go
m := storage.RaceMeeting{
    // ...
    Status:           "scheduled",
    IsCancelled:      false,
    Source:           "openf1",
    DataAsOfUTC:      now,
    SourceHash:       fmt.Sprintf("%d", r.MeetingKey),
}
```

The calendar service read it back and passed it straight to the response. Nothing flipped it to `completed` after the race ended. That field was a single-write cache with no invalidation.

The fix is the same shape as Day 12. Derive at read time. The calendar service already has a clock injected for testing, so the derivation slots in next to the cancellation override:

```go
// backend/internal/api/calendar/service.go
if !m.IsCancelled {
    m.Status = deriveMeetingStatus(now, m.StartDatetimeUTC, m.EndDatetimeUTC)
}

// ...

func deriveMeetingStatus(now, start, end time.Time) string {
    if start.IsZero() {
        return string(domain.StatusUnknown)
    }
    if start.After(now) {
        return string(domain.StatusScheduled)
    }
    if end.IsZero() || end.After(now) {
        return string(domain.StatusScheduled)
    }
    return string(domain.StatusCompleted)
}
```

Cancellation override (Bahrain and Saudi Arabia for 2026, removed by Liberty) still wins — it's set above this block. Everything else follows from the calendar plus the wall clock.

Result on the live site:

![2026 calendar — past races now show COMPLETED, future races SCHEDULED, Bahrain and Saudi Arabia struck through as CANCELLED](images/calendar-desktop.png)

The pre-season testing weekends, AUS, CHN, and JPN are all green "Completed." Miami onward stays "Scheduled." Bahrain and Saudi Arabia, the cancelled rounds, render with the line-through and the red badge — the override path is still in front of the derivation path.

I want to flag one thing about the smallness of this fix. The data was right. The schema was right. The frontend rendering was right. The bug was *one missing branch* — a piece of derivation logic that already existed for sessions and just hadn't been ported to the meeting layer. The cost of writing it the second time was about ten lines of Go and four unit tests. The cost of *not noticing* the asymmetry between the two layers was several weeks of a calendar page that lied to me about what had already happened.

---

## Bug 2: The race I want to read about is at the bottom of the page

This one was a UX bug, not a correctness bug. The frontend had always sorted sessions on the round detail page by a hand-rolled session-type rank:

```tsx
const sessionOrder: Record<string, number> = {
    practice1: 1, practice2: 2, practice3: 3,
    sprint_qualifying: 4, sprint: 5,
    qualifying: 6, race: 7,
};

const sortedSessions = [...data.sessions].sort(
    (a, b) =>
        (sessionOrder[a.session_type] ?? 99) -
        (sessionOrder[b.session_type] ?? 99)
);
```

That ordering is correct *before* the round happens. You're looking ahead to the weekend; you want FP1 first because FP1 happens first.

It is exactly wrong *after* the round happens. You came to the page to look at the race result. You don't want to scroll past three practice sessions and qualifying to get to it.

The fix is a one-line conditional:

```tsx
const allCompleted =
    data.sessions.length > 0 &&
    data.sessions.every((s) => s.status === 'completed');

const sortedSessions = [...data.sessions].sort((a, b) => {
    if (allCompleted) {
        return (
            new Date(b.date_start_utc).getTime() -
            new Date(a.date_start_utc).getTime()
        );
    }
    return (
        (sessionOrder[a.session_type] ?? 99) -
        (sessionOrder[b.session_type] ?? 99)
    );
});
```

Sorting completed weekends by `date_start_utc` descending puts the race on top, qualifying below, then practice 3 / 2 / 1. It also handles sprint weekends naturally — a Sprint Qualifying / Sprint / Qualifying / Race weekend produces the right order without me having to keep the rank table in sync with how F1 schedules its sprint formats. Date-based ordering is durable; rank-table ordering drifts.

This is the kind of bug that's ten times harder to spot than to fix. The page worked. Every test passed. The data was right. The rendering was right. It was just *the wrong information at the top of the page for the most common time you'd look at it.* You can stare past that for months.

---

## Bug 3: Three near-duplicate tables, all of them ugly

The squished result tables turned out to be the largest part of the diff and the most interesting cleanup.

Each session card on the round detail page rendered through one of three components — `RaceResults.tsx`, `QualifyingResults.tsx`, `PracticeResults.tsx` — chosen by a switch on `session_type`. They had each been written separately, at different times, with different assumptions, and they had drifted apart:

- The race component had a "Not Classified" divider for DNFs. Quali and practice did not.
- The race component highlighted the fastest-lap row with a tint. Quali silently dropped that highlight even when the fastest-lap flag was present.
- All three had a `<table className="results-table ...">` with no Tailwind sizing — every column auto-fit, which is exactly what produced the squished look.
- All three had their own bespoke lap-time formatter. The shared `LapTimeDisplay` design-system component existed but only one of them used it.

That kind of trio is a drift hazard. The bug I'd come to fix lived inside the tables, but the *next* bug lives in the seam between the three components. So the right move was to consolidate them onto the table component that already existed (`SessionResultsTable.tsx` — written during the Feature 4 design pass but never actually wired in) and then delete the legacy three.

The redesigned table:

![Japanese Grand Prix round detail — race on top with podium border accents on positions 1, 2, and 3, team color swatches next to team names, evenly distributed columns, "Not Classified" divider for the DNFs](images/round-detail-desktop.png)

A few specific moves are worth calling out:

**Fixed column widths via `<colgroup>`.** Tailwind's `table-fixed` plus a `<colgroup>` of explicit widths (`w-14` for Pos and #, `w-[22%]` for Team, `w-28` for Gap, etc.) means the columns lay out predictably regardless of content. The driver name column gets the leftover space. No more squish.

**Right-aligned `font-mono tabular-nums` on numeric columns.** This is the single biggest readability win. Lap times like `1:30.612` and `1:30.747` line up digit-for-digit so the eye can compare them at a glance. The `tabular-nums` Tailwind utility flips on `font-variant-numeric: tabular-nums` — the variable-width digits in Inter become equal-width just for the numeric cells. Bringing the existing `LapTimeDisplay` design-system component into all three session types means quali no longer drops the fastest-lap highlight.

**Team color swatch as a "badge" placeholder.** The Team cell renders a 1px-wide colored bar next to the team name, sourced from `getTeamColor()` in the design system. You can see Red Bull blue, Ferrari red, McLaren orange, Mercedes teal, etc. lining up on the left edge of every team cell:

```tsx
<td className="px-3 py-3">
  <div className="flex items-center gap-2 min-w-0">
    <span
      aria-hidden
      className="inline-block h-4 w-1 shrink-0 rounded-sm"
      style={{ backgroundColor: getTeamColor(r.team_name) }}
    />
    <span className="truncate text-muted-foreground">{r.team_name}</span>
  </div>
</td>
```

I'd love to put real team logos there. F1 team marks are licensed and I haven't gone through the rights work, so the swatch is a placeholder for an `<img>` that will slot in cleanly when the assets land. The rest of the layout is already designed to host it.

**Podium border accents on race sessions.** Position 1 gets a gold left-border, P2 silver, P3 bronze. Practice and qualifying don't get the accents — those sessions don't have a podium in the canonical sense, and the visual would be misleading. One conditional, race-only:

```tsx
function podiumBorder(position: number, isRace: boolean): string {
    if (!isRace) return '';
    switch (position) {
        case 1: return 'border-l-2 border-l-amber-400';
        case 2: return 'border-l-2 border-l-zinc-300';
        case 3: return 'border-l-2 border-l-amber-700';
        default: return '';
    }
}
```

**Absorbed the "Not Classified" divider into the shared component.** The race component used to render a `<tr>` with a `colSpan` divider above DNF rows, then the DNFs themselves dimmed. That UX is good — it's the same convention F1's official timing pages use — and it shouldn't have been a race-only feature lost the moment somebody decides to consolidate. The new shared table partitions race results into classified and non-classified, renders the divider, and dims the non-classified rows with `opacity-60`. In the screenshot above you can see Doohan and Hamilton below the divider.

After all that, the three legacy components could come out:

```
delete frontend/src/features/rounds/PracticeResults.tsx
delete frontend/src/features/rounds/QualifyingResults.tsx
delete frontend/src/features/rounds/RaceResults.tsx
delete frontend/tests/rounds/PracticeResults.test.tsx
delete frontend/tests/rounds/QualifyingResults.test.tsx
delete frontend/tests/rounds/RaceResults.test.tsx
```

Net: −499 lines on that commit, after the new shared component and its tests already landed in the previous one.

---

## What unifies these three bugs

Reading back over the diff, all three bugs are the same shape. They're all places where we kept a piece of *derived* state — something that should be a function of inputs we already have — as if it were a *fact*.

- Bug 1: meeting status was a single-write cache of `(start, end, now)`.
- Bug 2: session order on a completed weekend was a function of `date_start_utc`, but we computed it from a hand-rolled rank table.
- Bug 3: the result table layout was three frozen-in-time copies of what should have been one declarative component.

The fixes all collapse derived-state-as-fact into actual derivation. The status comes from the dates and the clock. The order comes from the dates. The table comes from one component with conditionals on session type.

There's a longer essay in here about why this pattern shows up so much in dashboard code specifically — I think it's because dashboards are mostly "render the world's current state," and any time you cache the world's state you're one render cycle away from it being stale. But the practical takeaway for this project is shorter: when something on a dashboard looks wrong and the data layer is right, look for a place where someone wrote down the answer to a question that should still be a question.

---

## What's live

- **PR [#27](https://github.com/karlkuhnhausen/f1-race-intelligence/pull/27)** ([Calendar status fix + rounds page polish](https://github.com/karlkuhnhausen/f1-race-intelligence/pull/27)) — 4 commits, +604 / −582.
- **Calendar page**: past races correctly show "Completed," cancellations still strike through with the red badge.
- **Round detail page**: completed weekends list sessions newest-first; results tables have fixed columns, right-aligned tabular numerals, team color swatches, podium accents on race, "Not Classified" divider for DNFs.
- **Tests**: 57 frontend tests passing (was 69; net −12 from deleting six legacy test files and adding one new five-case `SessionResultsTable.test.tsx`). All 19 backend test packages green, including three new calendar-service cases.
- **Cleanup**: three legacy result components deleted. One canonical `SessionResultsTable` going forward.

A small follow-up surfaced on the way: pre-season testing currently shows up as round 1 and round 2 on the calendar, which pushes Australia to round 3 instead of round 1. That's an OpenF1 ingest-numbering issue and out of scope for this PR. Filed for next session.

---

[← Day 13: An Empty String Is a Wildcard — Closing the AKS Allowlist Hole](day-13-allowlist-guard.md) · [Day 15: Feature 4 — What a "Pure Frontend" Feature Actually Cost →](day-15-feature-004-wrap.md)
