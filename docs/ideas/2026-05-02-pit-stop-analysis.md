# Pit Stop Analysis Over the Season

**Status**: brainstorm
**Captured**: 2026-05-02

## Problem / motivation
Pit crews are a major performance differentiator that's largely invisible in the
current dashboard. Teams obsess over pit stop KPIs (stationary time, total loss
time, consistency) because a tenth here compounds across a season. A view that
surfaces team- and crew-level pit performance over the season would be a
distinctive angle vs. generic timing sites.

## Sketch of the idea
A "Pit Wall" page (or section under a team/driver detail view) showing:

- **Per-stop detail**: session, lap, driver, stationary time, total pit-lane
  loss time, tyre change, any penalty/unsafe-release flag.
- **Per-team season aggregates**: median and p90 stationary time, fastest stop,
  number of stops, consistency (stdev), trend line across rounds.
- **Per-driver-crew aggregates**: same KPIs but scoped to the crew that
  services that car (in F1 each driver has a dedicated crew within a team).
- **Leaderboards**: fastest single stop of the season, most consistent crew,
  most improved crew round-over-round.
- **Round comparison**: side-by-side bar chart of all teams' median stop time
  for a selected round.

KPIs to track (standard pit-crew measures):
- Stationary time (wheels-stopped to wheels-rolling) — primary KPI
- Total pit-lane loss time (pit entry to pit exit delta vs. green-lap time)
- Consistency / variance across stops in a session and across the season
- Tyre-change-only vs. front-wing-change vs. damage stops (categorize)
- Unsafe release / penalty incidents

## Open questions
- **Data source**: does OpenF1 expose pit stop stationary time directly, or do
  we have to derive it from `pit` + lap/timing data? Need to spike the OpenF1
  schema before committing to a spec.
- **Crew vs. team attribution**: how do we model "the #44 crew" vs. "Mercedes"?
  Probably a `crew` concept keyed by (team, car_number, season) since crews
  are dedicated per car within a team.
- **Damage / front-wing stops**: how to classify so they don't pollute the
  "fastest stop" leaderboard. May need a duration threshold or a tyre-changed
  signal.
- **Caching/refresh**: per Principle III, all OpenF1 data lands in Cosmos
  first. Pit stops are immutable once a session ends, so a one-shot ingest
  per completed session is fine — no live refresh needed for v1.
- **UI placement**: standalone "Pit Wall" page vs. tab under existing
  Standings/Round Detail. Probably standalone.

## Why not now
- Feature 004 (design system) just wrapped; want to let it settle.
- Need an OpenF1 data spike before I can scope this — risk of discovering
  pit stationary time isn't directly available and having to scrap the design.
- No urgent user pull yet; this is a "would be cool" not a gap.

## Promotion criteria (when to turn into a real spec)
- OpenF1 data spike confirms which KPIs are computable.
- At least one round of fresh data captured in Cosmos to validate aggregates.
- Decision made on crew-vs-team data model.

## Links
- OpenF1 docs: https://openf1.org/
- Constitution Principle III (data residency): `.specify/memory/constitution.md`
