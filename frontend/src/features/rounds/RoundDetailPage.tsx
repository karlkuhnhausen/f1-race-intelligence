import { useEffect, useState } from 'react';
import { useParams, useSearchParams, Link } from 'react-router-dom';
import { fetchRoundDetail, type RoundDetailResponse, type SessionDetail } from './roundApi';
import SessionResultsTable from './SessionResultsTable';
import SessionTicker from './SessionTicker';
import SessionRecapStrip from './SessionRecapStrip';
import { formatLocalDateTime, isWithinWeekendWindow } from './sessionTime';

export default function RoundDetailPage() {
  const { round } = useParams<{ round: string }>();
  const [searchParams] = useSearchParams();
  const year = Number(searchParams.get('year') ?? 2026);
  const roundNum = Number(round);

  const [data, setData] = useState<RoundDetailResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!roundNum || roundNum < 1) return;
    setLoading(true);
    fetchRoundDetail(roundNum, year)
      .then(setData)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [roundNum, year]);

  if (loading) return <div className="text-muted-foreground">Loading round details…</div>;
  if (error) return <div className="text-negative">Error: {error}</div>;
  if (!data) return null;

  // Order sessions: practice → qualifying → race for upcoming/in-progress
  // rounds. For fully completed rounds, flip to date-descending so the race
  // sits at the top and FP1 at the bottom.
  const sessionOrder: Record<string, number> = {
    practice1: 1, practice2: 2, practice3: 3,
    sprint_qualifying: 4, sprint: 5,
    qualifying: 6, race: 7,
  };

  const allCompleted =
    data.sessions.length > 0 &&
    data.sessions.every((s) => s.status === 'completed');

  const weekendActive = isWithinWeekendWindow(data.sessions);

  const sortedSessions = [...data.sessions].sort((a, b) => {
    if (allCompleted) {
      return (
        new Date(b.date_start_utc).getTime() -
        new Date(a.date_start_utc).getTime()
      );
    }
    return (
      (sessionOrder[a.session_type] ?? 99) -
      (sessionOrder[b.session_type] ?? 99)
    );
  });

  return (
    <section className="space-y-6">
      <Link
        to="/calendar"
        className="inline-block font-display text-sm uppercase tracking-wider text-accent-cyan hover:text-foreground transition-colors"
      >
        ← Back to Calendar
      </Link>
      <div>
        <p className="font-display text-xs uppercase tracking-[0.2em] text-accent-red">
          Round {data.round}
        </p>
        <h2 className="mt-1 font-display text-3xl font-bold tracking-tight">
          {data.race_name}
        </h2>
        <p className="text-muted-foreground">
          {data.circuit_name} — {data.country_name}
        </p>
        {data.data_as_of_utc && (
          <p className="mt-1 text-sm text-muted-foreground font-mono">
            Data as of: {new Date(data.data_as_of_utc).toLocaleString()}
          </p>
        )}
      </div>

      {sortedSessions.length === 0 ? (
        <p className="text-muted-foreground">No session data available yet for this round.</p>
      ) : (
        <>
          <SessionRecapStrip sessions={data.sessions} />
          <div className="space-y-4">
            {sortedSessions.map((session) => (
              <SessionCard
                key={session.session_type}
                session={session}
                weekendActive={weekendActive}
              />
            ))}
          </div>
        </>
      )}
    </section>
  );
}

function SessionCard({
  session,
  weekendActive,
}: {
  session: SessionDetail;
  weekendActive: boolean;
}) {
  const statusColor =
    session.status === 'completed'
      ? 'bg-positive/20 text-positive'
      : session.status === 'in_progress'
        ? 'bg-accent-red/20 text-accent-red'
        : 'bg-surface text-muted-foreground border border-border';

  const statusLabel =
    session.status === 'completed'
      ? 'Completed'
      : session.status === 'in_progress'
        ? 'Live'
        : session.status === 'upcoming'
          ? 'Upcoming'
          : session.status;

  const showUpcomingCountdown =
    session.status === 'upcoming' && weekendActive;
  const showLiveCountdown = session.status === 'in_progress';
  const showCompletedAt = session.status === 'completed';

  return (
    <div className="rounded-lg border border-border bg-surface p-5">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h3 className="font-display text-xl font-bold tracking-tight">{session.session_name}</h3>
        <time
          dateTime={session.date_start_utc}
          title={session.date_start_utc}
          className="text-sm text-muted-foreground font-mono"
        >
          {formatLocalDateTime(session.date_start_utc)}
        </time>
      </div>
      <p className="mt-2">
        <span
          className={`inline-flex items-center rounded-md px-2 py-0.5 text-xs font-display font-bold uppercase tracking-wider ${statusColor}`}
        >
          {statusLabel}
        </span>
      </p>

      {showCompletedAt && (
        <p
          data-testid="session-completed-at"
          className="mt-2 text-sm text-muted-foreground"
        >
          Completed{' '}
          <time
            dateTime={session.date_end_utc}
            title={session.date_end_utc}
            className="font-mono"
          >
            {formatLocalDateTime(session.date_end_utc)}
          </time>
        </p>
      )}

      {showUpcomingCountdown && (
        <SessionTicker
          mode="countdown"
          targetUtc={session.date_start_utc}
          label={`Until ${session.session_name}`}
          className="mt-4"
        />
      )}

      {showLiveCountdown && (
        <SessionTicker
          mode="elapsed"
          targetUtc={session.date_start_utc}
          label={`${session.session_name} elapsed`}
          className="mt-4"
        />
      )}

      <div className="mt-4">
        {session.status === 'upcoming' ? (
          <p className="text-muted-foreground">Not yet available</p>
        ) : session.results.length > 0 ? (
          <SessionResultsTable
            results={session.results}
            sessionType={session.session_type}
          />
        ) : (
          <p className="text-muted-foreground">No results available.</p>
        )}
      </div>
    </div>
  );
}
