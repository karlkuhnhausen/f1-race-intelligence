import { useEffect, useState } from 'react';

export interface CountdownValues {
  days: number;
  hours: number;
  minutes: number;
  seconds: number;
  total_ms: number;
  expired: boolean;
}

/**
 * useCountdown — returns a live countdown that ticks every second.
 * @param targetUTC ISO 8601 timestamp or null if no target.
 */
export function useCountdown(targetUTC: string | null): CountdownValues | null {
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    if (!targetUTC) return;
    const id = setInterval(() => setNow(Date.now()), 1_000);
    return () => clearInterval(id);
  }, [targetUTC]);

  if (!targetUTC) return null;

  const targetMs = new Date(targetUTC).getTime();
  const diffMs = targetMs - now;

  if (diffMs <= 0) {
    return { days: 0, hours: 0, minutes: 0, seconds: 0, total_ms: 0, expired: true };
  }

  const days = Math.floor(diffMs / (1000 * 60 * 60 * 24));
  const hours = Math.floor((diffMs % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
  const minutes = Math.floor((diffMs % (1000 * 60 * 60)) / (1000 * 60));
  const seconds = Math.floor((diffMs % (1000 * 60)) / 1000);

  return { days, hours, minutes, seconds, total_ms: diffMs, expired: false };
}
