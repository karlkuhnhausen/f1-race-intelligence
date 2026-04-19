# Research: Race Calendar and Championship Standings

## Decision 1: Poll cadence and data freshness policy

- Decision: Poll OpenF1 meetings every 5 minutes and refresh standings every 5 minutes in backend scheduled workers.
- Rationale: Meets FR-001 and SC-004 freshness requirements without excessive request churn.
- Alternatives considered:
  - 1-minute polling: rejected due to higher cost and little user-visible benefit.
  - 15-minute polling: rejected because it weakens freshness SLA during race weekends.

## Decision 2: Cancellation handling model

- Decision: Apply product override rules in backend for Bahrain R4 and Saudi Arabia R5 with `status=cancelled`, `is_cancelled=true`, and `cancelled_label`.
- Rationale: Requirement is explicit and must be deterministic regardless of upstream feed inconsistencies.
- Alternatives considered:
  - Depend entirely on upstream status: rejected because upstream may not expose immediate geopolitical cancellation semantics.

## Decision 3: Next-race countdown source of truth

- Decision: Compute `next_round` and `countdown_target_utc` on backend and expose to frontend via calendar response.
- Rationale: Enforces tier boundaries and avoids duplicate browser logic.
- Alternatives considered:
  - Frontend-side computation: rejected due to architecture rule and divergence risk.

## Decision 4: Cosmos DB schema and partitioning

- Decision: Store normalized race and standings documents keyed by season and entity type with partition key `/season`.
- Rationale: Read patterns are season-centric (2026) and this keeps queries simple/cost-effective in serverless mode.
- Alternatives considered:
  - Partition by round/team: rejected because most requests are full-season table reads.

## Decision 5: Upstream fault tolerance

- Decision: Serve last successful cached snapshot with `data_as_of_utc` and health metrics when upstream calls fail.
- Rationale: Preserves user experience and supports transparent staleness reporting.
- Alternatives considered:
  - Fail closed on upstream errors: rejected because dashboard would become unavailable unnecessarily.

## Decision 6: Minimal dependency policy for implementation

- Decision: Prefer standard library scheduling/time handling where practical; add third-party libs only with explicit owner and justification.
- Rationale: Constitution mandates dependency discipline and maintainability.
- Alternatives considered:
  - Broad utility frameworks: rejected due to increased transitive risk and governance overhead.

## Open questions (to resolve during implementation)

- Confirm Hyprace standings endpoint contract and authentication requirements for production use.
- Confirm whether countdown target is race session start (race) or meeting start; default is race start UTC.