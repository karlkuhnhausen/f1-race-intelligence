import type { SessionResultEntry } from './roundApi';
import LapTimeDisplay from '../design-system/LapTimeDisplay';
import { getTeamColor } from '../design-system/teamColors';

interface Props {
  results: SessionResultEntry[];
  sessionType: string;
}

const RACE_TYPES = new Set(['race', 'sprint']);
const QUALIFYING_TYPES = new Set(['qualifying', 'sprint_qualifying']);
const PRACTICE_TYPES = new Set(['practice1', 'practice2', 'practice3']);

const NON_CLASSIFIED_STATUSES = new Set([
  'DNF',
  'DNS',
  'DSQ',
  'Retired',
  'Disqualified',
]);

const headerLeft =
  'px-3 py-3 text-left font-display text-xs uppercase tracking-wider text-muted-foreground';
const headerRight =
  'px-3 py-3 text-right font-display text-xs uppercase tracking-wider text-muted-foreground';

const cellMonoRight = 'px-3 py-3 text-right font-mono tabular-nums';
const cellLeft = 'px-3 py-3';

function podiumBorder(position: number, isRace: boolean): string {
  if (!isRace) return '';
  switch (position) {
    case 1:
      return 'border-l-2 border-l-amber-400';
    case 2:
      return 'border-l-2 border-l-zinc-300';
    case 3:
      return 'border-l-2 border-l-amber-700';
    default:
      return '';
  }
}

function TeamCell({ teamName }: { teamName: string }) {
  const color = getTeamColor(teamName);
  return (
    <td className={cellLeft}>
      <div className="flex items-center gap-2 min-w-0">
        <span
          aria-hidden
          data-testid="team-color-swatch"
          className="inline-block h-4 w-1 shrink-0 rounded-sm"
          style={{ backgroundColor: color }}
        />
        <span className="truncate text-muted-foreground">{teamName}</span>
      </div>
    </td>
  );
}

function DriverCell({ acronym, name }: { acronym: string; name: string }) {
  return (
    <td className={`${cellLeft} whitespace-nowrap`}>
      <span className="font-mono font-bold text-accent-cyan mr-2">{acronym}</span>
      <span className="font-display">{name}</span>
    </td>
  );
}

interface RowProps {
  r: SessionResultEntry;
  idx: number;
  isRace: boolean;
  isQualifying: boolean;
  isPractice: boolean;
  nonClassified?: boolean;
}

function ResultRow({
  r,
  idx,
  isRace,
  isQualifying,
  isPractice,
  nonClassified,
}: RowProps) {
  const baseTint = r.fastest_lap
    ? 'bg-accent-cyan/5'
    : idx % 2 === 1
      ? 'bg-background/30'
      : '';
  const dimmed = nonClassified ? 'opacity-60' : '';

  return (
    <tr className={`border-b border-border last:border-0 ${baseTint} ${dimmed}`}>
      <td
        className={`${cellMonoRight} font-bold text-foreground ${podiumBorder(
          r.position,
          isRace && !nonClassified
        )}`}
      >
        {r.position}
      </td>
      <td className={`${cellMonoRight} text-muted-foreground`}>{r.driver_number}</td>
      <DriverCell acronym={r.driver_acronym} name={r.driver_name} />
      <TeamCell teamName={r.team_name} />
      {isRace && (
        <>
          <td className={`${cellLeft} text-muted-foreground`}>
            {r.finishing_status ?? '—'}
          </td>
          <td className={`${cellMonoRight} text-foreground`}>
            {r.gap_to_leader ?? '—'}
          </td>
          <td className={`${cellMonoRight} text-foreground`}>{r.points ?? 0}</td>
        </>
      )}
      {isQualifying && (
        <>
          <td className={cellMonoRight}>
            <LapTimeDisplay time={r.q1_time} />
          </td>
          <td className={cellMonoRight}>
            <LapTimeDisplay time={r.q2_time} />
          </td>
          <td className={cellMonoRight}>
            <LapTimeDisplay time={r.q3_time} />
          </td>
        </>
      )}
      {isPractice && (
        <>
          <td className={cellMonoRight}>
            <LapTimeDisplay time={r.best_lap_time} />
          </td>
          <td className={cellMonoRight}>
            {r.gap_to_fastest != null ? (
              <LapTimeDisplay delta={-r.gap_to_fastest} deltaOnly />
            ) : (
              <span className="font-mono text-muted-foreground">—</span>
            )}
          </td>
        </>
      )}
    </tr>
  );
}

export default function SessionResultsTable({ results, sessionType }: Props) {
  const isRace = RACE_TYPES.has(sessionType);
  const isQualifying = QUALIFYING_TYPES.has(sessionType);
  const isPractice = PRACTICE_TYPES.has(sessionType);

  // Race sessions split classified from DNF/DNS/DSQ rows so the divider
  // mirrors the legacy RaceResults UX.
  let classified = results;
  let nonClassified: SessionResultEntry[] = [];
  if (isRace) {
    classified = results.filter(
      (r) => !NON_CLASSIFIED_STATUSES.has(r.finishing_status ?? '')
    );
    nonClassified = results.filter((r) =>
      NON_CLASSIFIED_STATUSES.has(r.finishing_status ?? '')
    );
  }

  const totalCols = isRace ? 7 : isQualifying ? 7 : isPractice ? 6 : 4;

  return (
    <div className="overflow-x-auto rounded-md border border-border">
      <table className="w-full text-sm table-fixed">
        <colgroup>
          <col className="w-14" />
          <col className="w-14" />
          <col className="w-56" />
          <col className="w-[22%]" />
          {isRace && (
            <>
              <col className="w-28" />
              <col className="w-24" />
              <col className="w-16" />
            </>
          )}
          {isQualifying && (
            <>
              <col />
              <col />
              <col />
            </>
          )}
          {isPractice && (
            <>
              <col className="w-32" />
              <col className="w-24" />
            </>
          )}
        </colgroup>
        <thead className="border-b border-border bg-background/50">
          <tr>
            <th className={headerRight}>Pos</th>
            <th className={headerRight}>#</th>
            <th className={headerLeft}>Driver</th>
            <th className={headerLeft}>Team</th>
            {isRace && (
              <>
                <th className={headerLeft}>Status</th>
                <th className={headerRight}>Gap</th>
                <th className={headerRight}>Pts</th>
              </>
            )}
            {isQualifying && (
              <>
                <th className={headerRight}>Q1</th>
                <th className={headerRight}>Q2</th>
                <th className={headerRight}>Q3</th>
              </>
            )}
            {isPractice && (
              <>
                <th className={headerRight}>Best Lap</th>
                <th className={headerRight}>Gap</th>
              </>
            )}
          </tr>
        </thead>
        <tbody>
          {classified.map((r, idx) => (
            <ResultRow
              key={r.driver_number}
              r={r}
              idx={idx}
              isRace={isRace}
              isQualifying={isQualifying}
              isPractice={isPractice}
            />
          ))}
          {isRace && nonClassified.length > 0 && (
            <>
              <tr className="bg-background/60">
                <td
                  colSpan={totalCols}
                  className="px-3 py-2 font-display text-xs uppercase tracking-wider text-muted-foreground"
                >
                  Not Classified
                </td>
              </tr>
              {nonClassified.map((r, idx) => (
                <ResultRow
                  key={r.driver_number}
                  r={r}
                  idx={idx}
                  isRace={isRace}
                  isQualifying={isQualifying}
                  isPractice={isPractice}
                  nonClassified
                />
              ))}
            </>
          )}
        </tbody>
      </table>
    </div>
  );
}
