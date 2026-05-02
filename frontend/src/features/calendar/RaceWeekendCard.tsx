import type { ActiveSessionDTO, RaceMeetingDTO } from './calendarApi';
import RaceCountdown from '../design-system/RaceCountdown';

interface RaceWeekendCardProps {
  round: RaceMeetingDTO;
  session: ActiveSessionDTO;
}

const SESSION_LABELS: Record<string, string> = {
  practice1: 'Practice 1',
  practice2: 'Practice 2',
  practice3: 'Practice 3',
  sprint_qualifying: 'Sprint Qualifying',
  sprint: 'Sprint',
  qualifying: 'Qualifying',
  race: 'Race',
};

function sessionLabel(session: ActiveSessionDTO): string {
  return (
    SESSION_LABELS[session.session_type] ??
    session.session_name ??
    session.session_type
  );
}

export default function RaceWeekendCard({ round, session }: RaceWeekendCardProps) {
  const label = sessionLabel(session);
  const isLive = session.status === 'in_progress';
  const isCompleted = session.status === 'completed';

  // Pick a countdown target appropriate for the session state:
  //  - upcoming    -> count down to session start
  //  - in_progress -> count down to session end (RaceCountdown shows "RACE LIVE" when expired)
  //  - completed   -> no countdown (weekend wrapping up)
  const countdownTarget = isLive
    ? session.date_end_utc
    : isCompleted
      ? null
      : session.date_start_utc;

  const countdownLabel = isLive
    ? `${label} live`
    : isCompleted
      ? `${label} complete`
      : `until ${label.toLowerCase()}`;

  const eyebrow = isLive ? 'RACE WEEKEND · LIVE' : 'RACE WEEKEND';

  return (
    <div
      className="rounded-lg border border-border bg-surface p-6 shadow-lg"
      role="region"
      aria-label="Race weekend status"
      data-testid="race-weekend-card"
      data-session-status={session.status}
    >
      <p className="font-display text-xs uppercase tracking-[0.2em] text-accent-red">
        {eyebrow}
      </p>
      <h3 className="mt-2 font-display text-2xl font-bold tracking-tight">
        {round.race_name}
      </h3>
      <p className="text-muted-foreground">
        {round.circuit_name}, {round.country_name}
      </p>
      <p className="mt-3 font-display text-sm uppercase tracking-wider text-muted-foreground">
        {isLive ? 'Now' : isCompleted ? 'Just finished' : 'Up next'}
      </p>
      <p className="font-display text-xl font-bold text-foreground">{label}</p>
      {countdownTarget ? (
        <div className="mt-4">
          <RaceCountdown targetUtc={countdownTarget} label={countdownLabel} />
        </div>
      ) : (
        <p className="mt-4 text-xs uppercase tracking-wider text-muted-foreground font-display">
          {countdownLabel}
        </p>
      )}
    </div>
  );
}
