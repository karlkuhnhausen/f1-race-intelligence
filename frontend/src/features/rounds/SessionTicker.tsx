import { cn } from '@/lib/utils';
import { useCountdown } from '@/features/calendar/useCountdown';
import { useElapsed } from './useElapsed';

export interface SessionTickerProps {
  /** ISO 8601 UTC datetime: session start (countdown target or elapsed origin). */
  targetUtc: string;
  /** countdown = count DOWN to target; elapsed = count UP from target. */
  mode: 'countdown' | 'elapsed';
  /** Small uppercase caption rendered below the digits. */
  label: string;
  /** Optional className passthrough. */
  className?: string;
}

function pad(n: number): string {
  return n.toString().padStart(2, '0');
}

/**
 * SessionTicker — digital HH:MM:SS ticker used on the Round Detail page.
 *
 * `countdown` mode renders time remaining until `targetUtc` and is unbounded
 * in hours (e.g. 71:23:45 when a session is ~3 days out). When the target
 * passes, it renders 00:00:00 — the page should switch to `elapsed` mode
 * once the backend marks the session in_progress on the next refresh.
 *
 * `elapsed` mode renders time since `targetUtc`. Before the start time the
 * ticker shows 00:00:00.
 */
export default function SessionTicker({
  targetUtc,
  mode,
  label,
  className,
}: SessionTickerProps) {
  const countdown = useCountdown(mode === 'countdown' ? targetUtc : null);
  const elapsed = useElapsed(mode === 'elapsed' ? targetUtc : null);

  let h = 0;
  let m = 0;
  let s = 0;

  if (mode === 'countdown' && countdown) {
    if (countdown.expired) {
      h = 0;
      m = 0;
      s = 0;
    } else {
      // useCountdown returns days separately; collapse into total hours so
      // the ticker is unbounded H:MM:SS for sessions that are days out.
      h = countdown.days * 24 + countdown.hours;
      m = countdown.minutes;
      s = countdown.seconds;
    }
  } else if (mode === 'elapsed' && elapsed) {
    h = elapsed.hours;
    m = elapsed.minutes;
    s = elapsed.seconds;
  }

  const digits = `${pad(h)}:${pad(m)}:${pad(s)}`;
  const colorClass =
    mode === 'elapsed' ? 'text-accent-red' : 'text-accent-cyan';

  return (
    <div
      data-testid="session-ticker"
      data-mode={mode}
      className={cn(className)}
    >
      <p
        className={cn(
          'font-mono text-3xl font-bold tabular-nums',
          colorClass,
        )}
        aria-live="polite"
      >
        {digits}
      </p>
      <p className="mt-2 text-xs font-display uppercase tracking-wider text-muted-foreground">
        {label}
      </p>
    </div>
  );
}
