import type { RaceMeetingDTO } from './calendarApi';
import RaceCountdown from '../design-system/RaceCountdown';

interface NextRaceCardProps {
  round: RaceMeetingDTO;
  countdownTarget: string;
}

export default function NextRaceCard({ round, countdownTarget }: NextRaceCardProps) {
  return (
    <div
      className="rounded-lg border border-border bg-surface p-6 shadow-lg"
      role="region"
      aria-label="Next race countdown"
    >
      <p className="font-display text-xs uppercase tracking-[0.2em] text-accent-red">
        Next Race
      </p>
      <h3 className="mt-2 font-display text-2xl font-bold tracking-tight">
        {round.race_name}
      </h3>
      <p className="text-muted-foreground">
        {round.circuit_name}, {round.country_name}
      </p>
      <p className="mt-1 text-sm text-muted-foreground font-mono">
        {new Date(round.start_datetime_utc).toLocaleDateString(undefined, {
          weekday: 'long',
          year: 'numeric',
          month: 'long',
          day: 'numeric',
        })}
      </p>
      <div className="mt-4">
        <RaceCountdown targetUtc={countdownTarget} label="until lights out" />
      </div>
    </div>
  );
}
