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

export interface DriverProgressionEntry {
  driver_number: number;
  driver_name: string;
  team_name: string;
  team_color: string;
  points_by_round: number[];
}

export interface TeamProgressionEntry {
  team_name: string;
  team_color: string;
  points_by_round: number[];
}

export interface DriversProgressionResponse {
  year: number;
  rounds: string[];
  drivers: DriverProgressionEntry[];
}

export interface ConstructorsProgressionResponse {
  year: number;
  rounds: string[];
  teams: TeamProgressionEntry[];
}

export async function fetchDriverProgression(year: number): Promise<DriversProgressionResponse> {
  return apiClient.get<DriversProgressionResponse>(`/standings/drivers/progression?year=${year}`);
}

export async function fetchConstructorProgression(year: number): Promise<ConstructorsProgressionResponse> {
  return apiClient.get<ConstructorsProgressionResponse>(`/standings/constructors/progression?year=${year}`);
}

export interface ComparisonDriverStats {
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

export interface ComparisonTeamStats {
  team_name: string;
  team_color: string;
  points: number;
  wins: number;
  podiums: number;
  dnfs: number;
}

export interface ComparisonDeltas {
  points: number;
  wins: number;
  podiums: number;
  dnfs: number;
  poles: number;
}

export interface DriverComparisonResponse {
  year: number;
  driver1: ComparisonDriverStats;
  driver2: ComparisonDriverStats;
  deltas: ComparisonDeltas;
  rounds: string[];
  driver1_points: number[];
  driver2_points: number[];
}

export interface ConstructorComparisonResponse {
  year: number;
  team1: ComparisonTeamStats;
  team2: ComparisonTeamStats;
  deltas: ComparisonDeltas;
  rounds: string[];
  team1_points: number[];
  team2_points: number[];
}

export async function fetchDriverComparison(year: number, driver1: number, driver2: number): Promise<DriverComparisonResponse> {
  return apiClient.get<DriverComparisonResponse>(`/standings/drivers/compare?year=${year}&driver1=${driver1}&driver2=${driver2}`);
}

export async function fetchConstructorComparison(year: number, team1: string, team2: string): Promise<ConstructorComparisonResponse> {
  return apiClient.get<ConstructorComparisonResponse>(`/standings/constructors/compare?year=${year}&team1=${encodeURIComponent(team1)}&team2=${encodeURIComponent(team2)}`);
}

export interface ConstructorDriverEntry {
  driver_number: number;
  driver_name: string;
  position: number;
  points: number;
  wins: number;
  podiums: number;
  points_percentage: number;
}

export interface ConstructorBreakdownResponse {
  year: number;
  team_name: string;
  team_points: number;
  drivers: ConstructorDriverEntry[];
}

export async function fetchConstructorDriverBreakdown(year: number, teamName: string): Promise<ConstructorBreakdownResponse> {
  return apiClient.get<ConstructorBreakdownResponse>(`/standings/constructors/${encodeURIComponent(teamName)}/drivers?year=${year}`);
}
