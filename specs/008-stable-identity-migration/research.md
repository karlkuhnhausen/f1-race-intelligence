# Research: Stable Identity Migration

## R1: How to resolve round → meeting_key for existing documents

**Decision**: Use `domain.BuildMeetingIndex` fed by calendar container data (`GetMeetingsBySeason`).

**Rationale**: The MeetingIndex is already the authoritative resolver used by the rounds and analysis APIs (see `rounds/service.go:319` and `analysis/service.go:150-168`). It filters out cancelled and testing meetings, then assigns sequential 1-indexed display rounds by start date. This gives us a deterministic mapping from any display round to its meeting_key.

**Alternatives considered**:
- Direct OpenF1 API calls to resolve meeting keys: Rejected — violates constitution (data residency), adds rate-limit risk, and calendar data is already cached in Cosmos.
- Hardcoded round→meeting_key lookup table: Rejected — fragile, requires manual updates each season.

## R2: How to find the session_key for analysis documents

**Decision**: For each analysis document (which has `season`, `round`, `session_type`), query the sessions container for the matching session document (by round or meeting_key) to get the `session_key`.

**Rationale**: Session documents already have `session_key` populated from OpenF1 ingestion. Analysis documents reference sessions by round+session_type, so we can look up the corresponding session document to get the session_key.

**Alternatives considered**:
- Store session_key directly during analysis ingestion: Already done for new documents (Phases 1-4). The backfill is only for pre-migration documents.
- OpenF1 API query: Rejected — unnecessary external call when data is already in Cosmos sessions container.

## R3: Idempotency strategy for the backfill

**Decision**: Check if `meeting_key > 0` on each document before updating. If already populated, skip. Use `UpsertSession` / `UpsertSessionResult` / upsert analysis methods which are naturally idempotent (same ID + partition key = replace).

**Rationale**: The Cosmos DB upsert semantic ensures no duplicate documents. Checking meeting_key > 0 before update avoids unnecessary writes and RU consumption. This matches the existing pattern in `backfillAnalysis` which checks `HasAnalysisData` before fetching.

**Alternatives considered**:
- Track stamped document IDs in a separate collection: Rejected — over-engineering for ~500 documents.
- Delete and re-create documents: Rejected — destructive and unnecessary.

## R4: Which document types need stamping

**Decision**: Three categories of documents need `meeting_key` populated:
1. **Session documents** (`type: "session"`) — need `meeting_key` field set
2. **Session result documents** (`type: "session_result"`) — need `meeting_key` field set
3. **Analysis documents** (`type: "analysis_position"`, `"analysis_interval"`, `"analysis_stint"`, `"analysis_pit"`, `"analysis_overtake"`) — need both `meeting_key` and `session_key` fields set

**Rationale**: These are all the document types that the rounds and analysis APIs query by meeting_key. The storage types already have the fields defined (added in Phase 2).

**Alternatives considered**:
- Stamp only session results (not session documents): Rejected — `GetSessionsByMeetingKey` also queries sessions.
- Skip analysis documents: Rejected — `GetSessionAnalysisByMeetingKey` needs meeting_key on analysis docs.

## R5: Backfill execution strategy

**Decision**: Process one season at a time (specified via `--season` flag, matching existing CLI pattern). Iterate through all documents in the sessions container, resolve meeting_key via MeetingIndex, update in place. No external API calls needed.

**Rationale**: The existing backfill CLI already processes by season with rate-limit controls and dry-run support. The same pattern applies here. Since we only query Cosmos DB (no OpenF1 calls), the rate-limit delay can be minimal (just to avoid hot partition throttling).

**Alternatives considered**:
- Process all seasons in one run: Possible but the `--season` flag pattern is established and allows targeted re-runs.
- Parallel processing: Rejected — documents are in the same partition (season), so parallelism won't help with Cosmos RU throughput.
