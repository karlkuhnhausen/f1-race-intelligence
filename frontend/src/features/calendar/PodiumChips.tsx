import type { PodiumEntryDTO } from './calendarApi';
import { getTeamColor } from '../design-system/teamColors';

interface Props {
  entries?: PodiumEntryDTO[];
}

/**
 * Compact podium display for a completed race row in the calendar table.
 * Renders up to three chips (P1/P2/P3) with a team color left bar, the
 * driver acronym, and the driver's current season championship points.
 *
 * Returns an em dash when no podium data is available (upcoming races,
 * or completed races whose results haven't been ingested yet).
 */
export default function PodiumChips({ entries }: Props) {
  if (!entries || entries.length === 0) {
    return <span className="text-muted-foreground" aria-hidden>—</span>;
  }

  return (
    <div className="flex flex-wrap items-center gap-1.5" data-testid="podium-chips">
      {entries.map((e) => (
        <PodiumChip key={e.position} entry={e} />
      ))}
    </div>
  );
}

function PodiumChip({ entry }: { entry: PodiumEntryDTO }) {
  const color = getTeamColor(entry.team_name);
  const positionLabel = `P${entry.position}`;
  const ariaLabel = `${positionLabel} ${entry.driver_name}, ${entry.team_name}, ${entry.season_points} season points`;

  return (
    <span
      className="inline-flex items-center gap-1.5 rounded-md border border-border bg-background/40 pl-1.5 pr-2 py-0.5 text-xs leading-none"
      title={`${positionLabel} • ${entry.driver_name} • ${entry.team_name} • ${entry.season_points} pts`}
      aria-label={ariaLabel}
      data-testid={`podium-chip-p${entry.position}`}
    >
      <span
        aria-hidden
        data-testid="podium-chip-color"
        className="inline-block h-3 w-1 shrink-0 rounded-sm"
        style={{ backgroundColor: color }}
      />
      <span className="font-display text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
        {positionLabel}
      </span>
      <span className="font-mono font-bold text-accent-cyan">{entry.driver_acronym}</span>
      <span className="font-mono tabular-nums text-foreground">
        {Math.round(entry.season_points)}
        <span className="ml-0.5 text-muted-foreground">pts</span>
      </span>
    </span>
  );
}
