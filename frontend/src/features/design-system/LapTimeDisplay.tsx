import { cn } from "@/lib/utils";

export interface LapTimeDisplayProps {
  /** Lap time in seconds (e.g., 87.654). Ignored when deltaOnly is true. */
  time?: number;
  /** Time delta in seconds; positive = faster, negative = slower, zero = neutral */
  delta?: number;
  /** If true, display only the delta (not the absolute time) */
  deltaOnly?: boolean;
  /** Optional className passthrough */
  className?: string;
}

function formatLapTime(seconds: number): string {
  if (!Number.isFinite(seconds)) return "—";
  const mins = Math.floor(seconds / 60);
  const secs = (seconds % 60).toFixed(3);
  return mins > 0 ? `${mins}:${secs.padStart(6, "0")}` : secs;
}

function formatDelta(delta: number): string {
  const abs = Math.abs(delta).toFixed(3);
  if (delta > 0) return `+${abs}`;
  if (delta < 0) return `−${abs}`;
  return abs;
}

function deltaColorClass(delta: number): string {
  if (delta > 0) return "text-positive";
  if (delta < 0) return "text-negative";
  return "text-foreground";
}

export default function LapTimeDisplay({
  time,
  delta,
  deltaOnly = false,
  className,
}: LapTimeDisplayProps) {
  if (deltaOnly) {
    if (delta === undefined) {
      return (
        <span className={cn("font-mono text-muted-foreground", className)}>
          —
        </span>
      );
    }
    return (
      <span
        data-testid="lap-time-delta"
        className={cn("font-mono", deltaColorClass(delta), className)}
      >
        {formatDelta(delta)}
      </span>
    );
  }

  return (
    <span
      data-testid="lap-time-display"
      className={cn("inline-flex items-baseline gap-2 font-mono", className)}
    >
      <span className="text-foreground">
        {time !== undefined ? formatLapTime(time) : "—"}
      </span>
      {delta !== undefined && (
        <span data-testid="lap-time-delta" className={deltaColorClass(delta)}>
          {formatDelta(delta)}
        </span>
      )}
    </span>
  );
}
