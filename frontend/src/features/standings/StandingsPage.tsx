import { useEffect, useState } from 'react';
import {
  fetchDriverStandings,
  fetchConstructorStandings,
  fetchDriverProgression,
  fetchConstructorProgression,
  type DriverStandingDTO,
  type ConstructorStandingDTO,
  type DriversProgressionResponse,
  type ConstructorsProgressionResponse,
} from './standingsApi';
import StandingsTable, {
  type StandingsRow,
} from '../design-system/StandingsTable';
import ProgressionChart, { type ProgressionEntry } from './ProgressionChart';
import YearPicker from './YearPicker';

type Tab = 'drivers' | 'constructors';
type ViewMode = 'table' | 'chart';

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
  const [viewMode, setViewMode] = useState<ViewMode>('table');
  const [drivers, setDrivers] = useState<DriverStandingDTO[]>([]);
  const [constructors, setConstructors] = useState<ConstructorStandingDTO[]>([]);
  const [driverProgression, setDriverProgression] = useState<DriversProgressionResponse | null>(null);
  const [constructorProgression, setConstructorProgression] = useState<ConstructorsProgressionResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [year, setYear] = useState(new Date().getFullYear());

  useEffect(() => {
    setLoading(true);
    setError(null);

    Promise.all([
      fetchDriverStandings(year),
      fetchConstructorStandings(year),
      fetchDriverProgression(year),
      fetchConstructorProgression(year),
    ])
      .then(([d, c, dp, cp]) => {
        setDrivers(d.rows ?? []);
        setConstructors(c.rows ?? []);
        setDriverProgression(dp);
        setConstructorProgression(cp);
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
      <div className="flex items-center gap-4">
        <h2 className="font-display text-3xl font-bold tracking-tight">
          {year} Standings
        </h2>
        <YearPicker selectedYear={year} onYearChange={setYear} />
      </div>

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
        <div className="ml-auto flex gap-1 items-center">
          <button
            type="button"
            onClick={() => setViewMode('table')}
            className={tabClass(viewMode === 'table')}
          >
            Table
          </button>
          <button
            type="button"
            onClick={() => setViewMode('chart')}
            className={tabClass(viewMode === 'chart')}
          >
            Chart
          </button>
        </div>
      </div>

      {viewMode === 'table' ? (
        tab === 'drivers' ? (
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
        )
      ) : (
        tab === 'drivers' && driverProgression ? (
          <ProgressionChart
            rounds={driverProgression.rounds}
            entries={driverProgression.drivers.map((d): ProgressionEntry => ({
              name: d.driver_name,
              color: d.team_color,
              pointsByRound: d.points_by_round,
            }))}
          />
        ) : constructorProgression ? (
          <ProgressionChart
            rounds={constructorProgression.rounds}
            entries={constructorProgression.teams.map((t): ProgressionEntry => ({
              name: t.team_name,
              color: t.team_color,
              pointsByRound: t.points_by_round,
            }))}
          />
        ) : null
      )}
    </section>
  );
}
