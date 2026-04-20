import type { SessionResultEntry } from './roundApi';

interface Props {
  results: SessionResultEntry[];
}

function formatRaceTime(seconds: number | undefined): string {
  if (seconds == null) return '—';
  const hours = Math.floor(seconds / 3600);
  const mins = Math.floor((seconds % 3600) / 60);
  const secs = (seconds % 60).toFixed(3);
  if (hours > 0) return `${hours}:${String(mins).padStart(2, '0')}:${secs.padStart(6, '0')}`;
  if (mins > 0) return `${mins}:${secs.padStart(6, '0')}`;
  return secs;
}

const NON_CLASSIFIED_STATUSES = new Set(['DNF', 'DNS', 'DSQ', 'Retired', 'Disqualified']);

export default function RaceResults({ results }: Props) {
  const classified = results.filter(
    (r) => !NON_CLASSIFIED_STATUSES.has(r.finishing_status ?? '')
  );
  const nonClassified = results.filter((r) =>
    NON_CLASSIFIED_STATUSES.has(r.finishing_status ?? '')
  );

  return (
    <table className="results-table race-results">
      <thead>
        <tr>
          <th>Pos</th>
          <th>#</th>
          <th>Driver</th>
          <th>Team</th>
          <th>Time / Gap</th>
          <th>Laps</th>
          <th>Pts</th>
        </tr>
      </thead>
      <tbody>
        {classified.map((r) => (
          <tr key={r.driver_number} className={r.fastest_lap ? 'fastest-lap' : ''}>
            <td>{r.position}</td>
            <td>{r.driver_number}</td>
            <td>
              <span className="driver-acronym">{r.driver_acronym}</span>{' '}
              {r.driver_name}
              {r.fastest_lap && <span className="fastest-lap-badge" title="Fastest Lap">⏱</span>}
            </td>
            <td>{r.team_name}</td>
            <td>
              {r.position === 1
                ? formatRaceTime(r.race_time)
                : r.gap_to_leader ?? '—'}
            </td>
            <td>{r.number_of_laps}</td>
            <td>{r.points ?? 0}</td>
          </tr>
        ))}
        {nonClassified.length > 0 && (
          <>
            <tr className="non-classified-divider">
              <td colSpan={7}>Not Classified</td>
            </tr>
            {nonClassified.map((r) => (
              <tr key={r.driver_number} className="non-classified">
                <td>{r.position}</td>
                <td>{r.driver_number}</td>
                <td>
                  <span className="driver-acronym">{r.driver_acronym}</span>{' '}
                  {r.driver_name}
                </td>
                <td>{r.team_name}</td>
                <td>{r.finishing_status}</td>
                <td>{r.number_of_laps}</td>
                <td>{r.points ?? 0}</td>
              </tr>
            ))}
          </>
        )}
      </tbody>
    </table>
  );
}
