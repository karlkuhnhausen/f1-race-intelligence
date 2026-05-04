import { apiClient } from '../../services/apiClient';

export interface DriverStandingDTO {
  position: number;
  driver_number: number;
  driver_name: string;
  team_name: string;
  team_color: string;
  points: number;
  wins: number;
  podiums: number;
  dnfs: number;
  poles: number;
}

export interface ConstructorStandingDTO {
  position: number;
  team_name: string;
  team_color: string;
  points: number;
  wins: number;
  podiums: number;
  dnfs: number;
}

export interface DriversStandingsResponse {
  year: number;
  data_as_of_utc: string;
  rows: DriverStandingDTO[];
}

export interface ConstructorsStandingsResponse {
  year: number;
  data_as_of_utc: string;
  rows: ConstructorStandingDTO[];
}

export async function fetchDriverStandings(year: number): Promise<DriversStandingsResponse> {
  return apiClient.get<DriversStandingsResponse>(`/standings/drivers?year=${year}`);
}

export async function fetchConstructorStandings(year: number): Promise<ConstructorsStandingsResponse> {
  return apiClient.get<ConstructorsStandingsResponse>(`/standings/constructors?year=${year}`);
}
