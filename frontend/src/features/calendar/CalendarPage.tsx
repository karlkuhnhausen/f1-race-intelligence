import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { fetchCalendar, type CalendarResponse, type RaceMeetingDTO } from './calendarApi';
import NextRaceCard from './NextRaceCard';
import RaceWeekendCard from './RaceWeekendCard';

export default function CalendarPage() {
  const [calendar, setCalendar] = useState<CalendarResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchCalendar(2026)
      .then(setCalendar)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <div className="text-muted-foreground">Loading calendar…</div>;
  if (error) return <div className="text-negative">Error: {error}</div>;
  if (!calendar) return null;

  return (
    <section className="space-y-6">
      <div>
        <h2 className="font-display text-3xl font-bold tracking-tight">
          2026 FIA Formula 1 World Championship
        </h2>
        <p className="mt-1 text-sm text-muted-foreground font-mono">
          Data as of: {new Date(calendar.data_as_of_utc).toLocaleString()}
        </p>
      </div>

      {(() => {
        if (calendar.next_round <= 0) return null;
        const nextRound = calendar.rounds.find((r) => r.round === calendar.next_round);
        if (!nextRound) return null;

        if (calendar.weekend_in_progress && calendar.active_session) {
          return <RaceWeekendCard round={nextRound} session={calendar.active_session} />;
        }

        if (calendar.countdown_target_utc) {
          return (
            <NextRaceCard round={nextRound} countdownTarget={calendar.countdown_target_utc} />
          );
        }

        return null;
      })()}

      <div className="overflow-hidden rounded-lg border border-border bg-surface">
        <table className="w-full text-sm">
          <thead className="border-b border-border bg-background/50">
            <tr>
              <th className="px-4 py-3 text-left font-display text-xs uppercase tracking-wider text-muted-foreground">Round</th>
              <th className="px-4 py-3 text-left font-display text-xs uppercase tracking-wider text-muted-foreground">Race</th>
              <th className="px-4 py-3 text-left font-display text-xs uppercase tracking-wider text-muted-foreground">Circuit</th>
              <th className="px-4 py-3 text-left font-display text-xs uppercase tracking-wider text-muted-foreground">Country</th>
              <th className="px-4 py-3 text-left font-display text-xs uppercase tracking-wider text-muted-foreground">Date</th>
              <th className="px-4 py-3 text-left font-display text-xs uppercase tracking-wider text-muted-foreground">Status</th>
            </tr>
          </thead>
          <tbody>
            {calendar.rounds.map((round) => (
              <RaceRow key={round.round} round={round} isNext={round.round === calendar.next_round} />
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function RaceRow({ round, isNext }: { round: RaceMeetingDTO; isNext: boolean }) {
  const baseRow = 'border-b border-border last:border-0 transition-colors';
  const stateClass = round.is_cancelled
    ? 'opacity-50 line-through'
    : isNext
      ? 'bg-accent-cyan/5 hover:bg-accent-cyan/10'
      : 'hover:bg-background/40';

  return (
    <tr className={`${baseRow} ${stateClass}`}>
      <td className="px-4 py-3 font-mono text-foreground">{round.round}</td>
      <td className="px-4 py-3">
        {round.is_cancelled ? (
          <span>{round.race_name}</span>
        ) : (
          <Link
            to={`/rounds/${round.round}?year=2026`}
            className="font-display font-bold text-foreground hover:text-accent-cyan transition-colors"
          >
            {round.race_name}
          </Link>
        )}
      </td>
      <td className="px-4 py-3 text-muted-foreground">{round.circuit_name}</td>
      <td className="px-4 py-3 text-muted-foreground">{round.country_name}</td>
      <td className="px-4 py-3 font-mono text-muted-foreground">
        {new Date(round.start_datetime_utc).toLocaleDateString()}
      </td>
      <td className="px-4 py-3">
        {round.is_cancelled ? (
          <span className="cancelled inline-flex items-center rounded-md bg-negative/20 px-2 py-0.5 text-xs font-display font-bold uppercase tracking-wider text-negative">
            {round.cancelled_label ?? 'Cancelled'}
          </span>
        ) : (
          <span
            className={`inline-flex items-center rounded-md px-2 py-0.5 text-xs font-display font-bold uppercase tracking-wider ${round.status} ${
              round.status === 'scheduled'
                ? 'bg-surface text-muted-foreground border border-border'
                : 'bg-positive/20 text-positive'
            }`}
          >
            {round.status}
          </span>
        )}
      </td>
    </tr>
  );
}
