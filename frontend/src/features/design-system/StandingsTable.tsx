import { getTeamColor } from "./teamColors";
import { cn } from "@/lib/utils";

export interface StandingsRow {
  position: number;
  name: string;
  constructorId: string;
  points: number;
  wins?: number;
  podiums?: number;
  dnfs?: number;
  poles?: number;
  /** Direct team color from API (hex without #). Falls back to constructorId lookup. */
  teamColor?: string;
}

export interface StandingsTableProps {
  /** Table heading */
  title: string;
  /** Ordered rows of standings data */
  rows: StandingsRow[];
  /** Column configuration — which optional columns to show */
  columns?: ("wins" | "podiums" | "dnfs" | "poles")[];
  /** Optional className passthrough */
  className?: string;
  /**
   * Label shown for the "name" column header. Defaults to "Driver" but can
   * be set to "Team" / "Constructor" for constructor standings tables.
   */
  nameLabel?: string;
}

const headerCellClass =
  "px-4 py-3 text-left font-display text-xs uppercase tracking-wider text-muted-foreground";

export default function StandingsTable({
  title,
  rows,
  columns = [],
  className,
  nameLabel = "Driver",
}: StandingsTableProps) {
  const showWins = columns.includes("wins");
  const showPodiums = columns.includes("podiums");
  const showDNFs = columns.includes("dnfs");
  const showPoles = columns.includes("poles");
  const colCount = 3 + [showWins, showPodiums, showDNFs, showPoles].filter(Boolean).length;

  return (
    <div
      data-testid="standings-table"
      className={cn(
        "overflow-hidden rounded-lg border border-border bg-surface",
        className
      )}
    >
      <div className="border-b border-border bg-background/50 px-4 py-3">
        <h3 className="font-display text-sm font-bold uppercase tracking-wider text-foreground">
          {title}
        </h3>
      </div>
      <table className="w-full text-sm">
        <thead className="border-b border-border bg-background/30">
          <tr>
            <th className={headerCellClass}>Pos</th>
            <th className={headerCellClass}>{nameLabel}</th>
            <th className={headerCellClass}>Points</th>
            {showWins && <th className={headerCellClass}>Wins</th>}
            {showPodiums && <th className={headerCellClass}>Podiums</th>}
            {showDNFs && <th className={headerCellClass}>DNFs</th>}
            {showPoles && <th className={headerCellClass}>Poles</th>}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, idx) => {
            const teamColor = row.teamColor
              ? `#${row.teamColor}`
              : getTeamColor(row.constructorId);
            return (
              <tr
                key={`${row.position}-${row.name}`}
                data-testid="standings-row"
                className={cn(
                  "border-b border-border last:border-0 transition-colors",
                  idx % 2 === 1 ? "bg-background/30" : "bg-surface"
                )}
                style={{ borderLeft: `3px solid ${teamColor}` }}
              >
                <td className="px-4 py-3 font-mono font-bold text-foreground">
                  {row.position}
                </td>
                <td className="px-4 py-3 font-display text-foreground">
                  {row.name}
                </td>
                <td className="px-4 py-3 font-mono text-foreground">
                  {row.points}
                </td>
                {showWins && (
                  <td className="px-4 py-3 font-mono text-muted-foreground">
                    {row.wins ?? 0}
                  </td>
                )}
                {showPodiums && (
                  <td className="px-4 py-3 font-mono text-muted-foreground">
                    {row.podiums ?? 0}
                  </td>
                )}
                {showDNFs && (
                  <td className="px-4 py-3 font-mono text-muted-foreground">
                    {row.dnfs ?? 0}
                  </td>
                )}
                {showPoles && (
                  <td className="px-4 py-3 font-mono text-muted-foreground">
                    {row.poles ?? 0}
                  </td>
                )}
              </tr>
            );
          })}
          {rows.length === 0 && (
            <tr>
              <td
                colSpan={colCount}
                className="px-4 py-6 text-center text-muted-foreground"
              >
                No standings available yet.
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
