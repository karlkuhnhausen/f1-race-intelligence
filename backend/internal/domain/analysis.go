package domain

// AnalysisPosition is the domain representation of a driver's lap-by-lap position data.
type AnalysisPosition struct {
	DriverNumber  int
	DriverName    string
	DriverAcronym string
	TeamName      string
	TeamColor     string
	Laps          []PositionLap
}

// PositionLap is a single lap's position for a driver.
type PositionLap struct {
	LapNumber int
	Position  int
}

// AnalysisInterval is the domain representation of a driver's gap data.
type AnalysisInterval struct {
	DriverNumber  int
	DriverAcronym string
	TeamName      string
	TeamColor     string
	Laps          []IntervalLap
}

// IntervalLap is a single lap's gap data for a driver.
type IntervalLap struct {
	LapNumber   int
	GapToLeader float64
	Interval    float64
}

// AnalysisStint is the domain representation of one tire stint.
type AnalysisStint struct {
	DriverNumber   int
	DriverAcronym  string
	TeamName       string
	StintNumber    int
	Compound       string // SOFT, MEDIUM, HARD, INTERMEDIATE, WET
	LapStart       int
	LapEnd         int
	TireAgeAtStart int
}

// AnalysisPit is the domain representation of one pit stop.
type AnalysisPit struct {
	DriverNumber  int
	DriverAcronym string
	TeamName      string
	LapNumber     int
	PitDuration   float64 // total pit lane time (seconds)
	StopDuration  float64 // stationary time (seconds); 0 if unavailable
}

// AnalysisOvertake is the domain representation of one overtake event.
type AnalysisOvertake struct {
	OvertakingDriverNumber int
	OvertakingDriverName   string
	OvertakenDriverNumber  int
	OvertakenDriverName    string
	LapNumber              int
	Position               int
}

// AnalysisData is the combined result of all analysis data for a session.
type AnalysisData struct {
	Positions []AnalysisPosition
	Intervals []AnalysisInterval
	Stints    []AnalysisStint
	Pits      []AnalysisPit
	Overtakes []AnalysisOvertake
}
