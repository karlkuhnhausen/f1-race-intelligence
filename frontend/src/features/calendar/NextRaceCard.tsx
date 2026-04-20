import type { RaceMeetingDTO } from './calendarApi';
import { useCountdown } from './useCountdown';

interface NextRaceCardProps {
  round: RaceMeetingDTO;
  countdownTarget: string;
}

export default function NextRaceCard({ round, countdownTarget }: NextRaceCardProps) {
  const countdown = useCountdown(countdownTarget);

  return (
    <div className="next-race-card" role="region" aria-label="Next race countdown">
      <h3>Next Race</h3>
      <p className="race-name">{round.race_name}</p>
      <p className="circuit">
        {round.circuit_name}, {round.country_name}
      </p>
      <p className="race-date">
        {new Date(round.start_datetime_utc).toLocaleDateString(undefined, {
          weekday: 'long',
          year: 'numeric',
          month: 'long',
          day: 'numeric',
        })}
      </p>
      {countdown && !countdown.expired && (
        <p className="countdown" aria-live="polite">
          {countdown.days > 0 && <span className="countdown-segment">{countdown.days}d </span>}
          <span className="countdown-segment">{countdown.hours}h </span>
          <span className="countdown-segment">{countdown.minutes}m </span>
          <span className="countdown-segment">{countdown.seconds}s</span>
          <span className="countdown-label"> until lights out</span>
        </p>
      )}
      {countdown?.expired && (
        <p className="countdown expired">Race underway!</p>
      )}
    </div>
  );
}
