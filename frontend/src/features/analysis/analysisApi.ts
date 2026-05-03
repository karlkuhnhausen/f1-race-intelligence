import { apiClient, ApiClientError } from '../../services/apiClient';
import type { SessionAnalysisResponse } from './analysisTypes';

/**
 * Fetch session analysis data for a given round and session type.
 * Returns null if analysis is not yet available (404).
 */
export async function fetchSessionAnalysis(
  round: number,
  sessionType: string,
  year: number = new Date().getFullYear(),
): Promise<SessionAnalysisResponse | null> {
  try {
    return await apiClient.get<SessionAnalysisResponse>(
      `/rounds/${round}/sessions/${sessionType}/analysis?year=${year}`,
    );
  } catch (err) {
    if (err instanceof ApiClientError && err.status === 404) {
      return null;
    }
    throw err;
  }
}
