package ingest

import (
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/domain"
	"github.com/karlkuhnhausen/f1-race-intelligence/backend/internal/storage"
)

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

// TestTransformSessionResult wraps TransformSessionResult for external test access.
func TestTransformSessionResult(
	driverNumber, position int,
	driverFullName, driverAcronym, teamName string,
	sessionType domain.SessionType,
	season, round, totalLaps int,
) storage.SessionResult {
	pos := openF1Position{
		DriverNumber: driverNumber,
		Position:     position,
	}
	var driver *openF1Driver
	if driverFullName != "" {
		driver = &openF1Driver{
			DriverNumber: driverNumber,
			FullName:     driverFullName,
			NameAcronym:  driverAcronym,
			TeamName:     teamName,
		}
	}
	return TransformSessionResult(pos, driver, sessionType, season, round, totalLaps)
}
