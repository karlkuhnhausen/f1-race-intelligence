export interface PositionLap {
  lap: number;
  position: number;
}

export interface PositionDriver {
  driver_number: number;
  driver_name: string;
  driver_acronym: string;
  team_name: string;
  team_colour: string;
  laps: PositionLap[];
}

export interface IntervalLap {
  lap: number;
  gap_to_leader: number;
  interval: number;
}

export interface IntervalDriver {
  driver_number: number;
  driver_acronym: string;
  team_name: string;
  team_colour: string;
  laps: IntervalLap[];
}

export interface Stint {
  driver_number: number;
  driver_acronym: string;
  team_name: string;
  stint_number: number;
  compound: "SOFT" | "MEDIUM" | "HARD" | "INTERMEDIATE" | "WET";
  lap_start: number;
  lap_end: number;
  tyre_age_at_start: number;
}

export interface PitStop {
  driver_number: number;
  driver_acronym: string;
  team_name: string;
  lap: number;
  pit_duration: number;
  stop_duration?: number;
}

export interface Overtake {
  overtaking_driver_number: number;
  overtaking_driver_name: string;
  overtaken_driver_number: number;
  overtaken_driver_name: string;
  lap: number;
  position: number;
}

export interface SessionAnalysisResponse {
  year: number;
  round: number;
  session_type: string;
  total_laps: number;
  positions: PositionDriver[];
  intervals?: IntervalDriver[];
  stints?: Stint[];
  pits?: PitStop[];
  overtakes?: Overtake[];
}
