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

export default function QualifyingResults({ results }: Props) {
  if (results.length === 0) {
    return <p>Not yet available</p>;
  }

  return (
    <table className="results-table qualifying-results">
      <thead>
        <tr>
          <th>Pos</th>
          <th>#</th>
          <th>Driver</th>
          <th>Team</th>
          <th>Q1</th>
          <th>Q2</th>
          <th>Q3</th>
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
            <td>{formatLapTime(r.q1_time)}</td>
            <td>{formatLapTime(r.q2_time)}</td>
            <td>{formatLapTime(r.q3_time)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
