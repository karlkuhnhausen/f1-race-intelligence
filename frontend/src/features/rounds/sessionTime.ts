import type { SessionDetail } from './roundApi';

/**
 * Formats an ISO 8601 UTC timestamp as a localized date+time string for
 * display, e.g. "Sat, Mar 15, 9:42 PM PDT". Uses the browser's locale and
 * timezone via Intl. Returns '' for empty or invalid input.
 */
export function formatLocalDateTime(iso: string): string {
  if (!iso) return '';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  return d.toLocaleString(undefined, {
    weekday: 'short',
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
    timeZoneName: 'short',
  });
}

/**
 * Returns true if the round is "this race weekend" — meaning at least one
 * session is currently in_progress, OR the earliest non-completed session
 * starts within `windowDays` of `now`. Used to gate per-session countdowns
 * on the Round Detail page so we don't decorate sessions for races months
 * away.
 */
export function isWithinWeekendWindow(
  sessions: SessionDetail[],
  now: number = Date.now(),
  windowDays: number = 7,
): boolean {
  const nonCompleted = sessions.filter((s) => s.status !== 'completed');
  if (nonCompleted.length === 0) return false;
  if (nonCompleted.some((s) => s.status === 'in_progress')) return true;

  const startTimes = nonCompleted
    .map((s) => new Date(s.date_start_utc).getTime())
    .filter((t) => !Number.isNaN(t));
  if (startTimes.length === 0) return false;

  const earliest = Math.min(...startTimes);
  const windowMs = windowDays * 24 * 60 * 60 * 1000;
  return earliest - now <= windowMs;
}
