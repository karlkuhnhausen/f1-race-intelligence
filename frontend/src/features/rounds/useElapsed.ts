import { useEffect, useState } from 'react';

export interface ElapsedValues {
  hours: number;
  minutes: number;
  seconds: number;
  total_ms: number;
  /** True once the start timestamp is in the past. */
  started: boolean;
}

/**
 * useElapsed — returns a live elapsed-time counter that ticks every second.
 * Counts UP from `startUTC`. While the start is still in the future, returns
 * zeros with `started=false` so the caller can render `00:00:00` rather than
 * a negative value. Hours are unbounded (no day rollover), since this is
 * intended for a session-elapsed stopwatch where sessions are at most a few
 * hours long.
 */
export function useElapsed(startUTC: string | null): ElapsedValues | null {
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    if (!startUTC) return;
    const id = setInterval(() => setNow(Date.now()), 1_000);
    return () => clearInterval(id);
  }, [startUTC]);

  if (!startUTC) return null;

  const startMs = new Date(startUTC).getTime();
  if (Number.isNaN(startMs)) return null;

  const diffMs = now - startMs;
  if (diffMs <= 0) {
    return { hours: 0, minutes: 0, seconds: 0, total_ms: 0, started: false };
  }

  const hours = Math.floor(diffMs / (1000 * 60 * 60));
  const minutes = Math.floor((diffMs % (1000 * 60 * 60)) / (1000 * 60));
  const seconds = Math.floor((diffMs % (1000 * 60)) / 1000);

  return { hours, minutes, seconds, total_ms: diffMs, started: true };
}
