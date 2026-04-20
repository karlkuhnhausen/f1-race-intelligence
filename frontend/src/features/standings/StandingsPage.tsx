import { useEffect, useState } from 'react';
import {
  fetchDriverStandings,
  fetchConstructorStandings,
  type DriverStandingDTO,
  type ConstructorStandingDTO,
} from './standingsApi';

type Tab = 'drivers' | 'constructors';

export default function StandingsPage() {
  const [tab, setTab] = useState<Tab>('drivers');
  const [drivers, setDrivers] = useState<DriverStandingDTO[]>([]);
  const [constructors, setConstructors] = useState<ConstructorStandingDTO[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const year = new Date().getFullYear();

  useEffect(() => {
    setLoading(true);
    setError(null);

    Promise.all([fetchDriverStandings(year), fetchConstructorStandings(year)])
      .then(([d, c]) => {
        setDrivers(d.rows ?? []);
        setConstructors(c.rows ?? []);
      })
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false));
  }, [year]);

  if (loading) return <p>Loading standings…</p>;
  if (error) return <p>Error: {error}</p>;

  return (
    <section>
      <div style={{ marginBottom: '1rem' }}>
        <button
          onClick={() => setTab('drivers')}
          style={{ fontWeight: tab === 'drivers' ? 'bold' : 'normal', marginRight: '0.5rem' }}
        >
          Drivers
        </button>
        <button
          onClick={() => setTab('constructors')}
          style={{ fontWeight: tab === 'constructors' ? 'bold' : 'normal' }}
        >
          Constructors
        </button>
      </div>

      {tab === 'drivers' ? (
        <table>
          <thead>
            <tr>
              <th>Pos</th>
              <th>Driver</th>
              <th>Team</th>
              <th>Points</th>
              <th>Wins</th>
            </tr>
          </thead>
          <tbody>
            {drivers.map((d) => (
              <tr key={d.position}>
                <td>{d.position}</td>
                <td>{d.driver_name}</td>
                <td>{d.team_name}</td>
                <td>{d.points}</td>
                <td>{d.wins}</td>
              </tr>
            ))}
            {drivers.length === 0 && (
              <tr>
                <td colSpan={5}>No driver standings available yet.</td>
              </tr>
            )}
          </tbody>
        </table>
      ) : (
        <table>
          <thead>
            <tr>
              <th>Pos</th>
              <th>Team</th>
              <th>Points</th>
            </tr>
          </thead>
          <tbody>
            {constructors.map((c) => (
              <tr key={c.position}>
                <td>{c.position}</td>
                <td>{c.team_name}</td>
                <td>{c.points}</td>
              </tr>
            ))}
            {constructors.length === 0 && (
              <tr>
                <td colSpan={3}>No constructor standings available yet.</td>
              </tr>
            )}
          </tbody>
        </table>
      )}
    </section>
  );
}
