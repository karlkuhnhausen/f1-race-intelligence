package analysis

// SessionAnalysisDTO is the top-level response for GET /api/v1/rounds/{round}/sessions/{type}/analysis.
type SessionAnalysisDTO struct {
	Year        int                 `json:"year"`
	Round       int                 `json:"round"`
	SessionType string              `json:"session_type"`
	TotalLaps   int                 `json:"total_laps"`
	Positions   []PositionDriverDTO `json:"positions"`
	Intervals   []IntervalDriverDTO `json:"intervals,omitempty"`
	Stints      []StintDTO          `json:"stints,omitempty"`
	Pits        []PitDTO            `json:"pits,omitempty"`
	Overtakes   []OvertakeDTO       `json:"overtakes,omitempty"`
}

// PositionDriverDTO contains position data for one driver.
type PositionDriverDTO struct {
	DriverNumber  int              `json:"driver_number"`
	DriverName    string           `json:"driver_name"`
	DriverAcronym string           `json:"driver_acronym"`
	TeamName      string           `json:"team_name"`
	TeamColour    string           `json:"team_colour"`
	Laps          []PositionLapDTO `json:"laps"`
}

// PositionLapDTO is one lap's position for a driver.
type PositionLapDTO struct {
	Lap      int `json:"lap"`
	Position int `json:"position"`
}

// IntervalDriverDTO contains gap data for one driver.
type IntervalDriverDTO struct {
	DriverNumber  int              `json:"driver_number"`
	DriverAcronym string           `json:"driver_acronym"`
	TeamName      string           `json:"team_name"`
	TeamColour    string           `json:"team_colour"`
	Laps          []IntervalLapDTO `json:"laps"`
}

// IntervalLapDTO is one lap's gap data for a driver.
type IntervalLapDTO struct {
	Lap         int     `json:"lap"`
	GapToLeader float64 `json:"gap_to_leader"`
	Interval    float64 `json:"interval"`
}

// StintDTO is one tire stint for one driver.
type StintDTO struct {
	DriverNumber   int    `json:"driver_number"`
	DriverAcronym  string `json:"driver_acronym"`
	TeamName       string `json:"team_name"`
	StintNumber    int    `json:"stint_number"`
	Compound       string `json:"compound"`
	LapStart       int    `json:"lap_start"`
	LapEnd         int    `json:"lap_end"`
	TyreAgeAtStart int    `json:"tyre_age_at_start"`
}

// PitDTO is one pit stop for one driver.
type PitDTO struct {
	DriverNumber  int     `json:"driver_number"`
	DriverAcronym string  `json:"driver_acronym"`
	TeamName      string  `json:"team_name"`
	Lap           int     `json:"lap"`
	PitDuration   float64 `json:"pit_duration"`
	StopDuration  float64 `json:"stop_duration,omitempty"`
}

// OvertakeDTO is one overtake event.
type OvertakeDTO struct {
	OvertakingDriverNumber int    `json:"overtaking_driver_number"`
	OvertakingDriverName   string `json:"overtaking_driver_name"`
	OvertakenDriverNumber  int    `json:"overtaken_driver_number"`
	OvertakenDriverName    string `json:"overtaken_driver_name"`
	Lap                    int    `json:"lap"`
	Position               int    `json:"position"`
}
