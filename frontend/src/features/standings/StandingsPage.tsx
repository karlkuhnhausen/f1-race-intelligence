import { useEffect, useState } from 'react';
import {
  fetchDriverStandings,
  fetchConstructorStandings,
  type DriverStandingDTO,
  type ConstructorStandingDTO,
} from './standingsApi';
import StandingsTable, {
  type StandingsRow,
} from '../design-system/StandingsTable';

type Tab = 'drivers' | 'constructors';

function driversToRows(drivers: DriverStandingDTO[]): StandingsRow[] {
  return drivers.map((d) => ({
    position: d.position,
    name: d.driver_name,
    constructorId: d.team_name,
    points: d.points,
    wins: d.wins,
    podiums: d.podiums,
    dnfs: d.dnfs,
    poles: d.poles,
    teamColor: d.team_color || undefined,
  }));
}

function constructorsToRows(
  constructors: ConstructorStandingDTO[],
): StandingsRow[] {
  return constructors.map((c) => ({
    position: c.position,
    name: c.team_name,
    constructorId: c.team_name,
    points: c.points,
    wins: c.wins,
    podiums: c.podiums,
    dnfs: c.dnfs,
    teamColor: c.team_color || undefined,
  }));
}

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

  if (loading) return <p className="text-muted-foreground">Loading standings…</p>;
  if (error) return <p className="text-negative">Error: {error}</p>;

  const tabClass = (active: boolean) =>
    `px-4 py-2 font-display text-sm uppercase tracking-wider transition-colors border-b-2 ${
      active
        ? 'border-accent-red text-foreground'
        : 'border-transparent text-muted-foreground hover:text-foreground'
    }`;

  return (
    <section className="space-y-6">
      <h2 className="font-display text-3xl font-bold tracking-tight">
        {year} Standings
      </h2>

      <div className="flex gap-2 border-b border-border">
        <button
          type="button"
          onClick={() => setTab('drivers')}
          className={tabClass(tab === 'drivers')}
        >
          Drivers
        </button>
        <button
          type="button"
          onClick={() => setTab('constructors')}
          className={tabClass(tab === 'constructors')}
        >
          Constructors
        </button>
      </div>

      {tab === 'drivers' ? (
        <StandingsTable
          title="Drivers Championship"
          nameLabel="Driver"
          rows={driversToRows(drivers)}
          columns={["wins", "podiums", "dnfs", "poles"]}
        />
      ) : (
        <StandingsTable
          title="Constructors Championship"
          nameLabel="Team"
          rows={constructorsToRows(constructors)}
          columns={["wins", "podiums", "dnfs"]}
        />
      )}
    </section>
  );
}
