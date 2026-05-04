package standings

import "time"

// DriverStandingDTO represents a single row in the drivers championship response.
type DriverStandingDTO struct {
	Position     int     `json:"position"`
	DriverNumber int     `json:"driver_number"`
	DriverName   string  `json:"driver_name"`
	TeamName     string  `json:"team_name"`
	TeamColor    string  `json:"team_color"`
	Points       float64 `json:"points"`
	Wins         int     `json:"wins"`
	Podiums      int     `json:"podiums"`
	DNFs         int     `json:"dnfs"`
	Poles        int     `json:"poles"`
}

// ConstructorStandingDTO represents a single row in the constructors championship response.
type ConstructorStandingDTO struct {
	Position  int     `json:"position"`
	TeamName  string  `json:"team_name"`
	TeamColor string  `json:"team_color"`
	Points    float64 `json:"points"`
	Wins      int     `json:"wins"`
	Podiums   int     `json:"podiums"`
	DNFs      int     `json:"dnfs"`
}

// DriversStandingsResponse is the top-level envelope for GET /api/v1/standings/drivers.
type DriversStandingsResponse struct {
	Year        int                 `json:"year"`
	DataAsOfUTC time.Time           `json:"data_as_of_utc"`
	Rows        []DriverStandingDTO `json:"rows"`
}

// ConstructorsStandingsResponse is the top-level envelope for GET /api/v1/standings/constructors.
type ConstructorsStandingsResponse struct {
	Year        int                      `json:"year"`
	DataAsOfUTC time.Time                `json:"data_as_of_utc"`
	Rows        []ConstructorStandingDTO `json:"rows"`
}
