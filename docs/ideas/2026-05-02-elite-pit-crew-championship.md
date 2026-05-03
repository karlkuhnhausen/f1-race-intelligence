# Elite Pit Crew Championship

**Status**: brainstorm
**Captured**: 2026-05-02
**Related**: [2026-05-02-pit-stop-analysis.md](2026-05-02-pit-stop-analysis.md) (depends on)

## Problem / motivation
Pit stop performance is a season-long story, not a per-race one. A running
"Pit Crew Championship" leaderboard — updated after every round and crowned
at season end — turns the raw pit stop data into a narrative the user comes
back for week after week. It's also a natural complement to the Drivers' and
Constructors' Championships that already exist in F1.

## Sketch of the idea
- **Running leaderboard**: ranks all 20 crews (one per car) by a composite
  pit-crew score, updated after each completed race round.
- **Round-by-round movement**: arrows / deltas showing how each crew moved
  vs. the previous round (think: standings page but for crews).
- **End-of-season crowning**: when the final round of the season is marked
  complete, a "Champion Pit Crew" is locked in and displayed prominently.
  Historical champions persist across seasons.
- **Crew profile page**: per-crew detail showing their stops across the
  season, their best/worst rounds, and where they rank on each KPI.

## Scoring model (to be designed — this is the hard part)
Candidate inputs, all season-to-date:
- Median stationary time (lower is better) — primary signal
- Consistency: stdev or p90 - p50 spread
- Best single stop (tiebreaker / flair)
- Penalty / unsafe-release count (negative)
- Number of stops completed (volume / sample-size weight)

Open scoring questions:
- **Points-per-round vs. season-aggregate**: F1 itself uses points-per-round
  (you can't lose points). A points model is more intuitive ("crew X scored
  18 points at Monaco") but requires defining a per-round ranking first.
  Aggregate stats are simpler but feel less like a championship.
- **Damage-stop handling**: front-wing changes and damage repairs must be
  excluded or categorized separately, otherwise a crew gets punished for
  their driver crashing.
- **Sample size fairness**: a crew with 2 stops at a wet race shouldn't be
  ranked against one with 4 dry stops. Weighting? Minimum-stops gate?
- **Tie-breakers**: if two crews tie on score, what wins? Best single stop?
  Fewer penalties? Head-to-head round wins?

## Open questions
- **Season boundary detection**: how do we know "the season is over" so we
  can crown a champion? Last round in the calendar marked Complete? An
  explicit admin/cron flag? This needs a clear trigger.
- **Historical champions**: where do crowned champions live? A new Cosmos
  container `pit_crew_champions` keyed by (season, crew_id)? Materialized
  once at season end so the result is stable even if data is reprocessed.
- **Mid-season crew changes**: rare but they happen (driver swaps, crew
  member injuries). v1 probably ignores this and treats "crew" as
  (team, car_number, season).
- **Refresh model**: per Principle III, recompute only on new completed-
  session data. The leaderboard is a derived view of pit stop data — fine
  to recompute on every round-completion event rather than on every read.

## Why not now
- **Hard dependency** on the pit stop analysis idea: we can't score crews
  until we have the underlying KPI data ingested and validated.
- **Scoring model needs design work**: this is the kind of decision worth
  prototyping on already-collected data before committing to a spec, so
  ship pit stop analysis first and use real numbers to inform the model.
- Crowning logic only matters at season end (Dec 2026), so there's no
  pressure to ship before mid-season.

## Promotion criteria (when to turn into a real spec)
- Pit stop analysis (predecessor idea) is shipped or near-shipped.
- A scoring model has been prototyped against at least 3 rounds of real
  Cosmos data and produces rankings that pass the smell test.
- Decision made on points-per-round vs. season-aggregate.
- Decision made on season-end trigger and historical-champion storage.

## Links
- Predecessor idea: [2026-05-02-pit-stop-analysis.md](2026-05-02-pit-stop-analysis.md)
- Constitution Principle III (data residency): `.specify/memory/constitution.md`
