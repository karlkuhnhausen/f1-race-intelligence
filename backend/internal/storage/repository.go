// Package storage defines the repository interfaces for the F1 Race Intelligence backend.
package storage

import (
	"context"
	"time"
)

// RaceMeeting represents one F1 season round stored in Cosmos DB.
type RaceMeeting struct {
	ID               string    `json:"id"`
	Season           int       `json:"season"`
	Round            int       `json:"round"`
	MeetingKey       int       `json:"meeting_key"`
	RaceName         string    `json:"race_name"`
	CircuitName      string    `json:"circuit_name"`
	CountryName      string    `json:"country_name"`
	StartDatetimeUTC time.Time `json:"start_datetime_utc"`
	EndDatetimeUTC   time.Time `json:"end_datetime_utc"`
	Status           string    `json:"status"`
	IsCancelled      bool      `json:"is_cancelled"`
	CancelledLabel   string    `json:"cancelled_label,omitempty"`
	CancelledReason  string    `json:"cancelled_reason,omitempty"`
	Source           string    `json:"source"`
	DataAsOfUTC      time.Time `json:"data_as_of_utc"`
	SourceHash       string    `json:"source_hash"`
}

// DriverStandingRow represents one row in the drivers championship.
type DriverStandingRow struct {
	ID          string    `json:"id"`
	Season      int       `json:"season"`
	Position    int       `json:"position"`
	DriverName  string    `json:"driver_name"`
	TeamName    string    `json:"team_name"`
	Points      float64   `json:"points"`
	Wins        int       `json:"wins"`
	DataAsOfUTC time.Time `json:"data_as_of_utc"`
	Source      string    `json:"source"`
}

// ConstructorStandingRow represents one row in the constructors championship.
type ConstructorStandingRow struct {
	ID          string    `json:"id"`
	Season      int       `json:"season"`
	Position    int       `json:"position"`
	TeamName    string    `json:"team_name"`
	Points      float64   `json:"points"`
	DataAsOfUTC time.Time `json:"data_as_of_utc"`
	Source      string    `json:"source"`
}

// CalendarRepository defines read/write operations for race meetings.
type CalendarRepository interface {
	UpsertMeeting(ctx context.Context, m RaceMeeting) error
	GetMeetingsBySeason(ctx context.Context, season int) ([]RaceMeeting, error)
	GetMeetingByID(ctx context.Context, season int, id string) (*RaceMeeting, error)
	GetMeetingByMeetingKey(ctx context.Context, season, meetingKey int) (*RaceMeeting, error)
	DeleteMeeting(ctx context.Context, season int, id string) error
}

// StandingsRepository defines read/write operations for championship standings.
type StandingsRepository interface {
	UpsertDriverStandings(ctx context.Context, rows []DriverStandingRow) error
	GetDriverStandings(ctx context.Context, season int) ([]DriverStandingRow, error)
	UpsertConstructorStandings(ctx context.Context, rows []ConstructorStandingRow) error
	GetConstructorStandings(ctx context.Context, season int) ([]ConstructorStandingRow, error)
}

// RaceControlSummary is the aggregated race-control state for a single session.
// Stored as a nested object within the session document in Cosmos DB.
type RaceControlSummary struct {
	RedFlagCount   int            `json:"red_flag_count"`
	SafetyCarCount int            `json:"safety_car_count"`
	VSCCount       int            `json:"vsc_count"`
	NotableEvents  []NotableEvent `json:"notable_events"`
	FetchedAtUTC   time.Time      `json:"fetched_at_utc"`
}

// NotableEvent is a single race-control activation within a session.
type NotableEvent struct {
	// EventType is one of: "red_flag", "safety_car", "vsc", "investigation"
	EventType string `json:"event_type"`
	// LapNumber is the lap on which the first (or only) activation occurred.
	// May be 0 for pre-race events.
	LapNumber int `json:"lap_number"`
	// Count is the number of distinct activations of this event type.
	Count int `json:"count"`
}

// Session represents one session within a race weekend stored in Cosmos DB.
type Session struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"` // document type discriminator: "session"
	Season       int       `json:"season"`
	Round        int       `json:"round"`
	MeetingKey   int       `json:"meeting_key"`
	SessionKey   int       `json:"session_key"`
	SessionName  string    `json:"session_name"`
	SessionType  string    `json:"session_type"`
	Status       string    `json:"status"`
	DateStartUTC time.Time `json:"date_start_utc"`
	DateEndUTC   time.Time `json:"date_end_utc"`
	DataAsOfUTC  time.Time `json:"data_as_of_utc"`
	Source       string    `json:"source"`

	// Finalized indicates the session is over and its results/drivers/laps
	// have been fully fetched and cached. Once true, the session poller
	// skips re-fetching from OpenF1.
	Finalized bool `json:"finalized,omitempty"`
	// FinalizedAtUTC is the time the session was first marked finalized.
	FinalizedAtUTC *time.Time `json:"finalized_at_utc,omitempty"`
	// SchemaVersion tracks the cached document layout. If the code's
	// current schema version is newer than the cached value, the
	// finalized flag is treated as stale and the session is re-fetched.
	SchemaVersion int `json:"schema_version,omitempty"`

	// Feature 005: race-control summary populated at finalization and by the
	// backfill tool. Nil for sessions finalized before Feature 005 shipped.
	RaceControlSummary *RaceControlSummary `json:"race_control_summary,omitempty"`
	// Feature 005: fastest lap duration in seconds, derived from the laps
	// fetched at finalization. Nil for sessions finalized before Feature 005.
	FastestLapTimeSeconds *float64 `json:"fastest_lap_time_seconds,omitempty"`
}

// SessionResult represents one driver's result within a session stored in Cosmos DB.
type SessionResult struct {
	ID            string    `json:"id"`
	Type          string    `json:"type"` // document type discriminator: "session_result"
	Season        int       `json:"season"`
	Round         int       `json:"round"`
	MeetingKey    int       `json:"meeting_key"`
	SessionKey    int       `json:"session_key"`
	SessionType   string    `json:"session_type"`
	Position      int       `json:"position"`
	DriverNumber  int       `json:"driver_number"`
	DriverName    string    `json:"driver_name"`
	DriverAcronym string    `json:"driver_acronym"`
	TeamName      string    `json:"team_name"`
	NumberOfLaps  int       `json:"number_of_laps"`
	DataAsOfUTC   time.Time `json:"data_as_of_utc"`
	Source        string    `json:"source"`

	// Race-specific fields
	FinishingStatus *string  `json:"finishing_status,omitempty"`
	RaceTime        *float64 `json:"race_time,omitempty"`
	GapToLeader     *string  `json:"gap_to_leader,omitempty"`
	Points          *float64 `json:"points,omitempty"`
	FastestLap      *bool    `json:"fastest_lap,omitempty"`

	// Qualifying-specific fields
	Q1Time *float64 `json:"q1_time,omitempty"`
	Q2Time *float64 `json:"q2_time,omitempty"`
	Q3Time *float64 `json:"q3_time,omitempty"`

	// Practice-specific fields
	BestLapTime  *float64 `json:"best_lap_time,omitempty"`
	GapToFastest *float64 `json:"gap_to_fastest,omitempty"`
}

// SessionRepository defines read/write operations for sessions and session results.
type SessionRepository interface {
	UpsertSession(ctx context.Context, s Session) error
	UpsertSessionResult(ctx context.Context, r SessionResult) error
	GetSessionsByRound(ctx context.Context, season, round int) ([]Session, error)
	GetSessionResultsByRound(ctx context.Context, season, round int) ([]SessionResult, error)
	// GetSessionsByMeetingKey returns all sessions for the given meeting_key.
	// Used by the API layer after resolving round → meeting_key via MeetingIndex.
	GetSessionsByMeetingKey(ctx context.Context, season, meetingKey int) ([]Session, error)
	// GetSessionResultsByMeetingKey returns all session results for the given meeting_key.
	GetSessionResultsByMeetingKey(ctx context.Context, season, meetingKey int) ([]SessionResult, error)
	// GetSessionResultsBySeason returns every cached SessionResult for the
	// given season across all rounds. Used to compute running championship
	// totals from OpenF1 race + sprint points without depending on a
	// separate standings provider.
	GetSessionResultsBySeason(ctx context.Context, season int) ([]SessionResult, error)
	// GetFinalizedSessionKeys returns the set of session_key values for the
	// season whose cached document has Finalized=true. The poller uses this
	// as a skip-list so it does not re-fetch results/drivers/laps for sessions
	// that already finished and were fully cached.
	GetFinalizedSessionKeys(ctx context.Context, season int) (map[int]int, error)
	// GetFinalizedSessions returns all session documents for the season where
	// finalized=true. Used by the backfill CLI to identify sessions that need
	// race-control summary population.
	GetFinalizedSessions(ctx context.Context, season int) ([]Session, error)
	// DeleteSession removes a session document by its ID.
	DeleteSession(ctx context.Context, season int, id string) error
	// DeleteSessionResultsBySessionType removes all session_result documents
	// for a given season, round, and session_type.
	DeleteSessionResultsBySessionType(ctx context.Context, season, round int, sessionType string) error
}

// --- Analysis data types ---

// SessionAnalysisPosition stores aggregated position data for one driver in one session.
type SessionAnalysisPosition struct {
	ID            string        `json:"id"`     // analysis_position_{round}_{sessiontype}_{drivernum}
	Type          string        `json:"type"`   // "analysis_position"
	Season        int           `json:"season"` // partition key
	Round         int           `json:"round"`
	MeetingKey    int           `json:"meeting_key"`
	SessionKey    int           `json:"session_key"`
	SessionType   string        `json:"session_type"`
	DriverNumber  int           `json:"driver_number"`
	DriverName    string        `json:"driver_name"`
	DriverAcronym string        `json:"driver_acronym"`
	TeamName      string        `json:"team_name"`
	TeamColor     string        `json:"team_colour"` //nolint:misspell // OpenF1 API field name
	Laps          []PositionLap `json:"laps"`
}

// PositionLap is a single lap's position for a driver.
type PositionLap struct {
	LapNumber int `json:"lap_number"`
	Position  int `json:"position"`
}

// SessionAnalysisInterval stores gap-to-leader data for one driver in one session.
type SessionAnalysisInterval struct {
	ID            string        `json:"id"`
	Type          string        `json:"type"` // "analysis_interval"
	Season        int           `json:"season"`
	Round         int           `json:"round"`
	MeetingKey    int           `json:"meeting_key"`
	SessionKey    int           `json:"session_key"`
	SessionType   string        `json:"session_type"`
	DriverNumber  int           `json:"driver_number"`
	DriverAcronym string        `json:"driver_acronym"`
	TeamName      string        `json:"team_name"`
	TeamColor     string        `json:"team_colour"` //nolint:misspell // OpenF1 API field name
	Laps          []IntervalLap `json:"laps"`
}

// IntervalLap is a single lap's gap data for a driver.
type IntervalLap struct {
	LapNumber   int     `json:"lap_number"`
	GapToLeader float64 `json:"gap_to_leader"`
	Interval    float64 `json:"interval"`
}

// SessionAnalysisStint stores one tire stint for one driver.
type SessionAnalysisStint struct {
	ID             string `json:"id"`
	Type           string `json:"type"` // "analysis_stint"
	Season         int    `json:"season"`
	Round          int    `json:"round"`
	MeetingKey     int    `json:"meeting_key"`
	SessionKey     int    `json:"session_key"`
	SessionType    string `json:"session_type"`
	DriverNumber   int    `json:"driver_number"`
	DriverAcronym  string `json:"driver_acronym"`
	TeamName       string `json:"team_name"`
	StintNumber    int    `json:"stint_number"`
	Compound       string `json:"compound"`
	LapStart       int    `json:"lap_start"`
	LapEnd         int    `json:"lap_end"`
	TireAgeAtStart int    `json:"tyre_age_at_start"` //nolint:misspell // OpenF1 API field name
}

// SessionAnalysisPit stores one pit stop event for one driver.
type SessionAnalysisPit struct {
	ID            string  `json:"id"`
	Type          string  `json:"type"` // "analysis_pit"
	Season        int     `json:"season"`
	Round         int     `json:"round"`
	MeetingKey    int     `json:"meeting_key"`
	SessionKey    int     `json:"session_key"`
	SessionType   string  `json:"session_type"`
	DriverNumber  int     `json:"driver_number"`
	DriverAcronym string  `json:"driver_acronym"`
	TeamName      string  `json:"team_name"`
	LapNumber     int     `json:"lap_number"`
	PitDuration   float64 `json:"pit_duration"`
	StopDuration  float64 `json:"stop_duration"`
}

// SessionAnalysisOvertake stores one overtake event.
type SessionAnalysisOvertake struct {
	ID                     string `json:"id"`
	Type                   string `json:"type"` // "analysis_overtake"
	Season                 int    `json:"season"`
	Round                  int    `json:"round"`
	MeetingKey             int    `json:"meeting_key"`
	SessionKey             int    `json:"session_key"`
	SessionType            string `json:"session_type"`
	OvertakingDriverNumber int    `json:"overtaking_driver_number"`
	OvertakingDriverName   string `json:"overtaking_driver_name"`
	OvertakenDriverNumber  int    `json:"overtaken_driver_number"`
	OvertakenDriverName    string `json:"overtaken_driver_name"`
	LapNumber              int    `json:"lap_number"`
	Position               int    `json:"position"`
}

// SessionAnalysisData is the combined query result for all analysis data in a session.
type SessionAnalysisData struct {
	Positions []SessionAnalysisPosition
	Intervals []SessionAnalysisInterval
	Stints    []SessionAnalysisStint
	Pits      []SessionAnalysisPit
	Overtakes []SessionAnalysisOvertake
}

// AnalysisRepository defines read/write operations for session analysis data.
type AnalysisRepository interface {
	UpsertSessionPositions(ctx context.Context, positions []SessionAnalysisPosition) error
	UpsertSessionIntervals(ctx context.Context, intervals []SessionAnalysisInterval) error
	UpsertSessionStints(ctx context.Context, stints []SessionAnalysisStint) error
	UpsertSessionPits(ctx context.Context, pits []SessionAnalysisPit) error
	UpsertSessionOvertakes(ctx context.Context, overtakes []SessionAnalysisOvertake) error
	GetSessionAnalysis(ctx context.Context, season, round int, sessionType string) (*SessionAnalysisData, error)
	// GetSessionAnalysisByMeetingKey queries analysis data using meeting_key instead of round.
	GetSessionAnalysisByMeetingKey(ctx context.Context, season, meetingKey int, sessionType string) (*SessionAnalysisData, error)
	HasAnalysisData(ctx context.Context, season, round int, sessionType string) (bool, error)
}

// --- Championship data types (Feature 007) ---

// DriverChampionshipSnapshot is a point-in-time record of a driver's championship
// standing after a specific Race or Sprint session.
type DriverChampionshipSnapshot struct {
	ID              string    `json:"id"`   // {season}-champ-driver-{session_key}-{driver_number}
	Type            string    `json:"type"` // "championship_driver"
	Season          int       `json:"season"`
	SessionKey      int       `json:"session_key"`
	MeetingKey      int       `json:"meeting_key"`
	DriverNumber    int       `json:"driver_number"`
	PositionStart   *int      `json:"position_start"` // nil for first race of season
	PositionCurrent int       `json:"position_current"`
	PointsStart     *float64  `json:"points_start"` // nil for first race of season
	PointsCurrent   float64   `json:"points_current"`
	DataAsOfUTC     time.Time `json:"data_as_of_utc"`
	Source          string    `json:"source"` // "openf1"
}

// TeamChampionshipSnapshot is a point-in-time record of a constructor team's
// championship standing after a specific Race or Sprint session.
type TeamChampionshipSnapshot struct {
	ID              string    `json:"id"`   // {season}-champ-team-{session_key}-{team_slug}
	Type            string    `json:"type"` // "championship_team"
	Season          int       `json:"season"`
	SessionKey      int       `json:"session_key"`
	MeetingKey      int       `json:"meeting_key"`
	TeamName        string    `json:"team_name"`
	TeamSlug        string    `json:"team_slug"`
	PositionStart   *int      `json:"position_start"`
	PositionCurrent int       `json:"position_current"`
	PointsStart     *float64  `json:"points_start"`
	PointsCurrent   float64   `json:"points_current"`
	DataAsOfUTC     time.Time `json:"data_as_of_utc"`
	Source          string    `json:"source"` // "openf1"
}

// ChampionshipSessionResult represents one driver's result in a completed Race
// or Sprint session. Used to derive wins, podiums, and DNFs.
type ChampionshipSessionResult struct {
	ID           string    `json:"id"`   // {season}-result-{session_key}-{driver_number}
	Type         string    `json:"type"` // "championship_result"
	Season       int       `json:"season"`
	SessionKey   int       `json:"session_key"`
	MeetingKey   int       `json:"meeting_key"`
	SessionType  string    `json:"session_type"` // "race" or "sprint"
	DriverNumber int       `json:"driver_number"`
	Position     int       `json:"position"`
	Points       float64   `json:"points"`
	DNF          bool      `json:"dnf"`
	DNS          bool      `json:"dns"`
	DSQ          bool      `json:"dsq"`
	NumberOfLaps int       `json:"number_of_laps"`
	GapToLeader  string    `json:"gap_to_leader"`
	Duration     float64   `json:"duration"`
	DataAsOfUTC  time.Time `json:"data_as_of_utc"`
	Source       string    `json:"source"` // "openf1"
}

// StartingGridEntry represents a driver's starting position for a race meeting.
// Used to derive pole positions.
type StartingGridEntry struct {
	ID           string    `json:"id"`   // {season}-grid-{meeting_key}-{driver_number}
	Type         string    `json:"type"` // "starting_grid"
	Season       int       `json:"season"`
	MeetingKey   int       `json:"meeting_key"`
	DriverNumber int       `json:"driver_number"`
	Position     int       `json:"position"`
	LapDuration  float64   `json:"lap_duration"`
	DataAsOfUTC  time.Time `json:"data_as_of_utc"`
	Source       string    `json:"source"` // "openf1"
}

// ChampionshipRepository defines read/write operations for championship standings data.
type ChampionshipRepository interface {
	UpsertDriverChampionshipSnapshots(ctx context.Context, snapshots []DriverChampionshipSnapshot) error
	GetDriverChampionshipSnapshots(ctx context.Context, season int) ([]DriverChampionshipSnapshot, error)
	UpsertTeamChampionshipSnapshots(ctx context.Context, snapshots []TeamChampionshipSnapshot) error
	GetTeamChampionshipSnapshots(ctx context.Context, season int) ([]TeamChampionshipSnapshot, error)
	UpsertChampionshipSessionResults(ctx context.Context, results []ChampionshipSessionResult) error
	GetChampionshipSessionResults(ctx context.Context, season int) ([]ChampionshipSessionResult, error)
	UpsertStartingGridEntries(ctx context.Context, entries []StartingGridEntry) error
	GetStartingGridEntries(ctx context.Context, season int) ([]StartingGridEntry, error)
}
