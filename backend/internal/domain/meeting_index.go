package domain

import (
	"sort"
	"strings"
	"time"
)

// MeetingIndexEntry holds the mapping between a display round number and
// the underlying OpenF1 meeting key for one race meeting.
type MeetingIndexEntry struct {
	DisplayRound     int
	MeetingKey       int
	RaceName         string
	StartDatetimeUTC time.Time
}

// MeetingIndex provides bidirectional lookup between display round numbers
// and OpenF1 meeting keys. It is computed from the current set of non-cancelled,
// non-testing meetings sorted by date.
type MeetingIndex struct {
	// ByRound maps display_round → meeting_key.
	ByRound map[int]MeetingIndexEntry
	// ByMeetingKey maps meeting_key → display_round.
	ByMeetingKey map[int]int
	// Entries is the ordered list of all active meetings.
	Entries []MeetingIndexEntry
}

// MeetingForIndex is the minimal data needed to build a MeetingIndex.
type MeetingForIndex struct {
	MeetingKey       int
	RaceName         string
	StartDatetimeUTC time.Time
	IsCancelled      bool
}

// BuildMeetingIndex constructs a MeetingIndex from a list of meetings.
// Cancelled and testing meetings are excluded from round assignment.
// The resulting display rounds are 1-indexed and sequential by start date.
func BuildMeetingIndex(meetings []MeetingForIndex) *MeetingIndex {
	// Filter to active (non-cancelled, non-testing) meetings.
	active := make([]MeetingForIndex, 0, len(meetings))
	for _, m := range meetings {
		if m.IsCancelled {
			continue
		}
		if isTestingName(m.RaceName) {
			continue
		}
		active = append(active, m)
	}

	// Sort by start date (stable for deterministic ordering).
	sort.SliceStable(active, func(i, j int) bool {
		return active[i].StartDatetimeUTC.Before(active[j].StartDatetimeUTC)
	})

	idx := &MeetingIndex{
		ByRound:      make(map[int]MeetingIndexEntry, len(active)),
		ByMeetingKey: make(map[int]int, len(active)),
		Entries:      make([]MeetingIndexEntry, 0, len(active)),
	}

	for i, m := range active {
		round := i + 1
		entry := MeetingIndexEntry{
			DisplayRound:     round,
			MeetingKey:       m.MeetingKey,
			RaceName:         m.RaceName,
			StartDatetimeUTC: m.StartDatetimeUTC,
		}
		idx.ByRound[round] = entry
		idx.ByMeetingKey[m.MeetingKey] = round
		idx.Entries = append(idx.Entries, entry)
	}

	return idx
}

// RoundForMeetingKey returns the display round for a given meeting key,
// or 0 if the meeting is not in the index (cancelled/testing/unknown).
func (idx *MeetingIndex) RoundForMeetingKey(meetingKey int) int {
	return idx.ByMeetingKey[meetingKey]
}

// MeetingKeyForRound returns the meeting key for a given display round,
// or 0 if the round is out of range.
func (idx *MeetingIndex) MeetingKeyForRound(round int) int {
	entry, ok := idx.ByRound[round]
	if !ok {
		return 0
	}
	return entry.MeetingKey
}

// TotalRounds returns the number of active rounds in the index.
func (idx *MeetingIndex) TotalRounds() int {
	return len(idx.Entries)
}

// isTestingName reports whether a meeting/race name refers to pre-season testing.
func isTestingName(raceName string) bool {
	n := strings.ToLower(raceName)
	return strings.Contains(n, "pre-season") ||
		strings.Contains(n, "pre season") ||
		strings.Contains(n, "preseason") ||
		strings.Contains(n, "testing")
}
