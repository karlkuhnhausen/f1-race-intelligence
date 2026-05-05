package ingest

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/domain"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

// OpenF1MeetingForTest mirrors the unexported openF1Meeting fields used by
// NormalizeMeetings. Exposed so tests in other packages can construct
// fixtures without depending on the unexported type.
type OpenF1MeetingForTest struct {
	MeetingName string
	DateStart   string
	DateEnd     string
	IsCancelled bool
	MeetingKey  int
}

// NormalizeMeetingsForTest wraps NormalizeMeetings for external test access.
func NormalizeMeetingsForTest(raw []OpenF1MeetingForTest, season int) []storage.RaceMeeting {
	internal := make([]openF1Meeting, len(raw))
	for i, r := range raw {
		internal[i] = openF1Meeting{
			MeetingName: r.MeetingName,
			DateStart:   r.DateStart,
			DateEnd:     r.DateEnd,
			IsCancelled: r.IsCancelled,
			MeetingKey:  r.MeetingKey,
		}
	}
	return NormalizeMeetings(internal, season)
}

// TestTransformSession wraps TransformSession for external test access.
func TestTransformSession(sessionKey int, sessionName string, meetingKey int, dateStart, dateEnd string, year, season, round int) storage.Session {
	raw := openF1Session{
		SessionKey:  sessionKey,
		SessionName: sessionName,
		MeetingKey:  meetingKey,
		DateStart:   dateStart,
		DateEnd:     dateEnd,
		Year:        year,
	}
	return TransformSession(raw, season, round)
}

// TestTransformSessionResult wraps TransformSessionResult for the simple path
// (driver number / position / total laps). Used by older tests that don't
// exercise the rich race / qualifying / practice fields.
func TestTransformSessionResult(
	driverNumber, position int,
	driverFullName, driverAcronym, teamName string,
	sessionType domain.SessionType,
	season, round, totalLaps int,
) storage.SessionResult {
	raw := openF1SessionResult{
		Position:     position,
		DriverNumber: driverNumber,
		NumberOfLaps: totalLaps,
	}
	driver := buildTestDriver(driverNumber, driverFullName, driverAcronym, teamName)
	return TransformSessionResult(raw, driver, sessionType, season, round)
}

// TestTransformSessionResultJSON wraps TransformSessionResult when the test
// needs full control over the upstream JSON shape (e.g. polymorphic duration
// arrays for qualifying, dnf/dns/dsq booleans, points). The provided JSON must
// be a single OpenF1 session_result object.
func TestTransformSessionResultJSON(
	rawJSON string,
	driverFullName, driverAcronym, teamName string,
	sessionType domain.SessionType,
	season, round int,
) (storage.SessionResult, error) {
	var raw openF1SessionResult
	if err := json.Unmarshal([]byte(rawJSON), &raw); err != nil {
		return storage.SessionResult{}, err
	}
	driver := buildTestDriver(raw.DriverNumber, driverFullName, driverAcronym, teamName)
	return TransformSessionResult(raw, driver, sessionType, season, round), nil
}

// TestDeriveFastestLap wraps DeriveFastestLap, accepting a JSON array of
// OpenF1 lap objects so callers can simulate real upstream payloads.
func TestDeriveFastestLap(lapsJSON string) (int, bool, error) {
	var laps []openF1Lap
	if err := json.Unmarshal([]byte(lapsJSON), &laps); err != nil {
		return 0, false, err
	}
	driver, ok := DeriveFastestLap(laps)
	return driver, ok, nil
}

func buildTestDriver(driverNumber int, fullName, acronym, team string) *openF1Driver {
	if fullName == "" {
		return nil
	}
	return &openF1Driver{
		DriverNumber: driverNumber,
		FullName:     fullName,
		NameAcronym:  acronym,
		TeamName:     team,
	}
}

// OpenF1SessionForTest mirrors the unexported openF1Session for external tests.
type OpenF1SessionForTest struct {
	SessionKey  int
	SessionName string
	MeetingKey  int
	DateStart   string
}

// BuildMeetingRoundMapForTest wraps buildMeetingRoundMap for external test access.
func BuildMeetingRoundMapForTest(sessions []OpenF1SessionForTest, cancelledKeys map[int]bool) map[int]int {
	internal := make([]openF1Session, len(sessions))
	for i, s := range sessions {
		internal[i] = openF1Session{
			SessionKey:  s.SessionKey,
			SessionName: s.SessionName,
			MeetingKey:  s.MeetingKey,
			DateStart:   s.DateStart,
		}
	}
	p := &SessionPoller{
		logger: slog.Default(),
		client: &http.Client{},
	}
	return p.buildMeetingRoundMap(internal, cancelledKeys)
}
