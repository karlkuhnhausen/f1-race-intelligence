import { getTeamColor } from "./teamColors";
import { cn } from "@/lib/utils";

export interface DriverCardProps {
  /** Driver's display name (e.g., "Max Verstappen") */
  name: string;
  /** Driver's racing number */
  number: number;
  /** Constructor/team identifier for color lookup (e.g., "redbull") */
  constructorId: string;
  /** Current championship position */
  position: number;
  /** Championship points total */
  points: number;
  /** Gap to leader as formatted string (e.g., "+12.5s" or "LEADER") */
  gap?: string;
  /** Optional className passthrough */
  className?: string;
}

export default function DriverCard({
  name,
  number,
  constructorId,
  position,
  points,
  gap,
  className,
}: DriverCardProps) {
  const teamColor = getTeamColor(constructorId);

  return (
    <div
      data-testid="driver-card"
      className={cn(
        "relative flex items-center gap-4 rounded-md bg-surface px-4 py-3 pl-5 shadow-sm",
        className
      )}
      style={{ borderLeft: `4px solid ${teamColor}` }}
    >
      <div className="flex flex-col items-center justify-center">
        <span className="font-mono text-3xl font-bold leading-none text-foreground">
          P{position}
        </span>
        <span className="mt-1 font-mono text-xs text-muted-foreground">
          #{number}
        </span>
      </div>
      <div className="flex-1 min-w-0">
        <p className="truncate font-display text-base font-bold tracking-tight text-foreground">
          {name}
        </p>
        <p className="text-xs uppercase tracking-wider text-muted-foreground">
          {constructorId.replace(/_/g, " ")}
        </p>
      </div>
      <div className="flex flex-col items-end">
        <span className="font-mono text-lg font-bold leading-none text-foreground">
          {points}
          <span className="ml-1 text-xs font-normal text-muted-foreground">
            pts
          </span>
        </span>
        {gap !== undefined && gap !== "" && (
          <span className="mt-1 font-mono text-xs text-muted-foreground">
            {gap}
          </span>
        )}
      </div>
    </div>
  );
}
