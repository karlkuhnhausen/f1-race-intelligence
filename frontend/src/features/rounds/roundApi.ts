import { apiClient } from '../../services/apiClient';

export interface SessionResultEntry {
  position: number;
  driver_number: number;
  driver_name: string;
  driver_acronym: string;
  team_name: string;
  number_of_laps: number;
  finishing_status?: string;
  race_time?: number;
  gap_to_leader?: string;
  points?: number;
  fastest_lap?: boolean;
  q1_time?: number;
  q2_time?: number;
  q3_time?: number;
  best_lap_time?: number;
  gap_to_fastest?: number;
}

export interface NotableEvent {
  event_type: 'red_flag' | 'safety_car' | 'vsc' | 'investigation';
  lap_number: number;
  count: number;
}

export interface SessionRecapSummary {
  // Race / Sprint
  winner_name?: string;
  winner_team?: string;
  gap_to_p2?: string;
  fastest_lap_holder?: string;
  fastest_lap_team?: string;
  fastest_lap_time_seconds?: number;
  total_laps?: number;
  // Qualifying / Sprint Qualifying
  pole_sitter_name?: string;
  pole_sitter_team?: string;
  pole_time?: number;
  gap_to_p2_qualifying?: string;
  q1_cutoff_time?: number;
  q2_cutoff_time?: number;
  // Practice
  best_driver_name?: string;
  best_driver_team?: string;
  best_lap_time?: number;
  // All sessions
  red_flag_count?: number;
  safety_car_count?: number;
  vsc_count?: number;
  top_event?: NotableEvent;
}

export interface SessionDetail {
  session_name: string;
  session_type: string;
  status: string;
  date_start_utc: string;
  date_end_utc: string;
  results: SessionResultEntry[];
  recap_summary?: SessionRecapSummary;
}

export interface RoundDetailResponse {
  year: number;
  round: number;
  race_name: string;
  circuit_name: string;
  country_name: string;
  data_as_of_utc: string;
  sessions: SessionDetail[];
}

export function fetchRoundDetail(round: number, year = 2026): Promise<RoundDetailResponse> {
  return apiClient.get<RoundDetailResponse>(`/rounds/${round}?year=${year}`);
}
