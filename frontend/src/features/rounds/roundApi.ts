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

export interface SessionDetail {
  session_name: string;
  session_type: string;
  status: string;
  date_start_utc: string;
  date_end_utc: string;
  results: SessionResultEntry[];
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
