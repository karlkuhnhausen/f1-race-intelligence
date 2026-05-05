# Data Model: Stable Identity Migration

## Entities

### MeetingIndex (computed, in-memory)

| Field | Type | Description |
|-------|------|-------------|
| ByRound | map[int]MeetingIndexEntry | Display round → entry lookup |
| ByMeetingKey | map[int]int | Meeting key → display round lookup |
| Entries | []MeetingIndexEntry | Ordered list of active meetings |

**Source**: Built by `domain.BuildMeetingIndex()` from calendar container data.  
**Lifecycle**: Computed per-request (stateless). Not stored.

### MeetingIndexEntry

| Field | Type | Description |
|-------|------|-------------|
| DisplayRound | int | 1-indexed sequential round number |
| MeetingKey | int | OpenF1 immutable meeting identifier |
| RaceName | string | Race weekend name |
| StartDatetimeUTC | time.Time | Meeting start date |

### Session (enriched — existing `storage.Session`)

| Field | Type | JSON | Migration Status |
|-------|------|------|-----------------|
| MeetingKey | int | `meeting_key` | ✅ Field exists on type. New docs populated at ingestion. Backfill needed for pre-migration docs where value = 0. |
| SessionKey | int | `session_key` | ✅ Already populated on all session docs (set during initial ingestion). |

### SessionResult (enriched — existing `storage.SessionResult`)

| Field | Type | JSON | Migration Status |
|-------|------|------|-----------------|
| MeetingKey | int | `meeting_key` | ✅ Field exists on type. New docs populated at ingestion. Backfill needed for pre-migration docs where value = 0. |
| SessionKey | int | `session_key` | ✅ Already populated on all session result docs. |

### Analysis Documents (enriched — all `storage.SessionAnalysis*` types)

Applies to: `SessionAnalysisPosition`, `SessionAnalysisInterval`, `SessionAnalysisStint`, `SessionAnalysisPit`, `SessionAnalysisOvertake`

| Field | Type | JSON | Migration Status |
|-------|------|------|-----------------|
| MeetingKey | int | `meeting_key` | ✅ Field exists on type. New docs populated at ingestion. Backfill needed for pre-migration docs where value = 0. |
| SessionKey | int | `session_key` | ✅ Field exists on type. New docs populated at ingestion. Backfill needed for pre-migration docs where value = 0. |

## Relationships

```
Calendar Container (RaceMeeting)
  └─ meeting_key (immutable anchor)
       │
       ├── Sessions Container (Session documents)
       │     └─ meeting_key links back to RaceMeeting
       │     └─ session_key is the session-level anchor
       │
       ├── Sessions Container (SessionResult documents)
       │     └─ meeting_key links back to RaceMeeting
       │     └─ session_key links to parent Session
       │
       └── Sessions Container (Analysis documents)
             └─ meeting_key links back to RaceMeeting
             └─ session_key links to parent Session
```

## Query Patterns

### Primary path (meeting_key available)

```
Round detail page request:
  1. API receives (season, round)
  2. Build MeetingIndex from calendar data
  3. Resolve: meeting_key = MeetingIndex.MeetingKeyForRound(round)
  4. Query: GetSessionsByMeetingKey(season, meeting_key)
  5. Query: GetSessionResultsByMeetingKey(season, meeting_key)
```

### Fallback path (pre-migration documents without meeting_key)

```
If meeting_key query returns empty:
  1. Fall back to: GetSessionsByRound(season, round)
  2. Fall back to: GetSessionResultsByRound(season, round)
```

## Validation Rules

- `meeting_key` MUST be > 0 for stamped documents (0 = not yet stamped)
- `session_key` MUST be > 0 for stamped analysis documents
- MeetingIndex excludes cancelled meetings (`IsCancelled = true`)
- MeetingIndex excludes testing sessions (name contains "testing"/"pre-season")
- Backfill skips documents where meeting_key cannot be resolved (logs warning)
- Backfill is idempotent: documents with meeting_key > 0 are skipped

## State Transitions

```
Document lifecycle:
  [Pre-migration] meeting_key = 0, session_key = 0
       │
       ▼ (backfill --stamp-meeting-keys)
       │
  [Stamped] meeting_key = <resolved>, session_key = <resolved>
       │
       (no further transitions — fields are immutable once set)
```
