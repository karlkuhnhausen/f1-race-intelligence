import { useEffect, useState } from 'react';
import {
  fetchDriverStandings,
  fetchConstructorStandings,
  fetchDriverProgression,
  fetchConstructorProgression,
  fetchDriverComparison,
  fetchConstructorComparison,
  type DriverStandingDTO,
  type ConstructorStandingDTO,
  type DriversProgressionResponse,
  type ConstructorsProgressionResponse,
  type DriverComparisonResponse,
  type ConstructorComparisonResponse,
} from './standingsApi';
import StandingsTable, {
  type StandingsRow,
} from '../design-system/StandingsTable';
import ProgressionChart, { type ProgressionEntry } from './ProgressionChart';
import YearPicker from './YearPicker';
import ComparisonPanel from './ComparisonPanel';

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
  const [comparison, setComparison] = useState<DriverComparisonResponse | ConstructorComparisonResponse | null>(null);
  const [compareLoading, setCompareLoading] = useState(false);
  const [selectedDrivers, setSelectedDrivers] = useState<number[]>([]);
  const [selectedTeams, setSelectedTeams] = useState<string[]>([]);

  useEffect(() => {
    setComparison(null);
    setSelectedDrivers([]);
    setSelectedTeams([]);
  }, [year]);

  useEffect(() => {
    if (selectedDrivers.length === 2) {
      setCompareLoading(true);
      fetchDriverComparison(year, selectedDrivers[0], selectedDrivers[1])
        .then(setComparison)
        .catch(() => setComparison(null))
        .finally(() => setCompareLoading(false));
    } else if (selectedTeams.length === 2) {
      setCompareLoading(true);
      fetchConstructorComparison(year, selectedTeams[0], selectedTeams[1])
        .then(setComparison)
        .catch(() => setComparison(null))
        .finally(() => setCompareLoading(false));
    } else {
      setComparison(null);
    }
  }, [selectedDrivers, selectedTeams, year]);

  function toggleDriverSelect(driverNumber: number) {
    setSelectedDrivers((prev) => {
      if (prev.includes(driverNumber)) return prev.filter((d) => d !== driverNumber);
      if (prev.length >= 2) return [prev[1], driverNumber];
      return [...prev, driverNumber];
    });
  }

  function toggleTeamSelect(teamName: string) {
    setSelectedTeams((prev) => {
      if (prev.includes(teamName)) return prev.filter((t) => t !== teamName);
      if (prev.length >= 2) return [prev[1], teamName];
      return [...prev, teamName];
    });
  }

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

      {viewMode === 'table' && (
        <div className="space-y-2">
          <p className="text-xs text-muted-foreground">
            {tab === 'drivers'
              ? `Click two drivers to compare${selectedDrivers.length > 0 ? ` (${selectedDrivers.length}/2 selected)` : ''}`
              : `Click two teams to compare${selectedTeams.length > 0 ? ` (${selectedTeams.length}/2 selected)` : ''}`}
          </p>
          <div className="flex flex-wrap gap-1">
            {tab === 'drivers'
              ? drivers.map((d) => (
                  <button
                    key={d.driver_number}
                    type="button"
                    onClick={() => toggleDriverSelect(d.driver_number)}
                    className={`text-xs px-2 py-1 rounded border ${
                      selectedDrivers.includes(d.driver_number)
                        ? 'border-accent-red bg-accent-red/10 text-foreground'
                        : 'border-border text-muted-foreground hover:text-foreground'
                    }`}
                  >
                    {d.driver_name}
                  </button>
                ))
              : constructors.map((c) => (
                  <button
                    key={c.team_name}
                    type="button"
                    onClick={() => toggleTeamSelect(c.team_name)}
                    className={`text-xs px-2 py-1 rounded border ${
                      selectedTeams.includes(c.team_name)
                        ? 'border-accent-red bg-accent-red/10 text-foreground'
                        : 'border-border text-muted-foreground hover:text-foreground'
                    }`}
                  >
                    {c.team_name}
                  </button>
                ))}
          </div>
          <ComparisonPanel data={comparison} loading={compareLoading} />
        </div>
      )}
    </section>
  );
}
