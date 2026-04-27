import type { SessionResultEntry } from './roundApi';

interface Props {
  results: SessionResultEntry[];
}

function formatLapTime(seconds: number | undefined): string {
  if (seconds == null) return '—';
  const mins = Math.floor(seconds / 60);
  const secs = (seconds % 60).toFixed(3);
  return mins > 0 ? `${mins}:${secs.padStart(6, '0')}` : secs;
}

export default function PracticeResults({ results }: Props) {
  if (results.length === 0) {
    return <p>Not yet available</p>;
  }

  return (
    <table className="results-table practice-results">
      <thead>
        <tr>
          <th>Pos</th>
          <th>#</th>
          <th>Driver</th>
          <th>Team</th>
          <th>Best Lap</th>
          <th>Gap</th>
          <th>Laps</th>
        </tr>
      </thead>
      <tbody>
        {results.map((r) => (
          <tr key={r.driver_number}>
            <td>{r.position}</td>
            <td>{r.driver_number}</td>
            <td>
              <span className="driver-acronym">{r.driver_acronym}</span>{' '}
              {r.driver_name}
            </td>
            <td>{r.team_name}</td>
            <td>{formatLapTime(r.best_lap_time)}</td>
            <td>
              {r.gap_to_fastest != null && r.gap_to_fastest > 0
                ? `+${r.gap_to_fastest.toFixed(3)}s`
                : r.gap_to_fastest === 0
                  ? '—'
                  : '—'}
            </td>
            <td>{r.number_of_laps}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
