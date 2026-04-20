package domain

// SessionType enumerates the types of sessions in an F1 weekend.
type SessionType string

const (
	SessionPractice1        SessionType = "practice1"
	SessionPractice2        SessionType = "practice2"
	SessionPractice3        SessionType = "practice3"
	SessionSprintQualifying SessionType = "sprint_qualifying"
	SessionSprint           SessionType = "sprint"
	SessionQualifying       SessionType = "qualifying"
	SessionRace             SessionType = "race"
)

// SessionStatus represents the availability status of a session.
type SessionStatus string

const (
	SessionStatusCompleted    SessionStatus = "completed"
	SessionStatusInProgress   SessionStatus = "in_progress"
	SessionStatusUpcoming     SessionStatus = "upcoming"
	SessionStatusNotAvailable SessionStatus = "not_available"
)

// FinishingStatus represents a driver's race finishing status.
type FinishingStatus string

const (
	FinishStatusFinished FinishingStatus = "Finished"
	FinishStatusDNF      FinishingStatus = "DNF"
	FinishStatusDNS      FinishingStatus = "DNS"
	FinishStatusDSQ      FinishingStatus = "DSQ"
)

// MapOpenF1SessionType maps OpenF1 session_name to our SessionType enum.
func MapOpenF1SessionType(name string) SessionType {
	switch name {
	case "Practice 1":
		return SessionPractice1
	case "Practice 2":
		return SessionPractice2
	case "Practice 3":
		return SessionPractice3
	case "Sprint Qualifying":
		return SessionSprintQualifying
	case "Sprint":
		return SessionSprint
	case "Qualifying":
		return SessionQualifying
	case "Race":
		return SessionRace
	default:
		return SessionType(name)
	}
}

// SessionTypeSlug returns the slug used in Cosmos DB document IDs.
func SessionTypeSlug(st SessionType) string {
	switch st {
	case SessionPractice1:
		return "fp1"
	case SessionPractice2:
		return "fp2"
	case SessionPractice3:
		return "fp3"
	case SessionSprintQualifying:
		return "sprint-qualifying"
	case SessionSprint:
		return "sprint"
	case SessionQualifying:
		return "qualifying"
	case SessionRace:
		return "race"
	default:
		return string(st)
	}
}

// IsRaceType returns true for race and sprint session types.
func IsRaceType(st SessionType) bool {
	return st == SessionRace || st == SessionSprint
}

// IsQualifyingType returns true for qualifying and sprint qualifying session types.
func IsQualifyingType(st SessionType) bool {
	return st == SessionQualifying || st == SessionSprintQualifying
}

// IsPracticeType returns true for practice session types.
func IsPracticeType(st SessionType) bool {
	return st == SessionPractice1 || st == SessionPractice2 || st == SessionPractice3
}
