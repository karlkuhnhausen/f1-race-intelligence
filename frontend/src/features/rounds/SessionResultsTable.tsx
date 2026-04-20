import type { SessionResultEntry } from './roundApi';

interface Props {
  results: SessionResultEntry[];
  sessionType: string;
}

const RACE_TYPES = new Set(['race', 'sprint']);
const QUALIFYING_TYPES = new Set(['qualifying', 'sprint_qualifying']);
const PRACTICE_TYPES = new Set(['practice1', 'practice2', 'practice3']);

function formatLapTime(seconds: number | undefined): string {
  if (seconds == null) return '—';
  const mins = Math.floor(seconds / 60);
  const secs = (seconds % 60).toFixed(3);
  return mins > 0 ? `${mins}:${secs.padStart(6, '0')}` : secs;
}

export default function SessionResultsTable({ results, sessionType }: Props) {
  const isRace = RACE_TYPES.has(sessionType);
  const isQualifying = QUALIFYING_TYPES.has(sessionType);
  const isPractice = PRACTICE_TYPES.has(sessionType);

  return (
    <table className="results-table">
      <thead>
        <tr>
          <th>Pos</th>
          <th>#</th>
          <th>Driver</th>
          <th>Team</th>
          {isRace && (
            <>
              <th>Status</th>
              <th>Gap</th>
              <th>Pts</th>
            </>
          )}
          {isQualifying && (
            <>
              <th>Q1</th>
              <th>Q2</th>
              <th>Q3</th>
            </>
          )}
          {isPractice && (
            <>
              <th>Best Lap</th>
              <th>Gap</th>
            </>
          )}
        </tr>
      </thead>
      <tbody>
        {results.map((r) => (
          <tr key={r.driver_number} className={r.fastest_lap ? 'fastest-lap' : ''}>
            <td>{r.position}</td>
            <td>{r.driver_number}</td>
            <td>
              <span className="driver-acronym">{r.driver_acronym}</span>{' '}
              {r.driver_name}
            </td>
            <td>{r.team_name}</td>
            {isRace && (
              <>
                <td>{r.finishing_status ?? '—'}</td>
                <td>{r.gap_to_leader ?? '—'}</td>
                <td>{r.points ?? 0}</td>
              </>
            )}
            {isQualifying && (
              <>
                <td>{formatLapTime(r.q1_time)}</td>
                <td>{formatLapTime(r.q2_time)}</td>
                <td>{formatLapTime(r.q3_time)}</td>
              </>
            )}
            {isPractice && (
              <>
                <td>{formatLapTime(r.best_lap_time)}</td>
                <td>{r.gap_to_fastest != null ? `+${r.gap_to_fastest.toFixed(3)}s` : '—'}</td>
              </>
            )}
          </tr>
        ))}
      </tbody>
    </table>
  );
}
