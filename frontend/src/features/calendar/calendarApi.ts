import { apiClient } from '../../services/apiClient';

export interface RaceMeetingDTO {
  round: number;
  race_name: string;
  circuit_name: string;
  country_name: string;
  start_datetime_utc: string;
  end_datetime_utc: string;
  status: 'scheduled' | 'cancelled' | 'completed' | 'unknown';
  is_cancelled: boolean;
  cancelled_label?: string;
  cancelled_reason?: string;
}

export interface ActiveSessionDTO {
  session_type: string;
  session_name: string;
  status: 'upcoming' | 'in_progress' | 'completed';
  date_start_utc: string;
  date_end_utc: string;
}

export interface CalendarResponse {
  year: number;
  data_as_of_utc: string;
  next_round: number;
  countdown_target_utc: string | null;
  weekend_in_progress?: boolean;
  active_session?: ActiveSessionDTO;
  rounds: RaceMeetingDTO[];
}

export async function fetchCalendar(year: number): Promise<CalendarResponse> {
  return apiClient.get<CalendarResponse>(`/calendar?year=${year}`);
}
