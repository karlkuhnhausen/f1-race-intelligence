import type { SessionResultEntry } from './roundApi';
import LapTimeDisplay from '../design-system/LapTimeDisplay';

interface Props {
  results: SessionResultEntry[];
  sessionType: string;
}

const RACE_TYPES = new Set(['race', 'sprint']);
const QUALIFYING_TYPES = new Set(['qualifying', 'sprint_qualifying']);
const PRACTICE_TYPES = new Set(['practice1', 'practice2', 'practice3']);

const headerCellClass =
  'px-3 py-2 text-left font-display text-xs uppercase tracking-wider text-muted-foreground';

export default function SessionResultsTable({ results, sessionType }: Props) {
  const isRace = RACE_TYPES.has(sessionType);
  const isQualifying = QUALIFYING_TYPES.has(sessionType);
  const isPractice = PRACTICE_TYPES.has(sessionType);

  // For practice sessions, the gap is rendered relative to the fastest lap.
  // We use LapTimeDisplay's deltaOnly mode with a negative sign (slower than leader).

  return (
    <div className="overflow-hidden rounded-md border border-border">
      <table className="w-full text-sm">
        <thead className="border-b border-border bg-background/50">
          <tr>
            <th className={headerCellClass}>Pos</th>
            <th className={headerCellClass}>#</th>
            <th className={headerCellClass}>Driver</th>
            <th className={headerCellClass}>Team</th>
            {isRace && (
              <>
                <th className={headerCellClass}>Status</th>
                <th className={headerCellClass}>Gap</th>
                <th className={headerCellClass}>Pts</th>
              </>
            )}
            {isQualifying && (
              <>
                <th className={headerCellClass}>Q1</th>
                <th className={headerCellClass}>Q2</th>
                <th className={headerCellClass}>Q3</th>
              </>
            )}
            {isPractice && (
              <>
                <th className={headerCellClass}>Best Lap</th>
                <th className={headerCellClass}>Gap</th>
              </>
            )}
          </tr>
        </thead>
        <tbody>
          {results.map((r, idx) => (
            <tr
              key={r.driver_number}
              className={`border-b border-border last:border-0 ${
                r.fastest_lap ? 'bg-accent-cyan/5' : idx % 2 === 1 ? 'bg-background/30' : ''
              }`}
            >
              <td className="px-3 py-2 font-mono font-bold text-foreground">{r.position}</td>
              <td className="px-3 py-2 font-mono text-muted-foreground">{r.driver_number}</td>
              <td className="px-3 py-2">
                <span className="font-mono font-bold text-accent-cyan mr-2">
                  {r.driver_acronym}
                </span>
                <span className="font-display">{r.driver_name}</span>
              </td>
              <td className="px-3 py-2 text-muted-foreground">{r.team_name}</td>
              {isRace && (
                <>
                  <td className="px-3 py-2 text-muted-foreground">{r.finishing_status ?? '—'}</td>
                  <td className="px-3 py-2 font-mono text-foreground">{r.gap_to_leader ?? '—'}</td>
                  <td className="px-3 py-2 font-mono text-foreground">{r.points ?? 0}</td>
                </>
              )}
              {isQualifying && (
                <>
                  <td className="px-3 py-2"><LapTimeDisplay time={r.q1_time} /></td>
                  <td className="px-3 py-2"><LapTimeDisplay time={r.q2_time} /></td>
                  <td className="px-3 py-2"><LapTimeDisplay time={r.q3_time} /></td>
                </>
              )}
              {isPractice && (
                <>
                  <td className="px-3 py-2"><LapTimeDisplay time={r.best_lap_time} /></td>
                  <td className="px-3 py-2">
                    {r.gap_to_fastest != null ? (
                      <LapTimeDisplay
                        delta={-r.gap_to_fastest}
                        deltaOnly
                      />
                    ) : (
                      <span className="font-mono text-muted-foreground">—</span>
                    )}
                  </td>
                </>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
