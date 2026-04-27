import { cn } from "@/lib/utils";
import { useCountdown } from "@/features/calendar/useCountdown";

export interface RaceCountdownProps {
  /** ISO 8601 UTC datetime string of the countdown target */
  targetUtc: string;
  /** Label shown next to the countdown (e.g., "until lights out") */
  label?: string;
  /** Optional className passthrough */
  className?: string;
}

export default function RaceCountdown({
  targetUtc,
  label,
  className,
}: RaceCountdownProps) {
  const countdown = useCountdown(targetUtc);

  if (!countdown || countdown.expired) {
    return (
      <div
        data-testid="race-countdown"
        data-state="expired"
        className={cn("font-display", className)}
      >
        <p className="text-2xl font-bold uppercase tracking-wider text-accent-red">
          RACE LIVE
        </p>
        {label && (
          <p className="mt-1 text-xs uppercase tracking-wider text-muted-foreground">
            {label}
          </p>
        )}
      </div>
    );
  }

  return (
    <div
      data-testid="race-countdown"
      data-state="counting"
      className={cn(className)}
    >
      <p
        className="flex flex-wrap items-baseline gap-3 font-mono text-3xl font-bold text-accent-cyan"
        aria-live="polite"
      >
        {countdown.days > 0 && <span>{countdown.days}d</span>}
        <span>{countdown.hours}h</span>
        <span>{countdown.minutes}m</span>
        <span>{countdown.seconds}s</span>
      </p>
      {label && (
        <p className="mt-2 text-xs font-display uppercase tracking-wider text-muted-foreground">
          {label}
        </p>
      )}
    </div>
  );
}
