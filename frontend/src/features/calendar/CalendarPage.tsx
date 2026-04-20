import { useEffect, useState } from 'react';
import { fetchCalendar, type CalendarResponse, type RaceMeetingDTO } from './calendarApi';

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

  if (loading) return <div className="loading">Loading calendar…</div>;
  if (error) return <div className="error">Error: {error}</div>;
  if (!calendar) return null;

  return (
    <section className="calendar-page">
      <h2>2026 FIA Formula 1 World Championship</h2>
      <p className="data-freshness">
        Data as of: {new Date(calendar.data_as_of_utc).toLocaleString()}
      </p>

      {calendar.next_round > 0 && calendar.countdown_target_utc && (
        <NextRaceHighlight
          round={calendar.rounds.find((r) => r.round === calendar.next_round)!}
          countdownTarget={calendar.countdown_target_utc}
        />
      )}

      <table className="calendar-table">
        <thead>
          <tr>
            <th>Round</th>
            <th>Race</th>
            <th>Circuit</th>
            <th>Country</th>
            <th>Date</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>
          {calendar.rounds.map((round) => (
            <RaceRow key={round.round} round={round} isNext={round.round === calendar.next_round} />
          ))}
        </tbody>
      </table>
    </section>
  );
}

function RaceRow({ round, isNext }: { round: RaceMeetingDTO; isNext: boolean }) {
  const rowClass = [
    round.is_cancelled ? 'cancelled' : '',
    isNext ? 'next-race' : '',
  ]
    .filter(Boolean)
    .join(' ');

  return (
    <tr className={rowClass}>
      <td>{round.round}</td>
      <td>{round.race_name}</td>
      <td>{round.circuit_name}</td>
      <td>{round.country_name}</td>
      <td>{new Date(round.start_datetime_utc).toLocaleDateString()}</td>
      <td>
        {round.is_cancelled ? (
          <span className="badge cancelled">{round.cancelled_label ?? 'Cancelled'}</span>
        ) : (
          <span className={`badge ${round.status}`}>{round.status}</span>
        )}
      </td>
    </tr>
  );
}

function NextRaceHighlight({
  round,
  countdownTarget,
}: {
  round: RaceMeetingDTO;
  countdownTarget: string;
}) {
  const targetDate = new Date(countdownTarget);
  const now = new Date();
  const diffMs = targetDate.getTime() - now.getTime();
  const days = Math.floor(diffMs / (1000 * 60 * 60 * 24));
  const hours = Math.floor((diffMs % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));

  return (
    <div className="next-race-card">
      <h3>Next Race</h3>
      <p className="race-name">{round.race_name}</p>
      <p className="circuit">{round.circuit_name}, {round.country_name}</p>
      <p className="countdown">
        {days > 0 ? `${days}d ${hours}h` : `${hours}h`} until lights out
      </p>
    </div>
  );
}
